// Package shpool implements pools for running heterogeneous SHELL processes
package shpool

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"syscall"
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
	p     Process
	c     *exec.Cmd
	start time.Time
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
	runningProcesses []*process
	waitingProcesses []*process
	runningCpus      int
	totalCpus        int
	start            time.Time
	err              error
	logger           wlogger
	options          *Options
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
	p := &Pool{mu: &sync.RWMutex{},
		runningProcesses: make([]*process, 0, 16),
		waitingProcesses: make([]*process, 0, 16),
		runningCpus:      0, totalCpus: cpus,
		start:   time.Now(),
		options: opts,
	}
	if logger == nil {
		logPrefix := strings.TrimLeft(strings.TrimSpace(opts.LogPrefix)+": ", ": ")
		p.logger = wlogger{&sync.Mutex{}, log.New(os.Stderr, logPrefix, log.Ldate|log.Ltime)}
	} else {
		p.logger = wlogger{&sync.Mutex{}, logger}
	}
	// this leaks a goroutine for every New().
	go func() {
		ticker := time.NewTicker(time.Second * 5)
		for _ = range ticker.C {
			p.check()
		}
	}()
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
	if len(p.p.Command) < 8192 {
		p.c = exec.Command(Shell, "-c", p.p.Command)
	} else {
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
	p.start = time.Now()
	v := p.c.Start()
	log.Println("starting:", p.c.ProcessState)
	return v
}

// checkRunning clears values from runningProcesses and returns
// an int indicating available capacity in the pool.
func (p *Pool) checkRunning() int {
	p.mu.RLock()
	if len(p.runningProcesses) == 0 {
		p.mu.RUnlock()
		return p.totalCpus - p.runningCpus
	}
	p.mu.RUnlock()
	p.mu.Lock()
	defer p.mu.Unlock()

	var used []int

	for i, proc := range p.runningProcesses {
		if proc.finished() {
			if !p.options.Quiet {
				rt := time.Since(proc.start)
				cmd := proc.p.Command
				if len(cmd) > 100 {
					cmd = cmd[0:100]
				}
				p.logger.Printf("finished process: %s (%s) in %s", proc.p.Prefix, cmd, rt)
			}
			if err := proc.c.Wait(); err != nil {
				p.logger.Printf("error running command: %s -> %s", proc.p.Command, err)
				p.err = err
				if p.options.StopOnError {
					p.KillAll()
					return -1
				}
				if err := proc.c.Wait(); err != nil {
					log.Println(err)
				}
				used = append(used, i)
			}
		}
	}
	if len(used) != 0 {
		sort.Reverse(sort.IntSlice(used))
		for _, i := range used {
			proc := p.runningProcesses[i]
			// TODO: report total time running here.
			p.runningCpus -= proc.p.CPUs
			p.runningProcesses = append(p.runningProcesses[:i], p.runningProcesses[i+1:]...)
		}

	}
	return p.totalCpus - p.runningCpus
}

// returns true if there are no waiting or running processes
func (p *Pool) checkWaiting() bool {
	p.mu.RLock()

	if p.totalCpus-p.runningCpus == 0 {
		p.mu.RUnlock()
		return false
	}

	if len(p.waitingProcesses) == 0 && len(p.runningProcesses) == 0 {
		p.mu.RUnlock()
		return true
	}

	p.mu.RUnlock()
	p.mu.Lock()
	defer p.mu.Unlock()

	available := p.totalCpus - p.runningCpus
	var used []int

	for i, w := range p.waitingProcesses {
		if w.p.CPUs > available {
			continue
		}
		available -= w.p.CPUs
		used = append(used, i)

	}
	if len(used) != 0 {
		sort.Reverse(sort.IntSlice(used))
		for _, i := range used {
			proc := p.waitingProcesses[i]
			if err := proc.submit(p); err != nil {
				p.err = err
			}
			p.runningProcesses = append(p.runningProcesses, proc)
			p.waitingProcesses = append(p.waitingProcesses[:i], p.waitingProcesses[i+1:]...)
		}
	}
	return false

}

// check returns true if there are no processes running or waiting.
func (p *Pool) check() bool {
	if available := p.checkRunning(); available > 0 {
		return p.checkWaiting()
	}
	return false
}

// Wait until all processes are finished.
func (pool *Pool) Wait() error {
	if pool.check() {
		pool.logger.Printf("finished all processes after: %s seconds", time.Now().Sub(pool.start))
		return pool.err
	}

	ticker := time.NewTicker(time.Second * 3)
	for t := range ticker.C {
		if pool.check() {
			pool.logger.Printf("finished all processes after: %s seconds", t.Sub(pool.start))
			ticker.Stop()
			return pool.err
		}
	}
	return nil
}

// Add a process to the pool.
func (pool *Pool) Add(p Process) {
	pool.mu.Lock()
	if p.CPUs == 0 {
		p.CPUs = 1
	}
	pr := process{p: p}
	pool.waitingProcesses = append(pool.waitingProcesses, &pr)
	pool.mu.Unlock()
	pool.check()
}

// Error returns any error in the pool
func (pool *Pool) Error() error {
	return pool.err
}

// KillAll processes in the pool.
func (pool *Pool) KillAll() {
	pool.mu.Lock()
	pool.waitingProcesses = pool.waitingProcesses[:0]

	for _, proc := range pool.runningProcesses {
		pool.logger.Print(proc.c.Process.Kill())
	}

	pool.mu.Unlock()
}

func (p *process) finished() bool {
	log.Println(p.c.ProcessState)

	proc, err := os.FindProcess(int(p.c.Process.Pid))
	//log.Println(proc, err)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	//log.Println(err)
	if err == nil {
		return false
	}
	log.Println(err)
	return true
}
