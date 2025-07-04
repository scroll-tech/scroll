#!/bin/bash
set -uex

PLONKY3_GPU_COMMIT=450ec18        # feynman
OPENVM_STARK_GPU_COMMIT=e3b2d6    # branch: sync/upstream-250702
OPENVM_GPU_COMMIT=75df915    # branch: patch-v1.2.1-rc.1-pipe

DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)

# checkout plonky3-gpu
cd $DIR/plonky3-gpu && git checkout ${PLONKY3_GPU_COMMIT}

# checkout openvm-stark-gpu
cd $DIR/openvm-stark-gpu && git checkout ${OPENVM_STARK_GPU_COMMIT}

# checkout openvm-gpu
cd $DIR/openvm-gpu && git checkout ${OPENVM_GPU_COMMIT}
