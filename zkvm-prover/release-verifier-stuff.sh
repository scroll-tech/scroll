#!/bin/bash

# release version
SCROLL_ZKVM_STUFFDIR ?= `realpath .work`
SCROLL_ZKVM_VERSION ?= 0.5.0rc1
DIR_OUTPUT="releases/${SCROLL_ZKVM_VERSION}/verifier"

STUFF_FILES=('root-verifier-committed-exe' 'root-verifier-vm-config' 'verifier.bin' 'openVmVk.json')

for stuff_file in "${STUFF_FILES[@]}"; do
  SRC="${SCROLL_ZKVM_STUFFDIR}/${stuff_file}"
  TARGET="${DIR_OUTPUT}/${stuff_file}"
  aws --profile default s3 cp $SRC s3://circuit-release/scroll-zkvm/$TARGET
done
