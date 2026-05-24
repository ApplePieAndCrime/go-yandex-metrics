package workerpool

import (
	"sync"
)

type worker struct {
	id       int
	taskChan <-chan func()
	wg       *sync.WaitGroup
}

type Pool struct {
	workers    []*worker
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
		workers:    make([]*worker, 0),
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.maxWorkers; i++ {
		w := &worker{
			id:       i,
			taskChan: p.taskChan,
			wg:       &p.wg,
		}
		w.wg.Add(1)
		p.workers = append(p.workers, w)
		go w.Start()
	}
}

func (w *worker) Start() {
	go func() {
		defer w.wg.Done()
		for task := range w.taskChan {
			task()
		}
	}()
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
	return p.errors
}

func (p *Pool) Wait() {
	close(p.taskChan)
	p.wg.Wait()
}
