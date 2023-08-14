.PHONY: lint docker clean coordinator coordinator_skip_libzkp mock_coordinator

IMAGE_NAME=coordinator
IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

ifeq (4.3,$(firstword $(sort $(MAKE_VERSION) 4.3)))
	ZKEVM_VERSION=$(shell grep -m 1 "scroll-prover" ../common/libzkp/impl/Cargo.lock.new | cut -d "#" -f2 | cut -c-7)
	HALO2_VERSION=$(shell grep -m 1 "halo2.git" ../common/libzkp/impl/Cargo.lock.new | cut -d "#" -f2 | cut -c-7)
	OLD_ZKEVM_VERSION=$(shell grep -m 1 "scroll-prover" ../common/libzkp/impl/Cargo.lock.old | cut -d "#" -f2 | cut -c-7)
	OLD_HALO2_VERSION=$(shell grep -m 1 "halo2.git" ../common/libzkp/impl/Cargo.lock.old | cut -d "#" -f2 | cut -c-7)
else
	ZKEVM_VERSION=$(shell grep -m 1 "scroll-prover" ../common/libzkp/impl/Cargo.lock.new | cut -d "\#" -f2 | cut -c-7)
	HALO2_VERSION=$(shell grep -m 1 "halo2.git" ../common/libzkp/impl/Cargo.lock.new | cut -d "\#" -f2 | cut -c-7)
	OLD_ZKEVM_VERSION=$(shell grep -m 1 "scroll-prover" ../common/libzkp/impl/Cargo.lock.old | cut -d "\#" -f2 | cut -c-7)
	OLD_HALO2_VERSION=$(shell grep -m 1 "halo2.git" ../common/libzkp/impl/Cargo.lock.old | cut -d "\#" -f2 | cut -c-7)
endif

ZK_VERSION=${ZKEVM_VERSION}-${HALO2_VERSION}
OLD_ZK_VERSION=${OLD_ZKEVM_VERSION}-${OLD_HALO2_VERSION}

pre-upgrade-zk:
	cd ../common/libzkp/impl && cp Cargo.lock.old Cargo.lock && cargo clean && cargo build --release && cp ./target/release/libzkp.so ../interface/liboldzkp.so
	cp -r ../common/libzkp/interface ./internal/logic/verifier/lib
	find ../common | grep libzktrie.so | xargs -I{} cp {} ./internal/logic/verifier/lib/liboldzktrie.so

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

libzkp: pre-upgrade-zk
	cd ../common/libzkp/impl && cp Cargo.lock.new Cargo.lock && cargo clean && cargo build --release && cp ./target/release/libzkp.so ../interface/
	cp -r ../common/libzkp/interface ./internal/logic/verifier/lib
	find ../common | grep libzktrie.so | xargs -I{} cp {} ./internal/logic/verifier/lib

coordinator: libzkp ## Builds the Coordinator instance.
	go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION} -X scroll-tech/common/version.OldZkVersion=${OLD_ZK_VERSION}" -o $(PWD)/build/bin/coordinator ./cmd

coordinator_skip_libzkp:
	go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -o $(PWD)/build/bin/coordinator ./cmd

mock_coordinator: ## Builds the mocked Coordinator instance.
	go build -tags="mock_prover mock_verifier" -o $(PWD)/build/bin/coordinator ./cmd

test-verifier: libzkp
	go test -tags ffi -timeout 0 -v ./internal/logic/verifier

test-gpu-verifier: libzkp
	go test -tags="gpu ffi" -timeout 0 -v ./internal/logic/verifier

lint: ## Lint the files - used for CI
	cp -r ../common/libzkp/interface ./internal/logic/verifier/lib
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/${IMAGE_NAME}:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/coordinator.Dockerfile

docker_push:
	docker push scrolltech/${IMAGE_NAME}:${IMAGE_VERSION}