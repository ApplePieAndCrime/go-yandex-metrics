package workerpool

import (
	"sync"
)

type Pool struct {
	maxWorkers int
	taskChan   chan func()
	wg         sync.WaitGroup
	mu         sync.Mutex
	errors     []error
}

func NewPool(maxWorkers int) *Pool {
	return &Pool{
		maxWorkers: maxWorkers,
		taskChan:   make(chan func()),
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for task := range p.taskChan {
		task()
	}
}

func (p *Pool) AddTask(task func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.taskChan <- task
}

func (p *Pool) AddError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errors = append(p.errors, err)
}

func (p *Pool) Errors() []error {
	p.mu.Lock()
	defer p.mu.Unlock()
	errCopy := make([]error, len(p.errors))
	copy(errCopy, p.errors)
	return errCopy
}

func (p *Pool) Close() {
	close(p.taskChan)
}

func (p *Pool) Wait() {
	p.wg.Wait()
}
