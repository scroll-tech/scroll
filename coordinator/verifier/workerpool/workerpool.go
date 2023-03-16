package workerpool

import (
	"sync"
)

// WorkerPool is responsible for creating workers and managing verify proof task between them
type WorkerPool struct {
	maxWorker     int
	taskQueueChan chan func()
	wg            sync.WaitGroup
}

// NewWorkerPool creates new worker pool with given amount of workers
func NewWorkerPool(maxWorker int) *WorkerPool {
	return &WorkerPool{
		maxWorker:     maxWorker,
		taskQueueChan: nil,
		wg:            sync.WaitGroup{},
	}
}

// Run runs WorkerPool
func (vwp *WorkerPool) Run() {
	vwp.taskQueueChan = make(chan func())
	for i := 0; i < vwp.maxWorker; i++ {
		go func() {
			for task := range vwp.taskQueueChan {
				if task != nil {
					task()
					vwp.wg.Done()
				} else {
					return
				}
			}
		}()
	}
}

// Stop stop WorkerPool
func (vwp *WorkerPool) Stop() {
	vwp.wg.Wait()
	// close task queue channel, so that all goruotines listening from it stop
	close(vwp.taskQueueChan)
}

// AddTask adds a task to WorkerPool
func (vwp *WorkerPool) AddTask(task func()) {
	vwp.wg.Add(1)
	vwp.taskQueueChan <- task
}
