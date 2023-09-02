.PHONY: mock_abi bridge_bins event_watcher gas_oracle rollup_relayer test lint clean docker

IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

mock_abi:
	cd .. && go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol ./rollup/mock_bridge/MockBridgeL1.sol --pkg mock_bridge --out ./rollup/mock_bridge/MockBridgeL1.go
	cd .. && go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol ./rollup/mock_bridge/MockBridgeL2.sol --pkg mock_bridge --out ./rollup/mock_bridge/MockBridgeL2.go

bridge_bins: ## Builds the Bridge bins.
	go build -o $(PWD)/build/bin/event_watcher ./cmd/event_watcher/
	go build -o $(PWD)/build/bin/gas_oracle ./cmd/gas_oracle/
	go build -o $(PWD)/build/bin/rollup_relayer ./cmd/rollup_relayer/

event_watcher: ## Builds the event_watcher bin
	go build -o $(PWD)/build/bin/event_watcher ./cmd/event_watcher/

gas_oracle: ## Builds the gas_oracle bin
	go build -o $(PWD)/build/bin/gas_oracle ./cmd/gas_oracle/

rollup_relayer: ## Builds the rollup_relayer bin
	go build -o $(PWD)/build/bin/rollup_relayer ./cmd/rollup_relayer/

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker_push:
	docker docker push scrolltech/gas-oracle:${IMAGE_VERSION}
	docker docker push scrolltech/event-watcher:${IMAGE_VERSION}
	docker docker push scrolltech/rollup-relayer:${IMAGE_VERSION}

docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/gas-oracle:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/gas_oracle.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/event-watcher:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/event_watcher.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/rollup-relayer:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/rollup_relayer.Dockerfile
