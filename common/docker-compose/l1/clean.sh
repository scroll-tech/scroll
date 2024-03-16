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
