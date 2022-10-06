.PHONY: lint docker clean bridge

IMAGE_NAME=bridge
IMAGE_VERSION=latest

mock_abi:
	solc --bin bridge/mock_bridge/Mock_Bridge.sol -o bridge/mock_bridge/ --overwrite
	solc --abi bridge/mock_bridge/Mock_Bridge.sol -o bridge/mock_bridge/ --overwrite
	abigen --bin=bridge/mock_bridge/Mock_Bridge.bin \
			--abi=bridge/mock_bridge/Mock_Bridge.abi \
			--pkg=mock_bridge --out=bridge/mock_bridge/Mock_Bridge.go
	awk '{sub("github.com/ethereum","github.com/scroll-tech")}1' bridge/mock_bridge/Mock_Bridge.go > temp && mv temp bridge/mock_bridge/Mock_Bridge.go

bridge: ## Builds the Bridge instance.
	GOBIN=$(PWD)/build/bin go install ./cmd/bridge

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker:
	docker container prune -f
	docker build -t scroll_l1geth ../internal/docker/l1geth/
	docker build -t scroll_l2geth ../internal/docker/l2geth/
	docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ../ -f ./Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}

bump_l2geth:
	go get -v -u github.com/scroll-tech/go-ethereum/cmd/geth@zkrollup
	go mod tidy
