package util

import (
	"sync"
)

type Pool struct {
	work chan func()   // work
	sem  chan struct{} // limit goroutine
	wg   sync.WaitGroup
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
	p.wg.Add(1)

	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.worker(task)
	}
}

// Do Task Forever
func (p *Pool) worker(task func()) {
	defer func() { <-p.sem }()

	for {
		task()
		p.wg.Done()
		task = <-p.work
	}
}

// Wait For Task End
func (p *Pool) Wait() {
	defer close(p.work)
	defer close(p.sem)
	p.wg.Wait()
}
