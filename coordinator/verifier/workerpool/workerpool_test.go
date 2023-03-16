package workerpool_test

import (
	"scroll-tech/coordinator/verifier/workerpool"

	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkerPool(t *testing.T) {
	as := assert.New(t)

	vwp := workerpool.NewWorkerPool(2)
	vwp.Run()
	var cnt int32 = 3

	task := func() {
		time.Sleep(500 * time.Millisecond)
		atomic.AddInt32(&cnt, -1)
	}

	go vwp.AddTask(task)
	go vwp.AddTask(task)
	go vwp.AddTask(task)

	time.Sleep(600 * time.Millisecond)
	as.Equal(int32(1), atomic.LoadInt32(&cnt))
	vwp.Stop()
	as.Equal(int32(0), atomic.LoadInt32(&cnt))

}
