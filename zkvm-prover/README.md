# Scroll zkVM

**zkVM-based Circuits (Guest Programs) with a complete Scroll Prover implementation**

## Repository

This repository contains the following member crates:

- [scroll-zkvm-circuit-types](./crates/circuits/types): Primitive and Common types used by the circuits
- [scroll-zkvm-chunk-circuit](./crates/circuits/chunk-circuit): Circuit for verification of a Scroll [chunk](TODO:doc)
- [scroll-zkvm-batch-circuit](./crates/circuits/batch-circuit): Circuit for verification of a Scroll [batch](TODO:doc)
- [scroll-zkvm-bundle-circuit](./crates/circuits/bundle-circuit): Circuit for verification of a Scroll [bundle](TODO:doc)
- [scroll-zkvm-prover](./crates/prover): Implementation for a Scroll Prover
- [scroll-zkvm-verifier](./crates/verifier): Implementation for a Verifier-only mode
- [scroll-zkvm-integration](./crates/integration): Integration tests for the Scroll Prover

## Overview

The Scroll zkVM Circuits are [openvm](https://book.openvm.dev/) based Guest Programs.

The [prover](./crates/prover) crate offers a minimalistic API for setting up, generating and verifying proofs for Scroll's zk-rollup.

For a deeper dive into our implementation, please refer the [interfaces](./docs/interfaces.md) doc.

## Testing

For more commands please refer the [Makefile](./Makefile).

### Build Guest Programs

In case you have made any changes to the guest programs, it is important to build them before running the tests.

```shell
$ make build-guest
```

Upon building the guest programs, the child commitments in [batch-circuit](./crates/circuits/batch-circuit/src/child_commitments.rs) and [bundle-circuit](./crates/circuits/bundle-circuit/src/child_commitments.rs) will be overwritten by `build-guest`.

### End-to-end tests for chunk-prover

```shell
$ RUST_MIN_STACK=16777216 make test-single-chunk
```

### End-to-end tests for batch-prover

```shell
$ RUST_MIN_STACK=16777216 make test-e2e-batch
```

### End-to-end tests for bundle-prover

```shell
$ RUST_MIN_STACK=16777216 make test-e2e-bundle
```

*Note*: Configure `RUST_LOG=debug` for debug logs or `RUST_LOG=none,scroll_zkvm_prover=debug` for logs specifically from the `scroll-zkvm-prover` crate.

## Usage of Prover API

### Dependency

Add the following dependency in your `Cargo.toml`:

```toml
[dependencies]
scroll-zkvm-prover = { git = "https://github.com/scroll-tech/zkvm-prover", branch = "master" }
```

### Chunk Prover

Prover capable of generating STARK proofs for a Scroll [chunk](TODO:doc):

```rust
use std::path::Path;

use scroll_zkvm_prover::{ChunkProver, task::ChunkProvingTask};

// Paths to the application exe and application config.
let path_exe = Path::new("./path/to/app.vmexe");
let path_app_config = Path::new("./path/to/openvm.toml");

// Optional directory to cache generated proofs on disk.
let cache_dir = Path::new("./path/to/cache/proofs");

// Setup prover.
let prover = ChunkProver::setup(&path_exe, &path_app_config, Some(&cache_dir))?;

// Proving task of a chunk with 3 blocks.
let block_witnesses = vec![
    sbv::primitives::types::BlockWitness { /* */ },
    sbv::primitives::types::BlockWitness { /* */ },
    sbv::primitives::types::BlockWitness { /* */ },
];
let task = ChunkProvingTask { block_witnesses };

// Generate a proof.
let proof = prover.gen_proof(&task)?;

// Verify proof.
prover.verify_proof(&proof)?;
```

### Batch Prover

Prover capable of generating STARK proofs for a Scroll [batch](TODO:doc):

```rust
use std::path::Path;

use scroll_zkvm_prover::{BatchProver, task::BatchProvingTask};

// Paths to the application exe and application config.
let path_exe = Path::new("./path/to/app.vmexe");
let path_app_config = Path::new("./path/to/openvm.toml");

// Optional directory to cache generated proofs on disk.
let cache_dir = Path::new("./path/to/cache/proofs");

// Setup prover.
let prover = BatchProver::setup(&path_exe, &path_app_config, Some(&cache_dir))?;

// Task that proves batching of a number of chunks.
let task = BatchProvingTask {
    chunk_proofs, // chunk proofs being aggregated in this batch
    batch_header, // the header for the batch
    blob_bytes,   // the EIP-4844 blob that makes this batch data available on L1
};

// Generate a proof.
let proof = prover.gen_proof(&task)?;

// Verify proof.
prover.verify_proof(&proof)?;
```

### Bundle Prover

Prover capable of generating EVM-verifiable halo2-based SNARK proofs for a Scroll [bundle](TODO:doc):

```rust
use std::path::Path;

use scroll_zkvm_prover::{BundleProver, task::BundleProvingTask};

// Paths to the application exe and application config.
let path_exe = Path::new("./path/to/app.vmexe");
let path_app_config = Path::new("./path/to/openvm.toml");

// Optional directory to cache generated proofs on disk.
let cache_dir = Path::new("./path/to/cache/proofs");

// Setup prover.
//
// The Bundle Prover's setup also looks into $HOME/.openvm for halo2-based setup parameters.
let prover = BundleProver::setup(&path_exe, &path_app_config, Some(&cache_dir))?;

// Task that proves batching of a number of chunks.
let task = BundleProvingTask {
    batch_proofs, // batch proofs being aggregated in this bundle
};

// Generate a proof.
let evm_proof = prover.gen_proof_evm(&task)?;

// Verify proof.
prover.verify_proof_evm(&evm_proof)?;
```
