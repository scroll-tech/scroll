.PHONY: prover lint tests_binary

RUST_MIN_STACK ?= 16777216
export RUST_MIN_STACK

CIRCUIT_STUFF = .work/chunk/app.vmexe .work/batch/app.vmexe .work/bundle/app.vmexe

ifeq (4.3,$(firstword $(sort $(MAKE_VERSION) 4.3)))
    PLONKY3_VERSION=$(shell grep -m 1 "Plonky3.git" ../Cargo.lock | cut -d "#" -f2 | cut -c-7)
else
    PLONKY3_VERSION=$(shell grep -m 1 "Plonky3.git" ../Cargo.lock | cut -d "\#" -f2 | cut -c-7)
endif

ZKVM_VERSION=$(shell ./print_high_zkvm_version.sh)
ifeq (${ZKVM_VERSION},)
    $(error ZKVM_VERSION not set)
else
    $(info ZKVM_VERSION is ${ZKVM_VERSION})
endif

ZKVM_COMMIT=$(shell echo ${ZKVM_VERSION} | cut -d " " -f2)
$(info ZKVM_COMMIT is ${ZKVM_COMMIT})

PLONKY3_GPU_VERSION=$(shell ./print_plonky3gpu_version.sh | sed -n '2p')

GIT_REV=$(shell git rev-parse --short HEAD)
GO_TAG=$(shell grep "var tag = " ../common/version/version.go | cut -d "\"" -f2)

ifeq (${GO_TAG},)
    $(error GO_TAG not set)
else
    $(info GO_TAG is ${GO_TAG})
endif

ifeq (${PLONKY3_GPU_VERSION},)
	# use plonky3 with CPU
    ZK_VERSION=${ZKVM_COMMIT}-${PLONKY3_VERSION}
else
	# use halo2_gpu
    ZK_VERSION=${ZKVM_COMMIT}-${PLONKY3_GPU_VERSION}
endif

prover:
	GO_TAG=${GO_TAG} GIT_REV=${GIT_REV} ZK_VERSION=${ZK_VERSION} cd ../crates/prover-bin && cargo build --release

tests_binary:
	cargo clean && cargo test --release --no-run
	ls ../target/release/deps/prover* | grep -v "\.d" | xargs -I{} ln -sf {} ./prover.test

lint:
	cargo check --all-features
	cargo clippy --all-features --all-targets -- -D warnings
	cargo fmt --all

$(CIRCUIT_STUFF):
	bash .work/download-release.sh

test_run: $(CIRCUIT_STUFF)
	GO_TAG=${GO_TAG} GIT_REV=${GIT_REV} ZK_VERSION=${ZK_VERSION} cargo run --release -p prover -- --config ./config.json