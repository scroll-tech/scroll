.PHONY: help fmt clippy test test-ci test-all

build:
	@cargo build --release -p libzkp-c
	@mkdir -p lib
	@cp -f ../../../../target/release/libzkp.so lib/

fmt:
	@cargo fmt --all -- --check

clean:
	@cargo clean --release -p libzkp -p libzkp-c -p l2geth
	@rm -f lib/libzkp.so

clippy:
	@cargo check --release --all-features
	@cargo clippy --release -- -D warnings
