#!/bin/bash

# Alas, the openvm CLI does not offer `clippy` as a subcommand,
# so we have to use `cargo` directly.  The options are copy-and-pasted from
# `openvm`'s `build` command.
clippycmd="cargo +nightly-2025-02-14 clippy \
  --target riscv32im-risc0-zkvm-elf \
  -Z build-std=alloc,core,proc_macro,panic_abort,std \
  -Z build-std-features=compiler-builtins-mem \
  --all-features \
  -- -D warnings"

cd crates/circuits/chunk-circuit; eval "$clippycmd"; cd ./../../..
cd crates/circuits/batch-circuit; eval "$clippycmd"; cd ./../../..
cd crates/circuits/bundle-circuit; eval "$clippycmd"; cd ./../../..
