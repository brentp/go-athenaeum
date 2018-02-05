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
)

// Process is filled by the user by specifying the command and the
// number of CPUs that command will utilize.
type Process struct {
	CPUs    int
	Command string
}

type process struct {
	p     Process
	c     *exec.Cmd
	start time.Time
}

// Pool orchestrates the work to be done.
type Pool struct {
	mu               *sync.RWMutex
	runningProcesses []*process
	waitingProcesses []*process
	runningCpus      int
	totalCpus        int
}

func New(cpus int) *Pool {
	p := &Pool{mu: &sync.RWMutex{},
		runningProcesses: make([]*process, 0, 16),
		waitingProcesses: make([]*process, 0, 16),
		runningCpus:      0, totalCpus: cpus}
	// this leaks a goroutine for every New().
	go func() {
		ticker := time.NewTicker(time.Second * 5)
		for _ = range ticker.C {
			p.check()
		}
	}()
	return p
}

var SHELL = "/bin/bash"

func init() {
	sh := os.Getenv("SHELL")
	if sh != "" {
		SHELL = sh
	}
}

func (p *process) submit() error {
	p.c = exec.Command(SHELL, "-c", p.p.Command)
	p.c.Env = append(p.c.Env, fmt.Sprintf("CPUS=%d", p.p.CPUs))
	p.c.Stderr = os.Stderr
	p.c.Stdout = os.Stdout
	// todo copy gargs kill setup.
	p.start = time.Now()
	return p.c.Start()
}

// checkRunning clears values from runningProcesses and returns
// an int indicating available capacity in the pool.
func (p *Pool) checkRunning() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.runningProcesses) == 0 {
		return p.totalCpus - p.runningCpus
	}
	p.mu.RUnlock()
	p.mu.Lock()
	defer p.mu.Unlock()

	var used []int

	for i, proc := range p.runningProcesses {
		if proc.finished() {
			if err := proc.c.Wait(); err != nil {
				log.Println(err)
			}
			used = append(used, i)
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
	defer p.mu.RUnlock()

	if p.totalCpus-p.runningCpus == 0 {
		return false
	}

	if len(p.waitingProcesses) == 0 && len(p.runningProcesses) == 0 {
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
			proc.submit()
			p.runningProcesses = append(p.runningProcesses, proc)
			p.waitingProcesses = append(p.waitingProcesses[:i], p.waitingProcesses[i+1:]...)
		}
	}
	return false

}

// check returns true if there are no processes running or waiting.
func (p *Pool) check() bool {
	if available := p.checkRunning(); available != 0 {
		return p.checkWaiting()
	}
	return false
}

func (pool *Pool) Add(p Process) {
	pool.mu.Lock()
	pr := process{p: p}
	pool.waitingProcesses = append(pool.waitingProcesses, &pr)
}

func (p *process) finished() bool {
	err := p.c.Process.Signal(syscall.Signal(0))
	if err == nil {
		panic("process not found")
	}
	return strings.Contains(strings.ToLower(err.Error()), "already finished")
}
