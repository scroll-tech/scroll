.PHONY: prover


ifeq (4.3,$(firstword $(sort $(MAKE_VERSION) 4.3)))
    ZKEVM_VERSION=$(shell grep -m 1 "zkevm-circuits" ./Cargo.lock | cut -d "#" -f2 | cut -c-7)
    HALO2_VERSION=$(shell grep -m 1 "halo2.git" ./Cargo.lock | cut -d "#" -f2 | cut -c-7)
else
    ZKEVM_VERSION=$(shell grep -m 1 "zkevm-circuits" ./Cargo.lock | cut -d "\#" -f2 | cut -c-7)
    HALO2_VERSION=$(shell grep -m 1 "halo2.git" ./Cargo.lock | cut -d "\#" -f2 | cut -c-7)
endif

HALO2_GPU_VERSION=$(shell ./print_halo2gpu_version.sh | sed -n '2p')

GIT_REV=$(shell git rev-parse --short HEAD)

ifeq (${HALO2_GPU_VERSION},)
	# use halo2_proofs with CPU
    ZK_VERSION=${ZKEVM_VERSION}-${HALO2_VERSION}
else
	# use halo2_gpu
    ZK_VERSION=${ZKEVM_VERSION}-${HALO2_GPU_VERSION}
endif

prover:
	GIT_REV=${GIT_REV} ZK_VERSION=${ZK_VERSION} cargo build --release