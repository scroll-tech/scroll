#!/bin/bash -e 
cp ./build/bin/prover ./gpu_prover

MALLOC_CONF=prof_leak:true,lg_prof_sample:0,prof_final:true \
LD_PRELOAD=/usr/lib/x86_64-linux-gnu/libjemalloc.so.2 ./gpu_prover
