.PHONY: clean build test docker

IMAGE_NAME=prover-stats-api
IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

all: build

build:
	GOBIN=$(PWD)/build/bin go build -o $(PWD)/build/bin/prover-stats-api ./cmd

clean: ## Empty out the bin folder
	@rm -rf build/bin

swag:
	swag init -g ./cmd/main.go

test:
	go test -v $(PWD)/...

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/prover-stats-api.Dockerfile

.PHONY: all
