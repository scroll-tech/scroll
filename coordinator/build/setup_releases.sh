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
  echo "Downloading assets for $FORK_NAME to $ASSET_DIR..."
  wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/verifier/verifier.bin -O ${ASSET_DIR}/verifier.bin
  wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/verifier/root_verifier_vk -O ${ASSET_DIR}/root_verifier_vk
  wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/verifier/openVmVk.json -O ${ASSET_DIR}/openVmVk.json
  wget https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/$SCROLL_ZKVM_VERSION/axiom_program_ids.json -O ${ASSET_DIR}/axiom_program_ids.json

  echo "Completed downloading assets for $FORK_NAME"
  echo "---"
done

echo "All verifier assets downloaded successfully"
