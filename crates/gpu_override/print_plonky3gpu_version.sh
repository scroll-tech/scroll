#!/bin/bash

higher_plonky3_item=`grep "plonky3-gpu" ./Cargo.lock | sort | uniq | awk -F "[#=]" '{print $3" "$4}' | sort -k 1 | tail -n 1`

higher_version=`echo $higher_plonky3_item | awk '{print $1}'`

higher_commit=`echo $higher_plonky3_item | cut -d ' ' -f2 | cut -c-7`

echo "$higher_version"
echo "$higher_commit"