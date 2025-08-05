#!/usr/bin/bash

apt update
apt install -y wget libdigest-sha-perl

# release version
if [ -z "${SCROLL_ZKVM_VERSION}" ]; then
  echo "SCROLL_ZKVM_VERSION not set"
  exit 1
fi

if [ -z "${HTTP_PORT}" ]; then
  echo "HTTP_PORT not set"
  exit 1
fi

if [ -z "${METRICS_PORT}" ]; then
  echo "METRICS_PORT not set"
  exit 1
fi

case $CHAIN_ID in
"5343532222") # staging network
  echo "staging network not supported"
  exit 1
  ;;
"534353") # alpha network
  echo "alpha network not supported"
  exit 1
  ;;
esac

BASE_DOWNLOAD_DIR="/verifier"
# Ensure the base directory exists
mkdir -p "$BASE_DOWNLOAD_DIR"

# Set subdirectories
OPENVM_DIR="$BASE_DOWNLOAD_DIR/openvm"

# Create necessary directories
mkdir -p "$OPENVM_DIR/verifier"

# Define URLs for OpenVM files (No checksum verification)
OPENVM_URLS=(
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/verifier/verifier.bin"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/verifier/root-verifier-vm-config"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/verifier/root-verifier-committed-exe"
)

# Download OpenVM files (No checksum verification, but skips if file exists)
for url in "${OPENVM_URLS[@]}"; do
  dest_subdir="$OPENVM_DIR/$(basename $(dirname "$url"))"
  mkdir -p "$dest_subdir"

  filepath="$dest_subdir/$(basename "$url")"
  echo "Downloading $filepath..."
  curl -o "$filepath" -L "$url"
done

mkdir -p "$HOME/.openvm"
ln -s "$OPENVM_DIR/params" "$HOME/.openvm/params"

echo "All files downloaded successfully! 🎉"

mkdir -p /usr/local/bin
wget https://github.com/ethereum/solidity/releases/download/v0.8.19/solc-static-linux -O /usr/local/bin/solc
chmod +x /usr/local/bin/solc

# Start coordinator
echo "Starting coordinator api"

RUST_BACKTRACE=1 exec coordinator_api --config /coordinator/config.json \
    --genesis /coordinator/genesis.json \
    --http --http.addr "0.0.0.0" --http.port ${HTTP_PORT} \
    --metrics --metrics.addr "0.0.0.0" --metrics.port ${METRICS_PORT} \
    --log.debug
