#!/bin/bash

# Define version mapping
declare -A VERSION_MAP
VERSION_MAP["euclid"]="0.4.3"
VERSION_MAP["feynman"]="0.5.2"

# release version
if [ -z "${SCROLL_ZKVM_VERSION}" ]; then
    
    # Check if first argument is provided and matches a known version name
    if [ -n "$1" ] && [ -n "${VERSION_MAP[$1]}" ]; then
        SCROLL_ZKVM_VERSION=${VERSION_MAP[$1]}
        echo "Setting SCROLL_ZKVM_VERSION to ${SCROLL_ZKVM_VERSION} based on '$1' argument"
    else
        # Default version if no argument or not recognized
        SCROLL_ZKVM_VERSION=0.5.2
    fi
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

