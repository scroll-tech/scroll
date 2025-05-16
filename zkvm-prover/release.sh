#!/bin/bash

# release version
if [ -z "${SCROLL_ZKVM_VERSION}" ]; then
  echo "SCROLL_ZKVM_VERSION not set"
  exit 1
else
  DIR_OUTPUT="releases/${SCROLL_ZKVM_VERSION}"
fi

# directory to read assets from
if [ -z "${SCROLL_ZKVM_TESTRUN_DIR}" ]; then
  echo "SCROLL_ZKVM_TESTRUN_DIR not set"
  exit 1
else
  DIR_INPUT="${SCROLL_ZKVM_TESTRUN_DIR}"
fi

# create all required directories for release
mkdir -p $DIR_OUTPUT/chunk
mkdir -p $DIR_OUTPUT/batch
mkdir -p $DIR_OUTPUT/bundle
mkdir -p $DIR_OUTPUT/verifier

# copy chunk-program related assets
cp ./crates/circuits/chunk-circuit/openvm/app.vmexe $DIR_OUTPUT/chunk/app.vmexe
cp ./crates/circuits/chunk-circuit/openvm.toml $DIR_OUTPUT/chunk/openvm.toml

# copy batch-program related assets
cp ./crates/circuits/batch-circuit/openvm/app.vmexe $DIR_OUTPUT/batch/app.vmexe
cp ./crates/circuits/batch-circuit/openvm.toml $DIR_OUTPUT/batch/openvm.toml

# copy bundle-program related assets
cp ./crates/circuits/bundle-circuit/openvm/app.vmexe $DIR_OUTPUT/bundle/app.vmexe
cp ./crates/circuits/bundle-circuit/openvm/app_euclidv1.vmexe $DIR_OUTPUT/bundle/app_euclidv1.vmexe
cp ./crates/circuits/bundle-circuit/openvm.toml $DIR_OUTPUT/bundle/openvm.toml
cp ./crates/circuits/bundle-circuit/openvm/verifier.bin $DIR_OUTPUT/bundle/verifier.bin
cp ./crates/circuits/bundle-circuit/openvm/verifier.sol $DIR_OUTPUT/bundle/verifier.sol
xxd -l 32 -p ./crates/circuits/bundle-circuit/digest_1 | tr -d '\n' | awk '{gsub("%", ""); print}' > $DIR_OUTPUT/bundle/digest_1.hex
xxd -l 32 -p ./crates/circuits/bundle-circuit/digest_2 | tr -d '\n' | awk '{gsub("%", ""); print}' > $DIR_OUTPUT/bundle/digest_2.hex
xxd -l 32 -p ./crates/circuits/bundle-circuit/digest_1_euclidv1 | tr -d '\n' | awk '{gsub("%", ""); print}' > $DIR_OUTPUT/bundle/digest_1_euclidv1.hex
xxd -l 32 -p ./crates/circuits/bundle-circuit/digest_2_euclidv1 | tr -d '\n' | awk '{gsub("%", ""); print}' > $DIR_OUTPUT/bundle/digest_2_euclidv1.hex

# copy verifier-only assets
cp $DIR_INPUT/bundle/root-verifier-vm-config $DIR_OUTPUT/verifier/root-verifier-vm-config
cp $DIR_INPUT/bundle/root-verifier-committed-exe $DIR_OUTPUT/verifier/root-verifier-committed-exe
cp ./crates/circuits/bundle-circuit/openvm/verifier.bin $DIR_OUTPUT/verifier/verifier.bin

# upload to s3
aws --profile default s3 cp $DIR_OUTPUT s3://circuit-release/scroll-zkvm/$DIR_OUTPUT --recursive
