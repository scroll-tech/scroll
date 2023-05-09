.PHONY: lint docker clean coordinator

IMAGE_NAME=coordinator
IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

ifeq (4.3,$(firstword $(sort $(MAKE_VERSION) 4.3)))
	ZK_VERSION=$(shell grep -m 1 "scroll-zkevm" ../common/libzkp/impl/Cargo.lock | cut -d "#" -f2 | cut -c-7)
else
	ZK_VERSION=$(shell grep -m 1 "scroll-zkevm" ../common/libzkp/impl/Cargo.lock | cut -d "\#" -f2 | cut -c-7)
endif

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

libzkp:
	cd ../common/libzkp/impl && cargo build --release && cp ./target/release/libzkp.a ../interface/
	rm -rf ./verifier/lib && cp -r ../common/libzkp/interface ./verifier/lib
	find ../common | grep libzktrie.so | xargs -I{} cp {} ./verifier/lib/

coordinator: libzkp ## Builds the Coordinator instance.
	go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -o $(PWD)/build/bin/coordinator ./cmd

test-verifier: libzkp
	go test -tags ffi -timeout 0 -v ./verifier

test-gpu-verifier: libzkp
	go test -tags="gpu ffi" -timeout 0 -v ./verifier

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/coordinator.Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}
