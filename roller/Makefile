.PHONY: lint docker clean roller mock_roller

IMAGE_NAME=go-roller
IMAGE_VERSION=latest

roller: ## Build the Roller instance.
	cd ../common/libzkp && bash build.sh
	GOBIN=$(PWD)/build/bin go build -o $(PWD)/build/bin/roller ./cmd

gpu-roller: ## Build the GPU Roller instance.
	cd ../common/libzkp && bash build.sh
	GOBIN=$(PWD)/build/bin go build -tags gpu -o $(PWD)/build/bin/roller ./cmd

mock_roller:
	GOBIN=$(PWD)/build/bin go build -tags mock -o $(PWD)/build/bin/roller $(PWD)/cmd

lastest-zk-version:
	curl -sL https://api.github.com/repos/scroll-tech/scroll-zkevm/commits | jq -r ".[0].sha"

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	cd ../common/libzkp/lib/ && rm *.a *.so
	@rm -rf build/bin

# docker:
# 	docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ../ -f ./Dockerfile
