.PHONY: lint docker clean roller mock_roller

IMAGE_NAME=go-roller
IMAGE_VERSION=latest

ifeq (4.3,$(firstword $(sort $(MAKE_VERSION) 4.3)))
	ZK_VERSION=$(shell grep -m 1 "common-rs" core/prover/rust/Cargo.lock | cut -d "#" -f2 | cut -c-7)
else
	ZK_VERSION=$(shell grep -m 1 "common-rs" core/prover/rust/Cargo.lock | cut -d "\#" -f2 | cut -c-7)
endif

libprover:
	cd core/prover/rust && cargo build --release && cp target/release/libprover.a ../lib/

roller: ## Build the Roller instance.
	cd core/prover/rust && cargo build --release && cp target/release/libprover.a ../lib/
	GOBIN=$(PWD)/build/bin go build -ldflags "-X scroll-tech/roller/core.ZK_VERSION=${ZK_VERSION}" -o $(PWD)/build/bin/roller ./cmd

mock_roller:
	GOBIN=$(PWD)/build/bin go build -tags mock_prover -o $(PWD)/build/bin/roller $(PWD)/cmd

gpu-roller: ## Build the GPU Roller instance.
	cd core/prover/rust && cargo build --release && cp target/release/libprover.a ../lib/
	GOBIN=$(PWD)/build/bin go build -ldflags "-X scroll-tech/roller/core.ZK_VERSION=${ZK_VERSION}" -tags gpu roller -o $(PWD)/build/bin/roller ./cmd

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
