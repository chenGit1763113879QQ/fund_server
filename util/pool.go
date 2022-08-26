package util

import (
	"sync/atomic"
	"time"
)

type Pool struct {
	work  chan func()   // work
	sem   chan struct{} // limit goroutine
	state int32
}

// Return New Pool
func NewPool(size int) *Pool {
	return &Pool{
		work: make(chan func()),
		sem:  make(chan struct{}, size),
	}
}

// Add Task To Pool
func (p *Pool) NewTask(task func()) {
	atomic.AddInt32(&p.state, 1)

	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.worker(task)
	}
}

// Do Task Forever
func (p *Pool) worker(task func()) {
	defer func() { <-p.sem }()

	var ok = true
	for ok {
		task()
		atomic.AddInt32(&p.state, -1)

		task, ok = <-p.work
	}
}

// Wait For Task End
func (p *Pool) Wait() {
	for {
		state := atomic.LoadInt32(&p.state)
		if state == 0 {
			return
		}
		time.Sleep(time.Microsecond)
	}
}
