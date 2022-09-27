#!/bin/bash
set -uex

export DX_CLEAN=${DX_CLEAN:-TRUE}
echo "DX_CLEAN: $DX_CLEAN"

function kill_tasks() {
  # kill last time running tasks:
  echo "===================process list=========================="
  ps aux | grep 'rollup_explorer' | grep -v grep | awk '{print $2 " " $11}'
  kill -9 $(ps aux | grep 'rollup_explorer' | grep -v grep | awk '{print $2}') || true
  echo "===================process list=========================="
}

function clean_data() {
  rm -rf docker-data
}

kill_tasks
docker compose -f docker-compose.yml down --remove-orphans
if [ $DX_CLEAN == 'TRUE' ]; then
  clean_data
fi
