#!/bin/bash

# release version
if [ -z "${SCROLL_ZKVM_VERSION}" ]; then
#   before we use version tag for zkvm, we can not acquire the correct version of zkvm from script
#    SCROLL_ZKVM_VERSION=$($SHELL ./print_high_zkvm_version.sh | cut -d' ' -f1|cut -c2-)
    SCROLL_ZKVM_VERSION=0.5.0rc0
fi

echo $SCROLL_ZKVM_VERSION

mkdir -p .work/chunk
mkdir -p .work/batch
mkdir -p .work/bundle

# chunk-circuit exe
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/chunk/app.vmexe -O .work/chunk/app.vmexe
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/chunk/openvm.toml -O .work/chunk/openvm.toml
# batch-circuit exe
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/batch/app.vmexe -O .work/batch/app.vmexe
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/batch/openvm.toml -O .work/batch/openvm.toml
# bundle-circuit exe
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/bundle/app.vmexe -O .work/bundle/app.vmexe
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/bundle/openvm.toml -O .work/bundle/openvm.toml

