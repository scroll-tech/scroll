#!/bin/bash

config_file=.cargo/config.toml
plonky3_gpu_path=$(grep 'path.*plonky3-gpu' "$config_file" | cut -d'"' -f2 | head -n 1)
plonky3_gpu_path=$(dirname "$plonky3_gpu_path")

if [ -z $plonky3_gpu_path ]; then
    exit 0
else
    pushd $plonky3_gpu_path
    commit_hash=$(git log --pretty=format:%h -n 1)
    echo "${commit_hash:0:7}"

    popd
fi