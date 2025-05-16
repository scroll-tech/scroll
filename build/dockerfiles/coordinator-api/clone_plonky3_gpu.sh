#!/bin/bash
set -uex

DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)

# clone plonky3-gpu if not exists
if [ ! -d $DIR/plonky3-gpu ]; then
    git clone git@github.com:scroll-tech/plonky3-gpu.git $DIR/plonky3-gpu
fi
cd $DIR/plonky3-gpu && git fetch --all --force
