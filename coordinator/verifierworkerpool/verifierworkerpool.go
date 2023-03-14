package verifierworkerpool

import (
	"sync"
)

// VerifierWorkerPool is responsible for creating workers and managing verify proof task between them
type VerifierWorkerPool struct {
	maxWorker     int
	taskQueueChan chan func()
	wg            sync.WaitGroup
}

// NewVerifierWorkerPool creates new worker pool with given amount of workers
func NewVerifierWorkerPool(maxWorker int) *VerifierWorkerPool {
	return &VerifierWorkerPool{
		maxWorker:     maxWorker,
		taskQueueChan: nil,
		wg:            sync.WaitGroup{},
	}
}

// Run runs VerifierWorkerPool
func (vwp *VerifierWorkerPool) Run() {
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

// Stop stop VerifierWorkerPool
func (vwp *VerifierWorkerPool) Stop() {
	vwp.wg.Wait()
	// close task queue channel, so that all goruotines listening from it stop
	close(vwp.taskQueueChan)
}

// AddTask adds a task to VerifierWorkerPool
func (vwp *VerifierWorkerPool) AddTask(task func()) {
	vwp.wg.Add(1)
	vwp.taskQueueChan <- task
}
