.PHONY: help fmt clippy test test-ci test-all

build:
	@cargo build --release

fmt:
	@cargo fmt --all -- --check

clippy:
	@cargo check --all-features
	@cargo clippy --release -- -D warnings
