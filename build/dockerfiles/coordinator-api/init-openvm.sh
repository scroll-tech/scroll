#!/bin/bash
set -uex

OPENVM_GPU_COMMIT=dfa10b4

DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)

# checkout openvm-gpu
if [ ! -d $DIR/openvm-gpu ]; then
    git clone git@github.com:scroll-tech/openvm-gpu.git $DIR/openvm-gpu
fi
cd $DIR/openvm-gpu && git fetch && git checkout ${OPENVM_GPU_COMMIT}
