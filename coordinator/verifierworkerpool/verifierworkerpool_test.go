package verifierworkerpool_test

import (
	"scroll-tech/coordinator/verifierworkerpool"

	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVerifierWorkerPool(t *testing.T) {
	as := assert.New(t)

	vwp := verifierworkerpool.NewVerifierWorkerPool(2)
	vwp.Run()
	var cnt int32 = 3

	task1 := func() {
		time.Sleep(500 * time.Millisecond)
		atomic.AddInt32(&cnt, -1)
	}

	task2 := func() {
		time.Sleep(1 * time.Second)
		atomic.AddInt32(&cnt, -1)
	}

	vwp.AddTask(task1)
	vwp.AddTask(task1)
	vwp.AddTask(task2)

	time.Sleep(600 * time.Millisecond)
	as.Equal(int32(1), atomic.LoadInt32(&cnt))
	vwp.Stop()
	as.Equal(int32(0), atomic.LoadInt32(&cnt))

}
