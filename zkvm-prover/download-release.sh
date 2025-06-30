#!/bin/bash

# release version
if [ -z "${SCROLL_ZKVM_VERSION}" ]; then
    SCROLL_ZKVM_VERSION=$($SHELL ./print_high_zkvm_version.sh | cut -d' ' -f1|cut -c2-)
fi

echo $SCROLL_ZKVM_VERSION

# chunk-circuit exe
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/chunk/app.vmexe -O .work/chunk/app.vmexe

# batch-circuit exe
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/batch/app.vmexe -O .work/batch/app.vmexe

# bundle-circuit exe
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/bundle/app.vmexe -O .work/bundle/app.vmexe

# bundle-circuit exe, legacy version, may not exist
wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/bundle/app_euclidv1.vmexe -O .work/bundle/app_euclidv1.vmexe || echo "legacy app not exist for $SCROLL_ZKVM_VERSION"
