#!/bin/bash
set -uex

PLONKY3_GPU_COMMIT=261b322        # v0.2.0
OPENVM_STARK_GPU_COMMIT=d91dbbc   # PR#47
OPENVM_GPU_COMMIT=ce88ec8         # branch: patch-v1.2.0

DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)

# checkout plonky3-gpu
if [ ! -d $DIR/plonky3-gpu ]; then
    git clone git@github.com:scroll-tech/plonky3-gpu.git $DIR/plonky3-gpu
fi
cd $DIR/plonky3-gpu && git fetch && git checkout ${PLONKY3_GPU_COMMIT}

# checkout openvm-stark-gpu
if [ ! -d $DIR/openvm-stark-gpu ]; then
    git clone git@github.com:scroll-tech/openvm-stark-gpu.git $DIR/openvm-stark-gpu
fi
cd $DIR/openvm-stark-gpu && git fetch && git checkout ${OPENVM_STARK_GPU_COMMIT}

# checkout openvm-gpu
if [ ! -d $DIR/openvm-gpu ]; then
    git clone git@github.com:scroll-tech/openvm-gpu.git $DIR/openvm-gpu
fi
cd $DIR/openvm-gpu && git fetch && git checkout ${OPENVM_GPU_COMMIT}
