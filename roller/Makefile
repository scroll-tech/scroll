.PHONY: lint docker clean roller

IMAGE_NAME=roller-go
IMAGE_VERSION=latest

prepare-libprover:
	cd roller/prover/rust && cargo build --release && cp target/release/libprover.a ../lib/

roller: ## Builds the Roller instance.
	cd roller/prover/rust && cargo build --release && cp target/release/libprover.a ../lib/
	GOBIN=$(PWD)/build/bin go install ./cmd/roller

gpu-roller: ## Builds the GPU Roller instance.
	cd roller/prover/rust && cargo build --release && cp target/release/libprover.a ../lib/
	GOBIN=$(PWD)/build/bin go install -tags gpu ./cmd/roller

test-prover:
	go test -timeout 0 -v ./roller/prover

test-gpu-prover:
	go test -tags gpu -timeout 0 -v ./roller/prover

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker:
	docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ./
