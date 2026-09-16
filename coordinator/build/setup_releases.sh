#!/bin/bash

# release version
if [ -z "${SCROLL_ZKVM_VERSION}" ]; then
  echo "SCROLL_ZKVM_VERSION not set"
  exit 1
fi

# default fork name from env or "galileo"
SCROLL_FORK_NAME="${SCROLL_FORK_NAME:-galileov2}"

# set ASSET_DIR by reading from config.json
CONFIG_FILE="bin/conf/config.template.json"
if [ ! -f "$CONFIG_FILE" ]; then
  echo "Config file $CONFIG_FILE not found"
  exit 1
fi

# get the number of verifiers in the array
VERIFIER_COUNT=$(jq -r '.prover_manager.verifier.verifiers | length' "$CONFIG_FILE")

if [ "$VERIFIER_COUNT" = "null" ] || [ "$VERIFIER_COUNT" -eq 0 ]; then
  echo "No verifiers found in config file"
  exit 1
fi

echo "Found $VERIFIER_COUNT verifier(s) in config"

# iterate through each verifier entry
for ((i=0; i<$VERIFIER_COUNT; i++)); do
  # extract assets_path for current verifier
  ASSETS_PATH=$(jq -r ".prover_manager.verifier.verifiers[$i].assets_path" "$CONFIG_FILE")
  FORK_NAME=$(jq -r ".prover_manager.verifier.verifiers[$i].fork_name" "$CONFIG_FILE")

  # skip if this verifier's fork doesn't match the target fork
  if [ "$FORK_NAME" != "$SCROLL_FORK_NAME" ]; then
    echo "Expect $SCROLL_FORK_NAME, skip current fork ($FORK_NAME)"
    continue
  fi

  if [ "$ASSETS_PATH" = "null" ]; then
    echo "Warning: Could not find assets_path for verifier $i, skipping..."
    continue
  fi

  echo "Processing verifier $i ($FORK_NAME): assets_path=$ASSETS_PATH"

  # check if it's an absolute path (starts with /)
  if [[ "$ASSETS_PATH" = /* ]]; then
    # absolute path, use as is
    ASSET_DIR="$ASSETS_PATH"
  else
    # relative path, prefix with "bin/"
    ASSET_DIR="bin/$ASSETS_PATH"
  fi

  echo "Using ASSET_DIR: $ASSET_DIR"

  # create directory if it doesn't exist
  mkdir -p "$ASSET_DIR"

  # assets for verifier-only mode
  # v0.9.0+ publishes verifier assets under releases/<ver>/verifier/, while
  # older releases used <ver>/verifier/ (no "releases/" segment). Probe once
  # and use whichever layout exists for this version.
  echo "Downloading assets for $FORK_NAME to $ASSET_DIR..."
  S3_BASE="https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm"
  if wget -q --spider "$S3_BASE/releases/$SCROLL_ZKVM_VERSION/verifier/verifier.bin"; then
    VERIFIER_URL_BASE="$S3_BASE/releases/$SCROLL_ZKVM_VERSION/verifier"
  else
    VERIFIER_URL_BASE="$S3_BASE/$SCROLL_ZKVM_VERSION/verifier"
  fi
  wget ${VERIFIER_URL_BASE}/verifier.bin -O ${ASSET_DIR}/verifier.bin
  wget ${VERIFIER_URL_BASE}/root_verifier_vk -O ${ASSET_DIR}/root_verifier_vk
  wget ${VERIFIER_URL_BASE}/openVmVk.json -O ${ASSET_DIR}/openVmVk.json
  # agg_vk.bin: the batch circuit's aggregation VK, mandatory for batch proof
  # verification (the coordinator refuses to start without it).
  wget ${VERIFIER_URL_BASE}/agg_vk.bin -O ${ASSET_DIR}/agg_vk.bin
  
  echo "Completed downloading assets for $FORK_NAME"
  echo "---"
done

echo "All verifier assets downloaded successfully"