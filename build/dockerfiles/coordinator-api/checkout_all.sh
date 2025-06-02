#!/bin/bash
set -uex

PLONKY3_GPU_COMMIT=261b322        # v0.2.0
OPENVM_STARK_GPU_COMMIT=3082234   # PR#48
OPENVM_GPU_COMMIT=8094b4f         # branch: patch-v1.2.0

DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)

# checkout plonky3-gpu
cd $DIR/plonky3-gpu && git checkout ${PLONKY3_GPU_COMMIT}

# checkout openvm-stark-gpu
cd $DIR/openvm-stark-gpu && git checkout ${OPENVM_STARK_GPU_COMMIT}

# checkout openvm-gpu
cd $DIR/openvm-gpu && git checkout ${OPENVM_GPU_COMMIT}
