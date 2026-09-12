# Current zk Stack — Dated Facts

> **Purpose**: single home for version-sensitive facts (S3 prefixes, digests, codec thresholds, production stack identity). Other documents should **link here** instead of embedding these values. Every fact carries a **last-verified** date — when an upgrade lands, update this file and only this file.
>
> Last full review: **2026-09-11** (during the canary parallel-upgrade run, report: `testing_reports/canary-parallel-upgrade-2026-09-11.md`).

## Production (Scroll mainnet)

| Fact | Value | Last verified |
|---|---|---|
| Production guest | pre-v0.9.0 era guest (`79c1f8c` per `git_version` in production proofs) | 2026-09-11 |
| `latestVerifier[10]` wrapper | `0x808297224e86b1a6055B5F790a2cE07Ed611f955`, startBatch 0 | 2026-09-11 |
| Wrapper `verifierDigest1` | `0x00398b786b500ca759ca2de2aee9c73bd8e28f1c80b49e1c53bc060a9a649269` | 2026-09-11 |
| Bundle cadence | ~1 bundle / 1–2 h (slower on weekends/low activity; multi-hour pauses observed) | 2026-09-11 |
| Contract addresses | see [`contract-addresses.md`](contract-addresses.md) | 2026-05-31 |

How to re-verify the production stack before any shadow test: follow/GUIDE.md "Determining the Production zk Stack" (on-chain wrapper digests vs newest remote-proof `instances` bytes 384–448 vs `git_version`).

## Release under test (this branch)

| Fact | Value | Last verified |
|---|---|---|
| Guest | zkvm-prover `bf887150` (labelled v0.9.0 / circuit_version v0.14.0 in configs) | 2026-09-11 |
| S3 base | `https://circuit-release.s3.us-west-2.amazonaws.com/scroll-zkvm/releases/v0.9.0/` | 2026-09-11 |
| Bundle digest1 (canonical) | `0x006770fb0f71f20718b614c8c5d3fb39482f0527806e67b0ea00ab5508245655` | 2026-09-11 |

## S3 layout per release

| Release | Prefix | Digest encoding | Notes |
|---|---|---|---|
| v0.7.1 and earlier | `scroll-zkvm/releases/v0.7.1/` | — | |
| v0.8.0 | `scroll-zkvm/v0.8.0/` (no `/releases/`!) | **Montgomery** — convert to canonical before deploying | verifier + circuits split across prefixes |
| v0.9.0+ | `scroll-zkvm/releases/v0.9.0/` | **canonical** — use as-is | unified prefix; flat circuit layout `<base>/<circuit>/app.vmexe` + `agg_vk.bin` |

Details: [`bundle-digest-encoding.md`](bundle-digest-encoding.md).

## Codec / fork thresholds (mainnet)

| Fork | Codec | Mainnet blocks | Notes |
|---|---|---|---|
| galileo | V9 | any (e.g. ≥ 26,653,680) | pre-V10 |
| galileoV2 | V10 | **≥ 33,750,000** | current; older blocks fail "mismatched post-state root" (Trap 50) |

## Sizing reference (single-GPU, RTX 4090, halo2-gpu build)

| Metric | Value | Last verified |
|---|---|---|
| Chunk proof | ~2 min (first task +10–15 min witness fetch) | 2026-09-11 |
| Bundle proof end-to-end | ~10–15 min (STARK agg layers dominate; halo2 SNARK `Create EVM proof` ≈ 6 s) | 2026-09-11 |
| SNARK-phase GPU peak | ~8.8 GiB | 2026-09-11 |
| 1×4090 vs mainnet cadence | keeps pace with ~1.5×–2 h/bundle production incl. backlog | 2026-09-11 |
| 30-batch bundles | ~30–90 min per bundle | 2026-07 |

## Tooling versions that matter

| Tool | Version note | Last verified |
|---|---|---|
| anvil | 1.7.1 — silent death/stall observed twice under long fork load (Trap 44); state-file relaunch recovery works | 2026-09-11 |
| solc | ≥ 0.8.24 required (`--evm-version cancun`) | 2026-08 |
| CUDA | 12.8 works; nvcc 12.4 ICEs on OpenVM (`cp_gen_be.c` assertion) | 2026-08-25 |
