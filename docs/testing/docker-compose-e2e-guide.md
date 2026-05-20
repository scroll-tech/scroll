# Docker Compose E2E Testing Guide

This document describes how to run the end-to-end (E2E) proving pipeline in a **production-like containerized setup** using Docker Compose. Unlike the bare-metal E2E test (Level 4 in `openvm-upgrade-testing-guide.md`), this setup:

- Runs all components in Docker containers.
- Places **Coordinator Proxy** between the prover and coordinator (same as production).
- Uses the same Docker images and base layers as the production deployment.

---

## Architecture

```
┌─────────────┐      ┌─────────────────────┐      ┌─────────────────┐
│   Prover    │─────>│ Coordinator Proxy   │─────>│ Coordinator API │
│  (Docker)   │      │  (port 8590)        │      │  (port 8390)    │
└─────────────┘      └─────────────────────┘      └─────────────────┘
                              │                            │
                              └────────────┬───────────────┘
                                           │
                                    ┌──────▼──────┐
                                    │  PostgreSQL │
                                    │ (Docker)    │
                                    └─────────────┘
```

- **PostgreSQL**: Existing `local_postgres` container (from `tests/prover-e2e/docker-compose.yml`).
- **Coordinator API**: Scroll L2 coordinator with `libzkp.so` and verifier assets.
- **Coordinator Proxy**: HTTP reverse proxy that load-balances across upstream coordinators. In this test there is only one upstream, but the proxy still handles auth, session management, and task routing exactly as in production.
- **Prover**: zkVM prover with GPU support, connecting to the proxy instead of directly to the coordinator.

---

## Prerequisites

1. Docker & Docker Compose installed.
2. NVIDIA Container Toolkit configured (for GPU prover).
3. `local_postgres` already running with imported block data:
   ```bash
   cd tests/prover-e2e
   ln -snf mainnet-galileoV2 conf
   make all
   ```
4. Halo2 SRS parameters downloaded to `~/.openvm/params/` (required for bundle SNARK):
   ```bash
   ls ~/.openvm/params/
   # kzg_bn254_23.srs
   # kzg_bn254_24.srs
   ```
   These are large KZG trusted-setup files used by OpenVM's Halo2 wrapper. If missing, the prover will panic during bundle proof generation.

---

## Building Images

### 1. Coordinator API

```bash
docker build \
  -f build/dockerfiles/coordinator-api.Dockerfile \
  -t scrolltech/coordinator-api:e2e-test .
```

Build stages:
- `chef` / `planner` / `zkp-builder`: Caches Rust dependencies with `cargo chef`, builds `libzkp.so`.
- `base` / `builder`: Downloads Go modules, builds `coordinator_api` binary with CGO linking against the `.so` from the Rust stage.
- Final `ubuntu:20.04` stage: Copies the binary and `libzkp.so` into a minimal runtime image.

### 2. Coordinator Proxy

```bash
docker build \
  -f build/dockerfiles/coordinator-proxy.Dockerfile \
  -t scrolltech/coordinator-proxy:e2e-test .
```

This is a pure Go build; no Rust/CGO is involved.

### 3. Prover

The prover is built **outside** the container (same as production CI) because GPU-enabled builds require:
- A CUDA development environment (`nvcc`, CUDA headers).
- Access to `nvidia-smi` during the `cuda-builder` crate's compile-time architecture detection.

Standard build (matches `devops` repo practice):

```bash
cd zkvm-prover
make prover        # GPU version
# or
make prover_cpu    # CPU version
```

Then package the pre-built binary into a runtime image:

```bash
docker build \
  -f build/dockerfiles/prover.Dockerfile \
  -t scrolltech/prover:e2e-test .
```

Runtime image:
- Base: `nvidia/cuda:12.9.1-runtime-ubuntu22.04`
- Installs `solc` 0.8.24 (needed for EVM proof generation)
- Copies `target/release/prover`

> **Why not build inside Docker?**  
> The upstream `openvm-cuda-builder` crate runs `nvidia-smi` at compile time to detect the GPU architecture (`sm_86`, etc.). Standard `docker build` does not expose the GPU to the build context, causing the architecture detection to fail. Production pipelines (see `git@github.com:scroll-tech/devops.git`) solve this by building on GPU-equipped CI runners and then copying the artifact into a slim runtime image.

---

## Configuration

All configs live in `tests/prover-e2e/docker-e2e/conf/`:

| File | Service | Key Points |
|------|---------|------------|
| `coordinator-api.json` | Coordinator API | `assets_path: "assets_v2"`, DB connects to `local_postgres:5432`, `chunk_collection_time_sec: 3600` |
| `coordinator-proxy.json` | Coordinator Proxy | Single upstream `coordinator-api:8390`, `compatible_mode: true`, verifier list must include target fork (e.g. `galileoV2`) or prover login is rejected |
| `prover.json` | Prover | `base_url: "http://coordinator-proxy:8590"`, circuits point to `galileov2` S3 assets |

> **Proxy verifier list:** The proxy's `verifier.verifiers` array cannot be empty. Even though the proxy does not verify proofs itself, it validates the prover's declared fork support during login. If empty, the prover receives `JWTCommonErr: invalid prover prover_version`.

---

## Docker Compose

File: `tests/prover-e2e/docker-e2e/docker-compose.yml`

Services:
- `coordinator-api`: ports `8390:8390`, mounts config + `assets_v2` + `genesis.json`.
- `coordinator-proxy`: ports `8590:8590`, mounts config, depends on `coordinator-api`.
- `prover`: mounts config + `testset.json` + `.work` cache + `~/.openvm/params`, GPU reservation, `RUST_MIN_STACK=16777216`.

Run:

```bash
cd tests/prover-e2e/docker-e2e
docker compose up -d coordinator-api coordinator-proxy
docker compose up -d prover
```

Follow logs:

```bash
docker logs -f prover
docker logs -f coordinator-api
docker logs -f coordinator-proxy
```

---

## Troubleshooting

### `stack overflow` in prover container

**Symptom:** `thread 'tokio-rt-worker' has overflowed its stack`

**Fix:** Set `RUST_MIN_STACK=16777216` (16 MiB) in the prover container environment. The default Rust thread stack is too small for OpenVM's deep recursion during Halo2 key generation.

### `kzg_bn254_23.srs does not exist`

**Symptom:** Prover panics during bundle proof with `Params file "/root/.openvm/params/kzg_bn254_23.srs" does not exist`.

**Fix:** Mount the host's `~/.openvm/params` directory into the container at `/root/.openvm/params`.

### `No such file or directory` in chunk verifier setup

**Symptom:** Coordinator API crashes on startup with `Setting up chunk verifier: No such file or directory`.

**Fix:** The `assets_v2` directory was not mounted correctly. Ensure the host path in `docker-compose.yml` resolves to `coordinator/build/bin/assets_v2` (contains `verifier.bin`, `root_verifier_vk`, `openVmVk.json`).

### `bind: address already in use` for port 8390/8590

**Fix:** `docker rm -f coordinator-api coordinator-proxy` before recreating.

---

## Verification

After the prover finishes (`All done!`), confirm in the coordinator API logs that all tasks are `status=verified`:

```bash
docker logs coordinator-api | grep "status=verified"
```

Expected output:
- 4 chunk tasks verified
- 2 batch tasks verified
- 1 bundle task verified
