.PHONY: lint docker clean roller

IMAGE_NAME=go-roller
IMAGE_VERSION=latest

ifeq (4.3,$(firstword $(sort $(MAKE_VERSION) 4.3)))
	ZK_VERSION=$(shell grep -m 1 "common-rs" ../common/zkp/rust/Cargo.lock | cut -d "#" -f2 | cut -c-7)
else
	ZK_VERSION=$(shell grep -m 1 "common-rs" ../common/zkp/rust/Cargo.lock | cut -d "\#" -f2 | cut -c-7)
endif

libzkp:
	cd ../common/zkp/rust && cargo build --release && cp ./target/release/libzkp.a ../lib/
	cp -r ../common/zkp/lib ./core/prover/

roller: ## Build the Roller instance.
	@libzkp
	GOBIN=$(PWD)/build/bin go build -ldflags "-X scroll-tech/roller/core.ZK_VERSION=${ZK_VERSION}" -o $(PWD)/build/bin/roller ./cmd

gpu-roller: ## Build the GPU Roller instance.
	@libzkp
	GOBIN=$(PWD)/build/bin go build -ldflags "-X scroll-tech/roller/core.ZK_VERSION=${ZK_VERSION}" -tags gpu -o $(PWD)/build/bin/roller ./cmd

test-prover:
	go test -timeout 0 -v ./core/prover

test-gpu-prover:
	go test -tags gpu -timeout 0 -v ./core/prover

lastest-zk-version:
	curl -sL https://api.github.com/repos/scroll-tech/common-rs/commits | jq -r ".[0].sha"

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

# docker:
# 	docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ../ -f ./Dockerfile
