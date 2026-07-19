package workerpool

import (
	"sync"
)

// Pool ограничивает число одновременно выполняемых задач.
type Pool struct {
	maxWorkers int
	taskChan   chan func()
	wg         sync.WaitGroup
	errMu      sync.Mutex
	errors     []error
}

// NewPool создаёт пул с указанным числом обработчиков.
func NewPool(maxWorkers int) *Pool {
	return &Pool{
		maxWorkers: maxWorkers,
		taskChan:   make(chan func(), maxWorkers),
	}
}

// Start запускает обработчики задач.
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

// AddTask добавляет задачу в очередь на выполнение.
func (p *Pool) AddTask(task func()) {
	p.taskChan <- task
}

// AddError сохраняет ошибку, возникшую при выполнении задачи.
func (p *Pool) AddError(err error) {
	p.errMu.Lock()
	defer p.errMu.Unlock()

	p.errors = append(p.errors, err)
}

// Errors возвращает копию накопленных ошибок.
func (p *Pool) Errors() []error {
	p.errMu.Lock()
	defer p.errMu.Unlock()

	errCopy := make([]error, len(p.errors))
	copy(errCopy, p.errors)

	return errCopy
}

// Close закрывает очередь задач.
func (p *Pool) Close() {
	close(p.taskChan)
}

// Wait ожидает завершения всех обработчиков.
func (p *Pool) Wait() {
	p.wg.Wait()
}
