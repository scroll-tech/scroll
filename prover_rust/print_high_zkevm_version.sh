#!/bin/bash
set -ue

higher_zkevm_item=`grep "zkevm-circuits.git" ./Cargo.lock | sort | uniq | awk -F "[#=]" '{print $3" "$4}' | sort -k 1 | tail -n 1`

higher_version=`echo $higher_zkevm_item | awk '{print $1}'`

higher_commit=`echo $higher_zkevm_item | cut -d ' ' -f2 | cut -c-7`

echo "$higher_version $higher_commit"