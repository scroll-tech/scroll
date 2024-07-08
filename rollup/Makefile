.PHONY: mock_abi libzstd rollup_bins gas_oracle rollup_relayer test lint clean docker

IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

mock_abi:
	git submodule update --init --recursive
	cd .. && solc --evm-version cancun --bin --abi --optimize --overwrite -o ./build/bin ./rollup/mock_bridge/MockBridge.sol
	cd .. && go run github.com/scroll-tech/go-ethereum/cmd/abigen --bin=./build/bin/MockBridge.bin --abi=./build/bin/MockBridge.abi --pkg=mock_bridge --out=./rollup/mock_bridge/MockBridge.go

rollup_bins: ## Builds the Rollup bins.
	go build -o $(PWD)/build/bin/gas_oracle ./cmd/gas_oracle/
	go build -o $(PWD)/build/bin/rollup_relayer ./cmd/rollup_relayer/

gas_oracle: ## Builds the gas_oracle bin
	go build -o $(PWD)/build/bin/gas_oracle ./cmd/gas_oracle/

rollup_relayer: ## Builds the rollup_relayer bin
	go build -o $(PWD)/build/bin/rollup_relayer ./cmd/rollup_relayer/

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

lint: mock_abi ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker_push:
	docker push scrolltech/gas-oracle:${IMAGE_VERSION}
	docker push scrolltech/rollup-relayer:${IMAGE_VERSION}

docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/gas-oracle:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/gas_oracle.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/rollup-relayer:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/rollup_relayer.Dockerfile
