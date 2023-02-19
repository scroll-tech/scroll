#!/bin/bash
set -uex

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
PROJ_DIR=$DIR"/.."

mkdir -p $PROJ_DIR/assets/params
wget https://circuit-release.s3.us-west-2.amazonaws.com/circuit-release/release-1220/test_seed -O $PROJ_DIR/assets/seed
wget https://circuit-release.s3.us-west-2.amazonaws.com/circuit-release/release-1220/test_params/params19 -O $PROJ_DIR/assets/params/params19
wget https://circuit-release.s3.us-west-2.amazonaws.com/circuit-release/release-1220/test_params/params26 -O $PROJ_DIR/assets/params/params26
