#!/bin/bash

config_file="~/.cargo/config.toml"  # 替换为你的文件路径

# 使用 grep 和 awk 提取路径
plonky3_gpu_path=$(grep -oP 'path\s*=\s*"\K/plonky3-gpu[^"]*' "$config_file")

if [ -d $plonky3_gpu_path ]; then
    pushd $plonky3_gpu_path

    commit_hash=$(git log --pretty=format:%h -n 1)
    echo "${commit_hash:0:7}"

    popd
else
    exit 0
fi