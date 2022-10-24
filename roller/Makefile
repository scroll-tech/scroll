.PHONY: lint docker clean roller mock_roller

IMAGE_NAME=go-roller
IMAGE_VERSION=latest

libprover:
	cd core/prover/rust && cargo build --release && cp target/release/libprover.a ../lib/

roller: ## Build the Roller instance.
	cd core/prover/rust && cargo build --release && cp target/release/libprover.a ../lib/
	GOBIN=$(PWD)/build/bin go build -o $(PWD)/build/bin/roller ./cmd

mock_roller:
	go build -o $(PWD)/build/bin/roller -tags mock_prover $(PWD)/cmd/roller

gpu-roller: ## Builds the GPU Roller instance.
	cd core/prover/rust && cargo build --release && cp target/release/libprover.a ../lib/
	GOBIN=$(PWD)/build/bin go build -tags gpu -o $(PWD)/build/bin/roller ./cmd

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
