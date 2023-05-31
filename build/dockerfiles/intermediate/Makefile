GO_VERSION := 1.18
PYTHON_VERSION := 3.10
RUST_VERSION := nightly-2022-12-10

.PHONY: all go-alpine-builder rust-builder rust-alpine-builder go-rust-alpine-builder go-rust-builder py-runner

go-rust-builder:
	docker build -t scrolltech/go-rust-builder:latest -f go-rust-builder.Dockerfile ./
	docker image tag scrolltech/go-rust-builder:latest scrolltech/go-rust-builder:go-$(GO_VERSION)-rust-$(RUST_VERSION)

go-alpine-builder:
	docker build -t scrolltech/go-alpine-builder:latest -f go-alpine-builder.Dockerfile ./
	docker image tag scrolltech/go-alpine-builder:latest scrolltech/go-alpine-builder:$(GO_VERSION)

rust-builder:
	docker build -t scrolltech/rust-builder:latest -f rust-builder.Dockerfile ./
	docker image tag scrolltech/rust-builder:latest scrolltech/rust-builder:$(RUST_VERSION)

rust-alpine-builder:
	docker build -t scrolltech/rust-alpine-builder:latest -f rust-alpine-builder.Dockerfile ./
	docker image tag scrolltech/rust-alpine-builder:latest scrolltech/rust-alpine-builder:$(RUST_VERSION)

go-rust-alpine-builder:
	docker build -t scrolltech/go-rust-alpine-builder:latest -f go-rust-alpine-builder.Dockerfile ./
	docker image tag scrolltech/go-rust-alpine-builder:latest scrolltech/go-rust-alpine-builder:go-$(GO_VERSION)-rust-$(RUST_VERSION)

py-runner:
	docker build -t scrolltech/py-runner:latest -f py-runner.Dockerfile ./
	docker image tag scrolltech/py-runner:latest scrolltech/py-runner:$(PYTHON_VERSION)

all: go-alpine-builder rust-builder rust-alpine-builder go-rust-alpine-builder go-rust-builder py-runner

publish:
	docker push scrolltech/go-alpine-builder:latest
	docker push scrolltech/go-alpine-builder:$(GO_VERSION)
	docker push scrolltech/rust-builder:latest
	docker push scrolltech/rust-builder:$(RUST_VERSION)
	docker push scrolltech/rust-alpine-builder:latest
	docker push scrolltech/rust-alpine-builder:$(RUST_VERSION)
	docker push scrolltech/go-rust-alpine-builder:latest
	docker push scrolltech/go-rust-alpine-builder:go-$(GO_VERSION)-rust-$(RUST_VERSION)
	docker push scrolltech/go-rust-builder:latest
	docker push scrolltech/go-rust-builder:go-$(GO_VERSION)-rust-$(RUST_VERSION)
	docker push scrolltech/py-runner:latest
	docker push scrolltech/py-runner:$(PYTHON_VERSION)
