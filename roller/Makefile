.PHONY: lint docker clean roller mock_roller

IMAGE_NAME=go-roller
IMAGE_VERSION=latest

ifeq (4.3,$(firstword $(sort $(MAKE_VERSION) 4.3)))
	ZK_VERSION=$(shell grep -m 1 "scroll-zkevm" ../common/libzkp/impl/Cargo.lock | cut -d "#" -f2 | cut -c-7)
else
	ZK_VERSION=$(shell grep -m 1 "scroll-zkevm" ../common/libzkp/impl/Cargo.lock | cut -d "\#" -f2 | cut -c-7)
endif

libzkp:
	cd ../common/libzkp/impl && cargo build --release && cp ./target/release/libzkp.a ../interface/
	rm -rf ./prover/lib && cp -r ../common/libzkp/interface ./prover/lib
	find ../common | grep libzktrie.so | xargs -I{} cp {} ./prover/lib/

roller: libzkp ## Build the Roller instance.
	GOBIN=$(PWD)/build/bin go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -o $(PWD)/build/bin/roller ./cmd

gpu-roller: libzkp ## Build the GPU Roller instance.
	GOBIN=$(PWD)/build/bin go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -tags gpu -o $(PWD)/build/bin/roller ./cmd

mock_roller:
	GOBIN=$(PWD)/build/bin go build -tags mock_prover -o $(PWD)/build/bin/roller $(PWD)/cmd

test-prover: libzkp
	go test -tags ffi -timeout 0 -v ./prover

test-gpu-prover: libzkp
	go test -tags="gpu ffi" -timeout 0 -v ./prover

lastest-zk-version:
	curl -sL https://api.github.com/repos/scroll-tech/scroll-zkevm/commits | jq -r ".[0].sha"

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

# docker:
# 	docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ../ -f ./Dockerfile
