package workerpool

import (
	"sync"
)

type Worker struct {
	id       int
	taskChan <-chan func()
	wg       *sync.WaitGroup
}

type Pool struct {
	workers    []*Worker
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
		workers:    make([]*Worker, 0),
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.maxWorkers; i++ {
		w := &Worker{
			id:       i,
			taskChan: p.taskChan,
			wg:       &p.wg,
		}
		p.workers = append(p.workers, w)
		go w.Start()
	}
}

func (w *Worker) Start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for task := range w.taskChan {
			task()
		}
	}()
}

func (p *Pool) AddTask(task func()) {
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
