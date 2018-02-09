// Package shpool implements pools for running heterogeneous SHELL processes
package shpool

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/brentp/go-athenaeum/tempclean"
	"github.com/brentp/go-athenaeum/unsplit"
	"github.com/fatih/color"
	isatty "github.com/mattn/go-isatty"
	"github.com/pkg/errors"
)

// Process is filled by the user by specifying the command and the
// number of CPUs that command will utilize.
type Process struct {
	// number of cpus used by this command.
	// This will be available as the environment variable 'CPUs' in the running process.
	CPUs int
	// the command to run in the shell.
	Command string
	// Prefix is prepended to the stderr and stdout of this command.
	// This will be available as the env var 'Prefix' in the running process.
	Prefix string
}

type process struct {
	p   Process
	c   *exec.Cmd
	err error
}

// wrap log.Logger so we can implement Write
type wlogger struct {
	mu *sync.Mutex
	*log.Logger
}

func (l wlogger) Write(b []byte) (int, error) {
	l.Logger.Print(string(b))
	return len(b), nil
}

// Pool orchestrates the work to be done.
type Pool struct {
	mu               *sync.RWMutex
	waitingProcesses []*process
	poller           chan *process
	runningCpus      int
	totalCpus        int
	workerWg         *sync.WaitGroup
	waiterWg         *sync.WaitGroup
	start            time.Time
	err              error
	logger           wlogger
	options          *Options
	ctx              context.Context
	cancel           context.CancelFunc
}

type Options struct {
	// Stop the entire pool on any error
	StopOnError bool
	// Log outputs will have this prefix
	LogPrefix string
	// don't show the running-type of each process.
	Quiet bool
}

// New creates a new pool with either the specified logger, or a logger
// with the given prefix.
func New(cpus int, logger *log.Logger, opts *Options) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{mu: &sync.RWMutex{},
		waitingProcesses: make([]*process, 0, 16),
		poller:           make(chan *process, cpus),
		workerWg:         &sync.WaitGroup{},
		waiterWg:         &sync.WaitGroup{},
		runningCpus:      0,
		totalCpus:        cpus,
		ctx:              ctx,
		cancel:           cancel,
		start:            time.Now(),
		options:          opts,
	}
	if logger == nil {
		logPrefix := strings.TrimLeft(strings.TrimSpace(opts.LogPrefix)+": ", ": ")
		p.logger = wlogger{&sync.Mutex{}, log.New(os.Stderr, logPrefix, log.Ldate|log.Ltime)}
	} else {
		p.logger = wlogger{&sync.Mutex{}, logger}
	}
	go p.poll()
	return p
}

var Shell = "/bin/bash"

func init() {
	sh := os.Getenv("SHELL")
	if sh != "" {
		Shell = sh
	}
}

// prefixer allows prefixing a lot with some prefix
type prefixer struct {
	w      wlogger
	prefix string
}

func (p *prefixer) Write(b []byte) (int, error) {
	L := len(b)
	if b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	p.w.mu.Lock()
	defer p.w.mu.Unlock()
	prefix := p.w.Prefix()
	defer p.w.SetPrefix(strings.TrimSpace(prefix))
	p.w.SetPrefix(p.w.Prefix() + "(" + p.prefix + ") ")
	sp := unsplit.New(b, []byte{'\n'})
	var n int

	for {
		line := sp.Next()
		if line == nil {
			break
		}
		nn, err := p.w.Write(line)
		n += nn + 1
		if err != nil {
			return n, err
		}
	}
	// can return len(b) here
	return L, nil
}

func init() {
	color.NoColor = !isatty.IsTerminal(os.Stderr.Fd())
}

var red = color.New(color.BgRed).Add(color.Bold).SprintfFunc()
var yellow = color.New(color.BgYellow).Add(color.Bold).SprintfFunc()

func (p *process) submit(pool *Pool) error {
	var t *os.File
	pool.workerWg.Add(1)
	if len(p.p.Command) < 8192 {
		p.c = exec.CommandContext(pool.ctx, Shell, "-c", p.p.Command)
	} else {
		// use a temp file for large files.
		var err error
		t, err = tempclean.TempFile(p.p.Prefix, ".sh")
		if err != nil {
			return errors.Wrap(err, "[shpool] error creating temp file")
		}
		if _, err := t.Write([]byte(p.p.Command)); err != nil {
			return errors.Wrap(err, "[shpool] error writing to temp file")
		}
		if err := t.Close(); err != nil {
			return errors.Wrap(err, "[shpool] error closing to temp file")
		}
		p.c = exec.Command(Shell, "-c", Shell+" "+t.Name())
	}
	p.c.Env = append(p.c.Env, fmt.Sprintf("CPUs=%d", p.p.CPUs))
	p.c.Env = append(p.c.Env, fmt.Sprintf("Prefix='%d'", p.p.Prefix))
	p.c.Stderr = &prefixer{w: pool.logger, prefix: red("[E]" + p.p.Prefix)}
	p.c.Stdout = &prefixer{w: pool.logger, prefix: yellow("[O]" + p.p.Prefix)}
	if t != nil {
		defer os.Remove(t.Name())
	}
	// todo copy gargs kill setup.
	err := p.c.Start()
	go func() {
		// wait in the background and notify the poller.
		p.err = p.c.Wait()
		if e := pool.ctx.Err(); e == nil {
			pool.poller <- p
		}
	}()
	return err
}

func (pool *Pool) checkErr(p *process) {
	if p.err == nil {
		return
	}
	pool.err = p.err

	pool.logger.Printf("error running command: %s -> %s", p.p.Command, p.err)
	if pool.options.StopOnError {
		pool.KillAll()
		close(pool.poller)
	}
}

func (pool *Pool) poll() {
	for p := range pool.poller {
		if !pool.options.Quiet {
			ut := p.c.ProcessState.UserTime()
			st := p.c.ProcessState.SystemTime()
			cmd := p.p.Command
			if len(cmd) > 100 {
				cmd = cmd[0:100]
			}
			pool.logger.Printf("finished process: %s (%s) in user-time:%s system-time:%s", p.p.Prefix, cmd, ut, st)
		}
		pool.mu.Lock()

		select {
		case <-pool.ctx.Done():
			break
		default:
		}

		pool.runningCpus -= p.p.CPUs
		pool.checkErr(p)
		pool.workerWg.Done()

		pool.sendWaiting()
		pool.mu.Unlock()
	}
}

// try to run more processes.
// must be called in a lock
func (pool *Pool) sendWaiting() {

	if len(pool.waitingProcesses) == 0 {
		return
	}

	available := pool.totalCpus - pool.runningCpus
	var used []int

	for i, w := range pool.waitingProcesses {
		if w.p.CPUs > available {
			continue
		}
		available -= w.p.CPUs
		used = append(used, i)

	}
	if len(used) != 0 {
		sort.Slice(used, func(i, j int) bool { return used[i] > used[j] })
		for _, i := range used {
			proc := pool.waitingProcesses[i]
			if err := proc.submit(pool); err != nil {
				pool.checkErr(proc)
			}
			pool.runningCpus += proc.p.CPUs
			pool.waiterWg.Done()
			pool.waitingProcesses = append(pool.waitingProcesses[:i], pool.waitingProcesses[i+1:]...)
		}
	}
}

// Wait until all processes are finished.
func (pool *Pool) Wait() error {
	pool.waiterWg.Wait()
	pool.workerWg.Wait()
	return pool.err
}

// Add a process to the pool.
func (pool *Pool) Add(p Process) {
	if p.CPUs > pool.totalCpus {
		panic("shpool: cant handle a process with more cpus than the pool")
	}
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if p.CPUs == 0 {
		p.CPUs = 1
	}
	pr := process{p: p}
	pool.waiterWg.Add(1)
	pool.waitingProcesses = append(pool.waitingProcesses, &pr)
	pool.sendWaiting()
}

// Error returns any error in the pool
func (pool *Pool) Error() error {
	return pool.err
}

// KillAll processes in the pool.
func (pool *Pool) KillAll() {
	pool.mu.Lock()
	pool.waitingProcesses = pool.waitingProcesses[:0]
	pool.cancel()
	pool.runningCpus = 0
	close(pool.poller)
	pool.waiterWg = &sync.WaitGroup{}
	pool.workerWg = &sync.WaitGroup{}
	pool.mu.Unlock()
}
