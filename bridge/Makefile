.PHONY: lint docker clean bridge

IMAGE_NAME=bridge
IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

mock_abi:
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol mock_bridge/MockBridgeL1.sol --pkg mock_bridge --out mock_bridge/MockBridgeL1.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol mock_bridge/MockBridgeL2.sol --pkg mock_bridge --out mock_bridge/MockBridgeL2.go

bridge: ## Builds the Bridge instance.
	go build -o $(PWD)/build/bin/bridge ./cmd

multi_bin: ## Builds multiple bins of Bridge(event_watcher, msg_relayer, rollup_relayer, batch_proposer)
	go build -o $(PWD)/build/bin/multibin/event_watcher ./multibin/event_watcher/cmd
	go build -o $(PWD)/build/bin/multibin/batch_proposer ./multibin/batch_proposer/cmd
	go build -o $(PWD)/build/bin/multibin/message_relayer ./multibin/message_relayer/cmd
	go build -o $(PWD)/build/bin/multibin/rollup_relayer ./multibin/rollup_relayer/cmd
test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

lint: ## Lint the files - used for CI.
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder.
	@rm -rf build/bin

docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/bridge.Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}
