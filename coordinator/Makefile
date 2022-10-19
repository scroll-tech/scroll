.PHONY: lint docker clean coordinator

IMAGE_NAME=coordinator
IMAGE_VERSION=latest

coordinator: ## Builds the Coordinator instance.
	go build -o $(PWD)/build/bin/coordinator ./cmd

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker:
	docker container prune -f
	docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ../ -f ./Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}
