include ../build/common.mk

.PHONY: lint docker clean coordinator coordinator_skip_libzkp mock_coordinator libzkp

IMAGE_VERSION=latest
REPO_ROOT_DIR=./..
LIBZKP_PATH=./internal/logic/libzkp/lib/$(LIB_ZKP_NAME)

ifeq (4.3,$(firstword $(sort $(MAKE_VERSION) 4.3)))
	ZKVM_VERSION=$(shell grep -m 1 "zkvm-prover?" ../Cargo.lock | cut -d "#" -f2 | cut -c-7)
	OPENVM_VERSION=$(shell grep -m 1 "openvm.git" ../Cargo.lock | cut -d "#" -f2 | cut -c-7)
else
	ZKVM_VERSION=$(shell grep -m 1 "zkvm-prover?" ../Cargo.lock | cut -d "\#" -f2 | cut -c-7)
	OPENVM_VERSION=$(shell grep -m 1 "openvm.git" ../Cargo.lock | cut -d "\#" -f2 | cut -c-7)
endif

ZK_VERSION=${ZKVM_VERSION}-${OPENVM_VERSION}

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

$(LIBZKP_PATH):
	$(MAKE) -C ./internal/logic/libzkp build

clean_libzkp:
	$(MAKE) -C ./internal/logic/libzkp clean

libzkp: clean_libzkp $(LIBZKP_PATH)

coordinator_api: $(LIBZKP_PATH) ## Builds the Coordinator api instance.
	go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -o $(PWD)/build/bin/coordinator_api ./cmd/api
	$(call macos_codesign,$(PWD)/build/bin/coordinator_api)

coordinator_cron:
	go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -o $(PWD)/build/bin/coordinator_cron ./cmd/cron

coordinator_tool:
	go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -o $(PWD)/build/bin/coordinator_tool ./cmd/tool

localsetup: coordinator_api ## Local setup: build coordinator_api, copy config, and setup releases
	mkdir -p build/bin/conf
	@echo "Copying configuration files..."
	@if [ -f "$(PWD)/conf/config.template.json" ]; then \
		SRC="$(PWD)/conf/config.template.json"; \
	else \
		SRC="$(CURDIR)/conf/config.json"; \
	fi; \
	cp -fL "$$SRC" "$(CURDIR)/build/bin/conf/config.template.json"
	@echo "Setting up releases..."
	cd $(CURDIR)/build && bash setup_releases.sh


#coordinator_api_skip_libzkp:
#	go build -ldflags "-X scroll-tech/common/version.ZkVersion=${ZK_VERSION}" -o $(PWD)/build/bin/coordinator_api ./cmd/api

mock_coordinator_api: ## Builds the mocked Coordinator instance.
	go build -tags="mock_prover mock_verifier" -o $(PWD)/build/bin/coordinator_api ./cmd/api

mock_coordinator_cron: ## Builds the mocked Coordinator instance.
	go build -tags="mock_prover mock_verifier" -o $(PWD)/build/bin/coordinator_cron ./cmd/cron

test-verifier: $(LIBZKP_PATH)
	go test -tags ffi -timeout 0 -v ./internal/logic/verifier

test-gpu-verifier: $(LIBZKP_PATH)
	go test -tags="gpu ffi" -timeout 0 -v ./internal/logic/verifier

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go
	cd ../ && cargo fmt --all -- --check && cargo clippy --release -- -D warnings

clean: ## Empty out the bin folder
	@rm -rf build/bin

docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/coordinator-api:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/coordinator-api.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/coordinator-cron:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/coordinator-cron.Dockerfile

docker_push:
	docker push scrolltech/coordinator-api:${IMAGE_VERSION}
	docker push scrolltech/coordinator-cron:${IMAGE_VERSION}
