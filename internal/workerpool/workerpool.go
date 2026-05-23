package workerpool

import (
	"sync"
)

type Worker struct {
	id             int
	taskChan       chan func()
	completedTasks int64
	pool           *Pool
}

type Pool struct {
	workers    []*Worker
	maxWorkers int
	taskChan   chan func()
	waitGroup  sync.WaitGroup
	notifyCh   chan struct{}
	taskCount  int64
	mu         sync.Mutex
	errors     []error
}

func NewPool(maxWorkers int) *Pool {
	return &Pool{
		maxWorkers: maxWorkers,
		workers:    make([]*Worker, 0),
		taskChan:   make(chan func()),
		waitGroup:  sync.WaitGroup{},
		notifyCh:   make(chan struct{}),
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.maxWorkers; i++ {
		worker := &Worker{id: i, taskChan: p.taskChan}
		p.workers = append(p.workers, worker)
		go worker.Start()
	}
}

func (w *Worker) Start() {
	go func() {
		for task := range w.taskChan {
			w.pool.Wait()
			task()
		}
	}()
}

func (p *Pool) AddTask(task func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.taskChan == nil {
		return
	}
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
	for range p.workers {
		<-p.taskChan
	}
}
