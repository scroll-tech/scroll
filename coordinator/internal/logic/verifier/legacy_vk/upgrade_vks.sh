#!/bin/bash

work_dir="$(dirname -- "${BASH_SOURCE[0]}")"
work_dir="$(cd -- "$work_dir" && pwd)"
echo $work_dir

rm $work_dir/*.vkey

version=release-v0.12.0
wget https://circuit-release.s3.us-west-2.amazonaws.com/${version}/vk_chunk.vkey -O $work_dir/vk_chunk.vkey
wget https://circuit-release.s3.us-west-2.amazonaws.com/${version}/vk_batch.vkey -O $work_dir/vk_batch.vkey
wget https://circuit-release.s3.us-west-2.amazonaws.com/${version}/vk_bundle.vkey -O $work_dir/vk_bundle.vkey