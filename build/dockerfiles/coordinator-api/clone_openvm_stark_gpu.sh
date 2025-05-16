#!/bin/bash
set -uex

DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)

# clone openvm-stark-gpu if not exists
if [ ! -d $DIR/openvm-stark-gpu ]; then
    git clone git@github.com:scroll-tech/openvm-stark-gpu.git $DIR/openvm-stark-gpu
fi
cd $DIR/openvm-stark-gpu && git fetch --all --force
