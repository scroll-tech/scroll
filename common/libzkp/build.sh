set -x
set -e

ZKEVM_VERSION=alpha-v1.0

wget https://github.com/scroll-tech/scroll-zkevm/releases/download/$ZKEVM_VERSION/libs.zip -O ./lib/libs.zip
unzip -d ./lib ./lib/libs.zip

export CHAIN_ID=534353 # change to correct chain_id
export RUST_MIN_STACK=100000000
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./lib:/usr/local/cuda/   # cuda only for GPU machine
export ZK_VERSION=$ZKEVM_VERSION