.PHONY: lint
# NOTE: We use $$(CURDIR) instead of $$(PWD) throughout.
# CURDIR is Make's built-in variable that tracks the current Makefile's directory
# and respects 'make -C'. PWD is inherited from the shell and does NOT change with -C,
# which causes binaries to be built to wrong paths when invoked via make -C <subdir>.

IMAGE_NAME=db_cli
IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

db_cli:
	go build -o $(CURDIR)/build/bin/db_cli ./cmd

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(CURDIR)/...

lint: ## Lint the files - used for CI
	GOBIN=$(CURDIR)/build/bin go run ../build/lint.go

docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/${IMAGE_NAME}.Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}
