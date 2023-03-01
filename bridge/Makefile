.PHONY: lint docker clean bridge

IMAGE_NAME=bridge
IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

mock_abi:
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol mock_bridge/MockBridgeL1.sol --pkg mock_bridge --out mock_bridge/MockBridgeL1.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol mock_bridge/MockBridgeL2.sol --pkg mock_bridge --out mock_bridge/MockBridgeL2.go

bridge_bins: ## Builds the Bridge bins.
	go build -o $(PWD)/build/bin/event_watcher ./cmd/event_watcher/
	go build -o $(PWD)/build/bin/batch_proposer ./cmd/batch_proposer/
	go build -o $(PWD)/build/bin/message_relayer ./cmd/msg_relayer/
	go build -o $(PWD)/build/bin/rollup_relayer ./cmd/rollup_relayer/

event_watcher: ## Builds the event_watcher bin
	go build -o $(PWD)/build/bin/event_watcher ./cmd/event_watcher/

batch_proposer: ## Builds the batch_proposer bin
	go build -o $(PWD)/build/bin/batch_proposer ./cmd/batch_proposer/

message_relayer: ## Builds the message_relayer bin
	go build -o $(PWD)/build/bin/message_relayer ./cmd/msg_relayer/

rollup_relayer: ## Builds the rollup_relayer bin
	go build -o $(PWD)/build/bin/rollup_relayer ./cmd/rollup_relayer/

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/batch_proposer:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/batch_proposer.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/event_watcher:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/event_watcher.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/rollup_relayer:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/rollup_relayer.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/msg_relayer:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/msg_relayer.Dockerfile
