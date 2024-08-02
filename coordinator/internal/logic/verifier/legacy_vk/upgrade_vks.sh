#!/bin/bash

work_dir="$(dirname -- "${BASH_SOURCE[0]}")"
work_dir="$(cd -- "$work_dir" && pwd)"
echo $work_dir

rm $work_dir/*.vkey

version=release-v0.11.4
wget https://circuit-release.s3.us-west-2.amazonaws.com/${version}/chunk_vk.vkey -O $work_dir/chunk_vk.vkey
wget https://circuit-release.s3.us-west-2.amazonaws.com/${version}/agg_vk.vkey -O $work_dir/agg_vk.vkey