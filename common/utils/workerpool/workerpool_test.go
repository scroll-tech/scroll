package workerpool_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/utils/workerpool"
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

	vwp.AddTask(task)
	vwp.AddTask(task)

	time.Sleep(600 * time.Millisecond)
	as.Equal(int32(1), atomic.LoadInt32(&cnt))
	vwp.AddTask(task)
	vwp.Stop()
	as.Equal(int32(0), atomic.LoadInt32(&cnt))
}

func TestWorkerPoolMaxWorkers(t *testing.T) {
	as := assert.New(t)

	vwp := workerpool.NewWorkerPool(2)
	vwp.Run()
	var cnt int32 = 3

	task := func() {
		time.Sleep(500 * time.Millisecond)
		atomic.AddInt32(&cnt, -1)
	}

	time1 := time.Now()
	vwp.AddTask(task)
	vwp.AddTask(task)
	vwp.AddTask(task)
	vwp.Stop()
	time2 := time.Now()
	as.Greater(time2.Sub(time1), time.Second*1)

}

func TestWorkerPoolStopAndStart(t *testing.T) {
	as := assert.New(t)
	vwp := workerpool.NewWorkerPool(1)
	var cnt int32 = 3

	task := func() {
		time.Sleep(500 * time.Millisecond)
		atomic.AddInt32(&cnt, -1)
	}

	vwp.Run()
	vwp.AddTask(task)
	vwp.AddTask(task)
	vwp.Stop()
	as.Equal(int32(1), atomic.LoadInt32(&cnt))

	vwp.Run()
	vwp.AddTask(task)
	vwp.Stop()
	as.Equal(int32(0), atomic.LoadInt32(&cnt))

}
