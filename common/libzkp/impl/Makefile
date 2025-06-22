.PHONY: help fmt clippy test test-ci test-all

build:
	@cargo build --release
	@mkdir -p ../lib
	@cp -f target/release/libzkp.so ../lib/

fmt:
	@cargo fmt --all -- --check

clean:
	@rm -f ../lib/libzkp.so

clippy:
	@cargo check --all-features
	@cargo clippy --release -- -D warnings
