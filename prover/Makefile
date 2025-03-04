.PHONY: prover lint tests_binary

ifeq (4.3,$(firstword $(sort $(MAKE_VERSION) 4.3)))
    HALO2_VERSION=$(shell grep -m 1 "halo2.git" ./Cargo.lock | cut -d "#" -f2 | cut -c-7)
else
    HALO2_VERSION=$(shell grep -m 1 "halo2.git" ./Cargo.lock | cut -d "\#" -f2 | cut -c-7)
endif

ZKEVM_VERSION=$(shell ./print_high_zkevm_version.sh)
ifeq (${ZKEVM_VERSION},)
    $(error ZKEVM_VERSION not set)
else
    $(info ZKEVM_VERSION is ${ZKEVM_VERSION})
endif

ZKEVM_COMMIT=$(shell echo ${ZKEVM_VERSION} | cut -d " " -f2)
$(info ZKEVM_COMMIT is ${ZKEVM_COMMIT})

HALO2_GPU_VERSION=$(shell ./print_halo2gpu_version.sh | sed -n '2p')

GIT_REV=$(shell git rev-parse --short HEAD)
GO_TAG=$(shell grep "var tag = " ../common/version/version.go | cut -d "\"" -f2)

ifeq (${GO_TAG},)
    $(error GO_TAG not set)
else
    $(info GO_TAG is ${GO_TAG})
endif

ifeq (${HALO2_GPU_VERSION},)
	# use halo2_proofs with CPU
    ZK_VERSION=${ZKEVM_COMMIT}-${HALO2_VERSION}
else
	# use halo2_gpu
    ZK_VERSION=${ZKEVM_COMMIT}-${HALO2_GPU_VERSION}
endif

prover:
	GO_TAG=${GO_TAG} GIT_REV=${GIT_REV} ZK_VERSION=${ZK_VERSION} cargo build --release

tests_binary:
	cargo clean && cargo test --release --no-run
	ls target/release/deps/prover* | grep -v "\.d" | xargs -I{} ln -sf {} ./prover.test

lint:
	cargo check --all-features
	cargo clippy --all-features --all-targets -- -D warnings
	cargo fmt --all
