.PHONY: lint docker clean prover mock-prover

ifeq (4.3,$(firstword $(sort $(MAKE_VERSION) 4.3)))
    ZKEVM_VERSION=$(shell grep -m 1 "zkevm-circuits" ../common/libzkp/impl/Cargo.lock | cut -d "#" -f2 | cut -c-7)
    HALO2_VERSION=$(shell grep -m 1 "halo2.git" ../common/libzkp/impl/Cargo.lock | cut -d "#" -f2 | cut -c-7)
else
    ZKEVM_VERSION=$(shell grep -m 1 "zkevm-circuits" ../common/libzkp/impl/Cargo.lock | cut -d "\#" -f2 | cut -c-7)
    HALO2_VERSION=$(shell grep -m 1 "halo2.git" ../common/libzkp/impl/Cargo.lock | cut -d "\#" -f2 | cut -c-7)
endif

HALO2_GPU_VERSION=$(shell ./print_halo2gpu_version.sh | sed -n '2p')

ifeq (${HALO2_GPU_VERSION},)
	# use halo2_proofs with CPU
    ZK_VERSION=${ZKEVM_VERSION}-${HALO2_VERSION}
else
	# use halo2_gpu
    ZK_VERSION=${ZKEVM_VERSION}-${HALO2_GPU_VERSION}
endif

libzkp:
	cd ../common/libzkp/impl && cargo clean && cargo build --release && cp ./target/release/libzkp.so ../interface/
	rm -rf ./core/lib && cp -r ../common/libzkp/interface ./core/lib
	find ../common | grep libzktrie.so | xargs -I{} cp {} ./core/lib/

prover: libzkp ## Build the Prover instance.
	GOBIN=$(PWD)/build/bin go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -o $(PWD)/build/bin/prover ./cmd

mock-prover: ## Build the mocked Prover instance.
	GOBIN=$(PWD)/build/bin go build -tags="mock_prover mock_verifier" -o $(PWD)/build/bin/prover ./cmd

gpu-prover: libzkp ## Build the GPU Prover instance.
	GOBIN=$(PWD)/build/bin go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -tags gpu -o $(PWD)/build/bin/prover ./cmd

test-prover: libzkp
	go test -tags ffi -timeout 0 -v ./prover

test-gpu-prover: libzkp
	go test -tags="gpu ffi" -timeout 0 -v ./prover

lastest-zk-version:
	curl -sL https://api.github.com/repos/scroll-tech/zkevm-circuits/commits | jq -r ".[0].sha"

lint: ## Lint the files - used for CI
	cp -r ../common/libzkp/interface ./core/lib
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin
