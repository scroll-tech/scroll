#!/bin/bash
set -ue

higher_zkvm_item=`grep "zkvm-prover" ../Cargo.lock | sort | uniq | awk -F "[#=]" '{print $3" "$4}' | sort -k 1 | tail -n 1`

higher_version=`echo $higher_zkvm_item | awk '{print $1}'`

higher_commit=`echo $higher_zkvm_item | cut -d ' ' -f2 | cut -c-7`

echo "$higher_version $higher_commit"