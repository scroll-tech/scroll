.PHONY: lint docker clean coordinator

IMAGE_NAME=coordinator
IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

coordinator: ## Builds the Coordinator instance.
	go build -o $(PWD)/build/bin/coordinator ./cmd

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker:
	docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/coordinator.Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}
