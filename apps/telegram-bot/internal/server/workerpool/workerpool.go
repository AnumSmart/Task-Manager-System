package workerpool

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
)

var (
	ErrAlreadyRunning = errors.New("workerpool: already running")
	ErrNotRunning     = errors.New("workerpool: not running")
	ErrTaskQueueFull  = errors.New("workerpool: task queue is full")
)

// Task - интерфейс для задачи, которую может обработать воркер
type Task interface {
	Process(ctx context.Context) error
}

// WorkerPool - пул воркеров для обработки задач
type WorkerPool struct {
	taskChan   chan Task
	errChan    chan error
	numWorkers int
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	isRunning  atomic.Bool
	mu         sync.RWMutex

	errorHandler func(error)
}

// NewWorkerPool - конструктор
func NewWorkerPool(numWorkers int, taskBuffer int, errBuffer int) *WorkerPool {
	return &WorkerPool{
		taskChan:   make(chan Task, taskBuffer),
		errChan:    make(chan error, errBuffer),
		numWorkers: numWorkers,
	}
}

// Start - запуск воркеров
func (p *WorkerPool) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning.Load() {
		return ErrAlreadyRunning
	}

	// создаём контекст с отменой для остановуи воркеров
	p.ctx, p.cancel = context.WithCancel(ctx)

	// Запускаем воркеров
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	// Запускаем обработчик ошибок
	p.wg.Add(1)
	go p.errorProcessor()

	p.isRunning.Store(true)

	log.Printf("✅ WorkerPool started with %d workers", p.numWorkers)

	return nil
}

// Submit - отправка задачи в пул
func (p *WorkerPool) Submit(task Task) error {
	if !p.isRunning.Load() {
		return ErrNotRunning
	}

	select {
	case <-p.ctx.Done():
		return context.Canceled
	case p.taskChan <- task:
		return nil
	default:
		// Канал задач переполнен
		return ErrTaskQueueFull
	}
}

// worker - обработчик задач
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	log.Printf("👷 Worker %d started", id)

	for {
		select {
		case <-p.ctx.Done():
			log.Printf("👷 Worker %d stopping", id)
			return
		case task := <-p.taskChan:
			// Обрабатываем задачу
			if err := task.Process(p.ctx); err != nil {
				// Отправляем ошибку в канал
				select {
				case p.errChan <- err:
				case <-p.ctx.Done():
					return
				default:
					// Канал ошибок переполнен
					if p.errorHandler != nil {
						p.errorHandler(err)
					} else {
						log.Printf("⚠️ Worker %d error: %v", id, err)
					}
				}
			}
		}
	}
}

// errorProcessor - обработчик ошибок из канала
func (p *WorkerPool) errorProcessor() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case err := <-p.errChan:
			if p.errorHandler != nil {
				p.errorHandler(err)
			} else {
				log.Printf("⚠️ WorkerPool error: %v", err)
			}
		}
	}
}

// Stop - остановка пула
func (p *WorkerPool) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning.Load() {
		return nil
	}

	log.Println("⏹️ Stopping worker pool...")

	// Отменяем контекст
	if p.cancel != nil {
		p.cancel()
	}

	// Ждем завершения с таймаутом
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("✅ Worker pool stopped")
	case <-ctx.Done():
		log.Println("⚠️ Context cancelled while stopping worker pool")
	}

	p.isRunning.Store(false)
	return nil
}

// Stats - статистика пула (опционально)
func (p *WorkerPool) Stats() (queued int, workers int, isRunning bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.taskChan), p.numWorkers, p.isRunning.Load()
}

// SetErrorHandler - установка обработчика ошибок
func (p *WorkerPool) SetErrorHandler(handler func(error)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorHandler = handler
}
