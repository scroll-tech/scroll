.PHONY: lint docker clean coordinator mock_coordinator

IMAGE_NAME=coordinator
IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

ifeq (4.3,$(firstword $(sort $(MAKE_VERSION) 4.3)))
	ZKEVM_VERSION=$(shell grep -m 1 "scroll-prover" ../common/libzkp/impl/Cargo.lock | cut -d "#" -f2 | cut -c-7)
	HALO2_VERSION=$(shell grep -m 1 "halo2.git" ../common/libzkp/impl/Cargo.lock | cut -d "#" -f2 | cut -c-7)
else
	ZKEVM_VERSION=$(shell grep -m 1 "scroll-prover" ../common/libzkp/impl/Cargo.lock | cut -d "\#" -f2 | cut -c-7)
	HALO2_VERSION=$(shell grep -m 1 "halo2.git" ../common/libzkp/impl/Cargo.lock | cut -d "\#" -f2 | cut -c-7)
endif

ZK_VERSION=${ZKEVM_VERSION}-${HALO2_VERSION}

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

libzkp:
	cd ../common/libzkp/impl && cargo clean && cargo build --release && cp ./target/release/libzkp.so ../interface/
	rm -rf ./verifier/lib && cp -r ../common/libzkp/interface ./verifier/lib
	find ../common | grep libzktrie.so | xargs -I{} cp {} ./verifier/lib/

coordinator: libzkp ## Builds the Coordinator instance.
	go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -o $(PWD)/build/bin/coordinator ./cmd

mock_coordinator: ## Builds the mocked Coordinator instance.
	go build -tags="mock_prover mock_verifier" -o $(PWD)/build/bin/coordinator ./cmd

test-verifier: libzkp
	go test -tags ffi -timeout 0 -v ./verifier

test-gpu-verifier: libzkp
	go test -tags="gpu ffi" -timeout 0 -v ./verifier

lint: ## Lint the files - used for CI
	cp -r ../common/libzkp/interface ./verifier/lib
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/coordinator.Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}
