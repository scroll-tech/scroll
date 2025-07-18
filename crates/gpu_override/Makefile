.PHONY: build update clean

ZKVM_COMMIT ?= freebuild
PLONKY3_GPU_VERSION=$(shell ./print_plonky3gpu_version.sh | sed -n '2p')
$(info PLONKY3_GPU_VERSION is ${PLONKY3_GPU_VERSION})

GIT_REV ?= $(shell git rev-parse --short HEAD)
GO_TAG ?= $(shell grep "var tag = " ../../common/version/version.go | cut -d "\"" -f2)
ZK_VERSION=${ZKVM_COMMIT}-${PLONKY3_GPU_VERSION}
$(info ZK_GPU_VERSION is ${ZK_VERSION})

clean:
	cargo clean -Z unstable-options --release -p prover --lockfile-path ./Cargo.lock

# build gpu prover, never touch lock file
build:
	GO_TAG=${GO_TAG} GIT_REV=${GIT_REV} ZK_VERSION=${ZK_VERSION} cargo build -Z unstable-options --release -p prover --locked --lockfile-path ./Cargo.lock

# update Cargo.lock while override config has been updated
update:
	GO_TAG=${GO_TAG} GIT_REV=${GIT_REV} ZK_VERSION=${ZK_VERSION} cargo build -Z unstable-options --release -p prover --lockfile-path ./Cargo.lock