.PHONY: lint docker clean coordinator

IMAGE_NAME=coordinator
IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

coordinator: ## Builds the Coordinator instance.
	cd ../common/zkp/rust && cargo build --release && cp ./target/release/libzkp.a ../lib/
	cp -r ../common/zkp/lib ./verifier/
	go build -o $(PWD)/build/bin/coordinator ./cmd

gpu-coordinator:
	cd ../common/zkp/rust && cargo build --release && cp ./target/release/libzkp.a ../lib/
	cp -r ../common/zkp/lib ./verifier/
	go build -tags gpu -o $(PWD)/build/bin/coordinator ./cmd

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker:
	docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/coordinator.Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}
