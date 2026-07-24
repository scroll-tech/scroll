.PHONY: prover prover_cpu lint tests_binary test_e2e_run test_run

RUST_LOG ?= off,scroll_zkvm_integration=debug,scroll_zkvm_verifier=debug,scroll_zkvm_prover=debug,p3_fri=warn,p3_dft=warn,openvm_circuit=warn
RUST_MIN_STACK ?= 16777216
export RUST_MIN_STACK

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

$(info PLONKY3_VERSION is ${PLONKY3_VERSION})

GIT_REV=$(shell git rev-parse --short HEAD)
GO_TAG=$(shell grep "var tag = " ../common/version/version.go | cut -d "\"" -f2)

ifeq (${GO_TAG},)
    $(error GO_TAG not set)
else
    $(info GO_TAG is ${GO_TAG})
endif

ZK_VERSION=${ZKVM_COMMIT}-${PLONKY3_VERSION}

E2E_HANDLE_SET ?= ../tests/prover-e2e/testset.json
DUMP_DIR ?= .work

version:
	echo ${GO_TAG}-${GIT_REV}-${ZK_VERSION}
	cargo clean --release -p scroll-proving-sdk

prover: version
	GO_TAG=${GO_TAG} GIT_REV=${GIT_REV} ZK_VERSION=${ZK_VERSION} cargo build --locked --release --features cuda -p prover

# GPU prover with halo2-gpu: SNARK (bundle) aggregation also runs on GPU.
# Needs a 24 GB-class GPU (RTX 3090/4090); superset of the `cuda` feature.
prover_halo2gpu: version
	GO_TAG=${GO_TAG} GIT_REV=${GIT_REV} ZK_VERSION=${ZK_VERSION} cargo build --locked --release --features halo2-gpu -p prover

prover_cpu: version
	GO_TAG=${GO_TAG} GIT_REV=${GIT_REV} ZK_VERSION=${ZK_VERSION} cargo build --locked --release -p prover

tests_binary:
	cargo clean && cargo test --release --no-run
	ls ../target/release/deps/prover* | grep -v "\.d" | xargs -I{} ln -sf {} ./prover.test

lint:
	cargo check --all-features
	cargo clippy --all-features --all-targets -- -D warnings
	cargo fmt --all

test_run: version 
	GO_TAG=${GO_TAG} GIT_REV=${GIT_REV} ZK_VERSION=${ZK_VERSION} cargo run --release -p prover -- --config ./config.json

test_e2e_run: ${E2E_HANDLE_SET} version
	GO_TAG=${GO_TAG} GIT_REV=${GIT_REV} ZK_VERSION=${ZK_VERSION} cargo run --release -p prover -- --config ./config.json handle ${E2E_HANDLE_SET}

test_e2e_run_gpu: ${E2E_HANDLE_SET} version
	GO_TAG=${GO_TAG} GIT_REV=${GIT_REV} ZK_VERSION=${ZK_VERSION} cargo run --release --features cuda -p prover -- --config ./config.json handle ${E2E_HANDLE_SET}

