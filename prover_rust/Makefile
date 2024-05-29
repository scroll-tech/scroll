.PHONY: prover

ifeq (4.3,$(firstword $(sort $(MAKE_VERSION) 4.3)))
    $(info ************  First ************)
    ZKEVM_VERSION=$(shell grep  "zkevm-circuits.git" ./Cargo.lock | sort | uniq | awk -F "[#=]" '{print $3"\t"$4}' | sort -k 1 | tail -n 1 | cut -d $'\t' -f2 | cut -c-7)
    HALO2_VERSION=$(shell grep -m 1 "halo2.git" ./Cargo.lock | cut -d "#" -f2 | cut -c-7)
else
    $(info ************  Second ************)
    ZKEVM_VERSION=$(shell grep  "zkevm-circuits.git" ./Cargo.lock | sort | uniq | awk -F "[\#=]" '{print $3"\t"$4}' | sort -k 1 | tail -n 1 | cut -d $'\t' -f2 | cut -c-7)
    HALO2_VERSION=$(shell grep -m 1 "halo2.git" ./Cargo.lock | cut -d "\#" -f2 | cut -c-7)
endif

HALO2_GPU_VERSION=$(shell ./print_halo2gpu_version.sh | sed -n '2p')

GIT_REV=$(shell git rev-parse --short HEAD)
GO_TAG=$(shell grep "var tag = " ../common/version/version.go | cut -d "\"" -f2)

ifeq (${ZKEVM_VERSION},)
    $(error ZKEVM_VERSION not set)
else
    $(info ZKEVM_VERSION is ${ZKEVM_VERSION})
endif

ifeq (${GO_TAG},)
    $(error GO_TAG not set)
else
    $(info GO_TAG is ${GO_TAG})
endif

ifeq (${HALO2_GPU_VERSION},)
	# use halo2_proofs with CPU
    ZK_VERSION=${ZKEVM_VERSION}-${HALO2_VERSION}
else
	# use halo2_gpu
    ZK_VERSION=${ZKEVM_VERSION}-${HALO2_GPU_VERSION}
endif

prover:
	GO_TAG=${GO_TAG} GIT_REV=${GIT_REV} ZK_VERSION=${ZK_VERSION} cargo build --release
	rm -rf ./lib && mkdir ./lib
	find target/ -name "libzktrie.so" | xargs -I{} cp {} ./lib