package util

import (
	"runtime"
	"sync"
)

type Pool struct {
	work chan Task
	sem  chan struct{} // limit goroutine
	wg   sync.WaitGroup
}

type Task struct {
	work   func(...string)
	params []string
}

// Return New Pool
func NewPool(size ...int) *Pool {
	num := runtime.NumCPU()
	if len(size) > 0 {
		num = size[0]
	}

	return &Pool{
		work: make(chan Task),
		sem:  make(chan struct{}, num),
	}
}

func (p *Pool) NewTask(task func(...string), params ...string) {
	p.wg.Add(1)

	t := Task{
		work:   task,
		params: params,
	}

	select {
	case p.work <- t:
	case p.sem <- struct{}{}:
		go p.worker(t)
	}
}

// Do Task Forever
func (p *Pool) worker(t Task) {
	defer func() { <-p.sem }()

	ok := true
	for ok {
		t.work(t.params...)
		p.wg.Done()
		t, ok = <-p.work
	}
}

func (p *Pool) Wait() {
	defer close(p.work)
	defer close(p.sem)
	p.wg.Wait()
}
