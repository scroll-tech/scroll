.PHONY: lint docker clean bridge

IMAGE_NAME=bridge
IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

mock_abi:
	abigen --sol mock_bridge/Mock_Bridge.sol --pkg mock_bridge --out mock_bridge/Mock_Bridge.go

bridge: ## Builds the Bridge instance.
	go build -o $(PWD)/build/bin/bridge ./cmd

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/bridge.Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}
