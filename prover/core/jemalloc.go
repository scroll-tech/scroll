package core

/*
// This cgo directive is what actually causes jemalloc to be linked in to the
// final Go executable
#cgo pkg-config: jemalloc

#include <jemalloc/jemalloc.h>

void _refresh_jemalloc_stats() {
	// You just need to pass something not-null into the "epoch" mallctl.
	size_t random_something = 1;
	mallctl("epoch", NULL, NULL, &random_something, sizeof(random_something));
}
int _get_jemalloc_active() {
	size_t stat, stat_size;
	stat = 0;
	stat_size = sizeof(stat);
	mallctl("stats.active", &stat, &stat_size, NULL, 0);
	return (int)stat;
}
*/
import "C"

import (
	"expvar"
	"sync"
)

func init() {
	var refreshLock sync.Mutex
	expvar.Publish("jemalloc_allocated", expvar.Func(func() interface{} {
		refreshLock.Lock()
		defer refreshLock.Unlock()
		C._refresh_jemalloc_stats()
		return C._get_jemalloc_active()
	}))
}
