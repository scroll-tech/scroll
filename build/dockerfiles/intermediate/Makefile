GO_VERSION := 1.18
PYTHON_VERSION := 3.10
RUST_VERSION := nightly-2022-08-23

.PHONY: all go-builder rust-builder rust-alpine-builder go-rust-builder py-runner

go-builder:
	docker build -t scrolltech/go-builder:latest -f go-builder ./
	docker image tag scrolltech/go-builder:latest scrolltech/go-builder:$(GO_VERSION)

rust-builder:
	docker build -t scrolltech/rust-builder:latest -f rust-builder ./
	docker image tag scrolltech/rust-builder:latest scrolltech/rust-builder:$(RUST_VERSION)

rust-alpine-builder:
	docker build -t scrolltech/rust-alpine-builder:latest -f rust-alpine-builder ./
	docker image tag scrolltech/rust-alpine-builder:latest scrolltech/rust-alpine-builder:$(RUST_VERSION)

go-rust-builder:
	docker build -t scrolltech/go-rust-builder:latest -f go-rust-builder ./
	docker image tag scrolltech/go-rust-builder:latest scrolltech/go-rust-builder:go-$(GO_VERSION)-rust-$(RUST_VERSION)

py-runner:
	docker build -t scrolltech/py-runner:latest -f py-runner ./
	docker image tag scrolltech/py-runner:latest scrolltech/py-runner:$(PYTHON_VERSION)

all: go-builder rust-builder rust-alpine-builder go-rust-builder py-runner

publish:
	docker push scrolltech/go-builder:latest
	docker push scrolltech/go-builder:$(GO_VERSION)
	docker push scrolltech/rust-builder:latest
	docker push scrolltech/rust-builder:$(RUST_VERSION)
	docker push scrolltech/rust-alpine-builder:latest
	docker push scrolltech/rust-alpine-builder:$(RUST_VERSION)
	docker push scrolltech/go-rust-builder:latest
	docker push scrolltech/go-rust-builder:go-$(GO_VERSION)-rust-$(RUST_VERSION)
	docker push scrolltech/py-runner:latest
	docker push scrolltech/py-runner:$(PYTHON_VERSION)
