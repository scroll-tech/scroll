#!/usr/bin/bash

apt update
apt install -y wget curl

BASE_DOWNLOAD_DIR="/openvm"
# Ensure the base directory exists
mkdir -p "$BASE_DOWNLOAD_DIR"

# Define URLs for OpenVM files (No checksum verification)
OPENVM_URLS=(
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/chunk/app.vmexe"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/chunk/openvm.toml"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/batch/app.vmexe"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/batch/openvm.toml"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/bundle/app.vmexe"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/bundle/app_euclidv1.vmexe"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/bundle/openvm.toml"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/bundle/verifier.bin"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/bundle/verifier.sol"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/bundle/digest_1.hex"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/bundle/digest_2.hex"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/bundle/digest_1_euclidv1.hex"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/bundle/digest_2_euclidv1.hex"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/params/kzg_bn254_22.srs"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/params/kzg_bn254_24.srs"
)

# Download OpenVM files (No checksum verification, but skips if file exists)
for url in "${OPENVM_URLS[@]}"; do
  dest_subdir="$BASE_DOWNLOAD_DIR/$(basename $(dirname "$url"))"
  mkdir -p "$dest_subdir"

  filepath="$dest_subdir/$(basename "$url")"
  echo "Downloading $filepath..."
  curl -o "$filepath" -L "$url"
done

mkdir -p "$HOME/.openvm"
ln -s "/openvm/params" "$HOME/.openvm/params"

mkdir -p /usr/local/bin
wget https://github.com/ethereum/solidity/releases/download/v0.8.19/solc-static-linux -O /usr/local/bin/solc
chmod +x /usr/local/bin/solc

mkdir -p /openvm/cache

RUST_MIN_STACK=16777216 RUST_BACKTRACE=1 exec prover --config /config.json

