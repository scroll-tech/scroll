set -x
set -e

ZKEVM_VERSION=alpha-v1.0

function check_sha256() {
  real_sha256=`shasum -a 256 $1`
  sha256_file=`cat $2`
  if [ "$real_sha256" != "$sha256_file" ]; then
      exit 1
  fi
}

wget https://github.com/scroll-tech/scroll-zkevm/releases/download/$ZKEVM_VERSION/libs.zip -O ./lib/libs.zip
wget https://github.com/scroll-tech/scroll-zkevm/releases/download/$ZKEVM_VERSION/zip.sha256 -O ./lib/zip.sha256
check_sha256 libs.zip zip.sha256
rm zip.sha256

unzip -d ./lib ./lib/libs.zip
cd ./lib && check_sha256 libzkp.a zkp.sha256 && check_sha256 libzktrie.so zktrie.sha256

rm ./lib/libs.zip
rm ./lib/*.sha256

export CHAIN_ID=534353 # change to correct chain_id
export RUST_MIN_STACK=100000000
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./lib:/usr/local/cuda/   # cuda only for GPU machine
export ZK_VERSION=$ZKEVM_VERSION

