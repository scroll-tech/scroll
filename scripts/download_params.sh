#!/bin/bash
set -uex

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
PROJ_DIR=$DIR"/.."

mkdir -p $PROJ_DIR/assets/params
wget https://circuit-release.s3.us-west-2.amazonaws.com/setup/params19 -O $PROJ_DIR/assets/params/params19
wget https://circuit-release.s3.us-west-2.amazonaws.com/setup/params20 -O $PROJ_DIR/assets/params/params20
wget https://circuit-release.s3.us-west-2.amazonaws.com/setup/params24 -O $PROJ_DIR/assets/params/params24
wget https://circuit-release.s3.us-west-2.amazonaws.com/setup/params25 -O $PROJ_DIR/assets/params/params25