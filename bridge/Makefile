.PHONY: lint docker clean bridge

IMAGE_NAME=bridge
IMAGE_VERSION=latest

mock_abi:
	solc --bin mock_bridge/Mock_Bridge.sol -o mock_bridge/ --overwrite
	solc --abi mock_bridge/Mock_Bridge.sol -o mock_bridge/ --overwrite
	abigen --bin=mock_bridge/Mock_Bridge.bin \
			--abi=mock_bridge/Mock_Bridge.abi \
			--pkg=mock_bridge --out=mock_bridge/Mock_Bridge.go
	awk '{sub("github.com/ethereum","github.com/scroll-tech")}1' mock_bridge/Mock_Bridge.go > temp && mv temp mock_bridge/Mock_Bridge.go

bridge: ## Builds the Bridge instance.
	go build -o $(PWD)/build/bin/bridge ./cmd

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker:
#	docker container prune -f
#	docker build -t scroll_l1geth ../common/docker/l1geth/
#	docker build -t scroll_l2geth ../common/docker/l2geth/
	docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ../ -f ./Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}
