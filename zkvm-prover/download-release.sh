#!/bin/bash

# release version
if [ -z "${SCROLL_ZKVM_VERSION}" ]; then
  echo "SCROLL_ZKVM_VERSION not set"
  exit 1
fi

# chunk-circuit exe
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/chunk/app.vmexe -O crates/circuits/chunk-circuit/openvm/app.vmexe

# batch-circuit exe
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/batch/app.vmexe -O crates/circuits/batch-circuit/openvm/app.vmexe

# bundle-circuit exe
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/bundle/app.vmexe -O crates/circuits/bundle-circuit/openvm/app.vmexe

# bundle-circuit exe, legacy version, may not exist
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/bundle/app_euclidv1.vmexe -O crates/circuits/bundle-circuit/openvm/app_euclidv1.vmexe || echo "legacy app not exist for $SCROLL_ZKVM_VERSION"

# assets for verifier-only mode
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/verifier/root-verifier-vm-config -O crates/verifier/testdata/root-verifier-vm-config
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/verifier/root-verifier-committed-exe -O crates/verifier/testdata/root-verifier-committed-exe
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/verifier/verifier.bin -O crates/verifier/testdata/verifier.bin
