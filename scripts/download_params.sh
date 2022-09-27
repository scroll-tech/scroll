#!/bin/bash
set -uex

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
PROJ_DIR=$DIR"/.."

mkdir -p $PROJ_DIR/assets/params
wget https://circuit-release.s3.us-west-2.amazonaws.com/circuit-release/release-0804-degree25/test_seed $PROJ_DIR/assets/seed
wget https://circuit-release.s3.us-west-2.amazonaws.com/circuit-release/release-0804-degree25/test_params/params18 -O $PROJ_DIR/assets/params/params18
wget https://circuit-release.s3.us-west-2.amazonaws.com/circuit-release/release-0804-degree25/test_params/params25 -O $PROJ_DIR/assets/params/params25
