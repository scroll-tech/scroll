#!/usr/bin/sh

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
ASSETS_DIR="$BASE_DOWNLOAD_DIR/assets"
OPENVM_DIR="$BASE_DOWNLOAD_DIR/openvm"

# Create necessary directories
mkdir -p "$ASSETS_DIR"
mkdir -p "$OPENVM_DIR/verifier"

# Define URLs for asset files
ASSETS_CHECKSUM_URL="https://circuit-release.s3.us-west-2.amazonaws.com/release-v0.13.1/sha256sum"
ASSETS_CHECKSUM_FILE="$ASSETS_DIR/sha256sum"

ASSETS_URLS=(
  "https://circuit-release.s3.us-west-2.amazonaws.com/release-v0.13.1/vk_batch.vkey"
  "https://circuit-release.s3.us-west-2.amazonaws.com/release-v0.13.1/vk_bundle.vkey"
  "https://circuit-release.s3.us-west-2.amazonaws.com/release-v0.13.1/vk_chunk.vkey"
)

# Define URLs for OpenVM files (No checksum verification)
OPENVM_URLS=(
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/verifier/verifier.bin"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/verifier/root-verifier-vm-config"
  "https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/0.2.0/verifier/root-verifier-committed-exe"
)

# Function to download and verify files (skips existing valid files)
download_and_verify() {
  local url="$1"
  local dest_dir="$2"
  local checksum_file="$3"

  local filename=$(basename "$url")
  local filepath="$dest_dir/$filename"

  if [[ -f "$filepath" ]]; then
    echo "Checking existing file: $filename..."
    if grep "$filename" "$checksum_file" | sed "s|$filename|$filepath|" | sha256sum --check --status; then
      echo "File is already present and valid ✅ - Skipping download."
      return
    else
      echo "File exists but checksum mismatch ❌ - Re-downloading."
      rm -f "$filepath"
    fi
  fi

  echo "Downloading $filename..."
  curl -o "$filepath" -L "$url"

  if [[ ! -f "$filepath" ]]; then
    echo "Download failed for $filename ❌"
    exit 1
  fi

  echo "Verifying checksum for $filename..."
  grep "$filename" "$checksum_file" | sed "s|$filename|$filepath|" | sha256sum --check --status || exit 1
  echo "Checksum verification passed for $filename ✅"
}

# Download and verify asset files
curl -o "$ASSETS_CHECKSUM_FILE" -L "$ASSETS_CHECKSUM_URL"
for url in "${ASSETS_URLS[@]}"; do
  download_and_verify "$url" "$ASSETS_DIR" "$ASSETS_CHECKSUM_FILE"
done

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
