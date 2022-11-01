.PHONY: lint docker clean roller

IMAGE_NAME=go-roller
IMAGE_VERSION=latest

ZK_VERSION=$(shell grep -m 1 "common-rs" core/prover/rust/Cargo.lock | cut -d "\#" -f2 | sed 's/"//g')

libprover:
	cd core/prover/rust && cargo build --release && cp target/release/libprover.a ../lib/

roller: ## Build the Roller instance.
	cd core/prover/rust && cargo build --release && cp target/release/libprover.a ../lib/
	GOBIN=$(PWD)/build/bin go build -ldflags "-X core.ZK_VERSION=${ZK_VERSION}" -o $(PWD)/build/bin/roller ./cmd

gpu-roller: ## Build the GPU Roller instance.
	cd core/prover/rust && cargo build --release && cp target/release/libprover.a ../lib/
	GOBIN=$(PWD)/build/bin go build -ldflags "-X core.ZK_VERSION=${ZK_VERSION}" -tags gpu -o $(PWD)/build/bin/roller ./cmd

test-prover:
	go test -timeout 0 -v ./core/prover

test-gpu-prover:
	go test -tags gpu -timeout 0 -v ./core/prover

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

# docker:
# 	docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ../ -f ./Dockerfile
