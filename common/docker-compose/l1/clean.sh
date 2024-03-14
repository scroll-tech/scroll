#!/bin/bash

# Loop until no Docker containers matching 'posl1' are found
while : ; do
  containers=$(docker ps -a --format '{{.Names}}' | grep posl1)
  if [[ -z "$containers" ]]; then
    break
  fi
  echo "find the following containers, removing..."
  echo "$containers"
  echo "$containers" | xargs -r docker stop
  echo "$containers" | xargs -r docker rm -f || echo "Warning: Failed to remove some containers."
done

# Loop until no Docker networks matching 'posl1' are found
while : ; do
  networks=$(docker network ls --format '{{.ID}} {{.Name}}' | grep posl1 | awk '{print $1}')
  if [[ -z "$networks" ]]; then
    break
  fi
  echo "find the following networks, removing..."
  echo "$networks"
  echo "$networks" | xargs -r docker network rm || echo "Warning: Failed to remove some networks."
done

# Remove consensus data directories if they exist
if [ -d "./consensus/beacondata" ] || [ -d "./consensus/validatordata" ] || [ -d "./consensus/genesis.ssz" ]; then
  rm -rf ./consensus/beacondata ./consensus/validatordata ./consensus/genesis.ssz
  echo "Consensus data removed."
else
  echo "No consensus data to remove."
fi

# Remove execution data directory if it exists
if [ -d "./execution/geth" ]; then
  rm -rf ./execution/geth
  echo "Execution data removed."
else
  echo "No execution data to remove."
fi
