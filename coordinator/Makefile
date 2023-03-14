.PHONY: lint docker clean coordinator

IMAGE_NAME=coordinator
IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

coordinator: ## Builds the Coordinator instance.
	cd ../common/libzkp && bash build.sh
	go build -o $(PWD)/build/bin/coordinator ./cmd

mock-coordinator:
	go build -tags mock -o $(PWD)/build/bin/coordinator ./cmd

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	cd ../common/libzkp/lib/ && rm *.a *.so *.zip
	@rm -rf build/bin

docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/coordinator.Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}
