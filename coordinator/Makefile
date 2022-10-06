.PHONY: lint docker clean coordinator

IMAGE_NAME=coordinator
IMAGE_VERSION=latest

coordinator: ## Builds the Coordinator instance.
	GOBIN=$(PWD)/build/bin go install ./cmd/coordinator

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker:
	docker container prune -f
	docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ../ -f ./Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}

bump_l2geth:
	go get -v -u github.com/scroll-tech/go-ethereum/cmd/geth@zkrollup
	go mod tidy
