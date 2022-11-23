.PHONY: lint

IMAGE_NAME=db_cli
IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

db_cli:
	go build -o $(PWD)/build/bin/db_cli ./cmd

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/${IMAGE_NAME}.Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}
