package main

/*
#cgo LDFLAGS: ./core/prover/lib/libprover.a -lm -ldl
#cgo gpu LDFLAGS: ./core/prover/lib/libprover.a -lm -ldl -lgmp -lstdc++ -lprocps -L/usr/local/cuda/lib64/ -lcudart
#include <stdlib.h>
#include "./lib/prover.h"
*/
import "C"

import (
	"fmt"
	"github.com/pkg/errors"
	"unsafe"
)

func main() {
	err := testPanic()
	fmt.Println("panic error is: ", err)
}

func testPanic() (err error) {
	tracesStr := C.CString("")
	defer func() {
		C.free(unsafe.Pointer(tracesStr))
		if r := recover(); r != nil {
			err = errors.Errorf("rust zk prove panic %d", r)
		}
	}()
	C.create_agg_proof(tracesStr)
	fmt.Println("create success")
	return
}
