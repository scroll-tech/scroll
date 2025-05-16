#!/bin/bash
set -uex

DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)

# clone openvm-gpu if not exists
if [ ! -d $DIR/openvm-gpu ]; then
    git clone git@github.com:scroll-tech/openvm-gpu.git $DIR/openvm-gpu
fi
cd $DIR/openvm-gpu && git fetch --all --force
