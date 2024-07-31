.PHONY: lint docker clean coordinator coordinator_skip_libzkp mock_coordinator

IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

ifeq (4.3,$(firstword $(sort $(MAKE_VERSION) 4.3)))
	ZKEVM_VERSION=$(shell grep -m 1 "zkevm-circuits" ../common/libzkp/impl/Cargo.lock | cut -d "#" -f2 | cut -c-7)
	HALO2_VERSION=$(shell grep -m 1 "halo2.git" ../common/libzkp/impl/Cargo.lock | cut -d "#" -f2 | cut -c-7)
else
	ZKEVM_VERSION=$(shell grep -m 1 "zkevm-circuits" ../common/libzkp/impl/Cargo.lock | cut -d "\#" -f2 | cut -c-7)
	HALO2_VERSION=$(shell grep -m 1 "halo2.git" ../common/libzkp/impl/Cargo.lock | cut -d "\#" -f2 | cut -c-7)
endif

ZK_VERSION=${ZKEVM_VERSION}-${HALO2_VERSION}

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

libzkp:
	cd ../common/libzkp/impl && cargo clean && cargo build --release && cp ./target/release/libzkp.so ../interface/
	rm -rf ./internal/logic/verifier/lib && cp -r ../common/libzkp/interface ./internal/logic/verifier/lib

coordinator_api: libzkp ## Builds the Coordinator api instance.
	go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -o $(PWD)/build/bin/coordinator_api ./cmd/api

coordinator_cron:
	go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -o $(PWD)/build/bin/coordinator_cron ./cmd/cron

coordinator_tool:
	go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -o $(PWD)/build/bin/coordinator_tool ./cmd/tool

coordinator_api_skip_libzkp:
	go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -o $(PWD)/build/bin/coordinator_api ./cmd/api

mock_coordinator_api: ## Builds the mocked Coordinator instance.
	go build -tags="mock_prover mock_verifier" -o $(PWD)/build/bin/coordinator_api ./cmd/api

mock_coordinator_cron: ## Builds the mocked Coordinator instance.
	go build -tags="mock_prover mock_verifier" -o $(PWD)/build/bin/coordinator_cron ./cmd/cron

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
	DOCKER_BUILDKIT=1 docker build -t scrolltech/coordinator-api:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/coordinator-api.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/coordinator-cron:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/coordinator-cron.Dockerfile

docker_push:
	docker push scrolltech/coordinator-api:${IMAGE_VERSION}
	docker push scrolltech/coordinator-cron:${IMAGE_VERSION}
