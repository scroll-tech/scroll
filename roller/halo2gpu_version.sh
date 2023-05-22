#!/bin/bash

config_file="~/.cargo/config"

if [[ $(head -n 1 "$config_file") == "#"* ]]; then
  exit 0
fi

halo2gpu_path=$(grep -Po '(?<=paths = \[")([^"]*)' $config_file)

pushd $halo2gpu_path

commit_hash=$(git log --pretty=format:%h -n 1)
echo "${commit_hash:0:7}"

popd
Ï
