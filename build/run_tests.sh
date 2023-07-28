#!/bin/bash
set -uex

profile_name=$1

exclude_dirs=("scroll-tech/bridge/cmd" "scroll-tech/bridge/tests" "scroll-tech/bridge/mock_bridge" "scroll-tech/coordinator/cmd" "scroll-tech/coordinator/internal/logic/verifier")

all_packages=$(go list ./... | grep -v "^scroll-tech/${profile_name}$")
coverpkg="scroll-tech/${profile_name}"

for pkg in $all_packages; do
    exclude_pkg=false
    for exclude_dir in "${exclude_dirs[@]}"; do
        if [[ $pkg == $exclude_dir* ]]; then
            exclude_pkg=true
            break
        fi
    done

    if [ "$exclude_pkg" = false ]; then
        coverpkg="$coverpkg,$pkg/..."
    fi
done

echo "coverage.${profile_name}.txt"
go test -v -race -gcflags="-l" -ldflags="-s=false" -coverpkg="$coverpkg" -coverprofile=../coverage.${profile_name}.txt -covermode=atomic ./...