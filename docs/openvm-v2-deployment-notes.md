# OpenVM v2 / Guest v0.9.0 — Production Deployment Notes

**Date:** 2026-09-16
**Author:** Zhang Zhuo
**Status:** upgrade in flight — code PR open, ops PRs open, not yet deployed
**Scope:** everything an operator needs to deploy the v0.9.0 (OpenVM v2) prover/coordinator
stack to mainnet, distilled from the shadow canary tests and the PR review rounds.

> Version-sensitive facts (production stack identity, digests) are canonically tracked in
> [`tests/shadow-testing/docs/CURRENT-STACK.md`](../tests/shadow-testing/docs/CURRENT-STACK.md).
> This document is a dated runbook snapshot; if the two diverge, CURRENT-STACK.md wins.

---

## 1. Component map — which repo/PR carries what

| Piece | Where | State (2026-09-16) |
|---|---|---|
| Code: prover / coordinator / libzkp, wire format, version gate | `scroll-tech/scroll` PR **#1816** (`feat/zkvm-v0.9.0-openvm2`) | open, review blockers addressed |
| Circuits / guest | `scroll-tech/zkvm-prover` `bf887150` = tag `v0.9.0` + 1 commit (halo2-gpu, #260). master HEAD == `bf887150`; lispc offered to tag **v0.9.1** at it | tag pending |
| Coordinator deploy: verifier asset download | `sre-helm-charts` PR **#3520** (parameter-downloader, adds `agg_vk.bin`) | open |
| Coordinator deploy: image + config | `sre-helm-charts` PR **#3521** (`v4.8.0-rc.1`, `min_prover_version: v4.8.0`, assets `openvm-v0.9.0`) | open |
| Prover deploy: operator config template | `devops` PR **#1181** (ansible `galileo-v2-config.json.template`) | open |
| Prover image build | `devops` workflow `prover-image-build.yml` | **caveats, see §5** |
| On-chain verifier switch (MVRV `updateVerifier`) | contract ops, outside these PRs | — |

Production provers are external/IDC operators deployed via
`devops/ansible/prover-deploy/prover-auto-deploy`; there is **no prover helm chart**
(only `prover-stats-api`), so prover config changes ship through ansible + operator comms.

## 2. S3 layout (`circuit-release` bucket) — the facts that bite

- **v0.9.0 circuits**: `releases/v0.9.0/{chunk,batch,bundle}/` — **flat** dirs holding
  `app.vmexe`, `openvm.toml`, `agg_vk.bin`.
- **v0.9.0 verifier assets**: `releases/v0.9.0/verifier/` — `verifier.bin`,
  `root_verifier_vk`, `openVmVk.json`, `agg_vk.bin` (the last one was missing until
  2026-09-16; it is a server-side copy of `batch/agg_vk.bin`).
- **Legacy**: circuits are vk-keyed (`releases/<fork>/<type>/<vk>/` archives for
  feynman/galileo — no `agg_vk.bin` exists for them and none can be produced cheaply);
  legacy verifier assets live at `<ver>/verifier/` **without** the `releases/` segment.
  v0.8.0 is published in **both** layouts.
- `AssetsLocationData::gen_asset_url` (prover-bin) builds `<base_url>/<type>/<vk>/` by
  default — so for the flat v0.9.0 layout **`asset_detours` is mandatory**, not optional.
- Reads are anonymous-public. Writes require `s3-circuit-release-admin`;
  `prover_stat_data_user` has **zero** permissions on this bucket (its GetObject is denied
  even on public objects) — don't waste time with it.
- `SCROLL_ZKVM_VERSION` convention is a **bare version** (`v0.9.0`) everywhere;
  `coordinator/build/setup_releases.sh` probes `releases/<ver>/` vs `<ver>/` and picks
  whichever layout exists.

## 3. Prover side — config, params, hardware

Mandatory config changes per operator (copy-ready values in
`zkvm-prover/config.json.template` on PR #1816):

- `child_circuit_vks`: `{1: <chunk VK>, 2: <batch VK>}` — without it **every**
  batch/bundle task fails with `missing child circuit vk`. (New fail-fast guard: a
  child VK equal to the parent VK now errors instead of deadlocking the prover.)
- `asset_detours`: chunk/batch/bundle VK → the three flat release dirs.
- `base_url` → `releases/v0.9.0/`.
- `sdk_config` needs nothing new (scroll-proving-sdk unchanged by #1816).

Runtime facts:

- **SRS unchanged**: `~/.openvm/params/kzg_bn254_{22,23,24}.srs` are the same files as
  v0.8.0. Existing hosts already have them; fresh hosts/containers must pre-stage them —
  the failure only surfaces at the **first bundle proof**, hours in (shadow-testing
  Trap 12). The ansible `docker-compose.yml.template` mount
  (`/home/scroll/.openvm` → `/openvm/params`) still works unchanged.
- **GPU**: halo2-gpu bundle SNARK peaks ~8.8 GiB VRAM; 24 GB-class GPUs suffice
  (validated on 4090s in the shadow canary run, including a mixed chunk+batch+bundle
  process after the deferral VRAM fixes).
- Bundle digests change with the new guest: v0.9.0 digest1 `0x006770fb…5655`
  (production v0.8.0: `0x00398b78…9269`) — use this to verify a prover is really
  producing new-guest proofs.

## 4. Prover image — build caveats (devops `prover-image-build.yml`)

- Must be built from post-#1816 code → version string `v4.8.0-<gitrev>-bf88715-`
  (trailing dash = empty plonky3 segment, same shape as the current production tag).
  Provisional tag from PR #1816 HEAD: **`v4.8.0-8cc4b0dd-bf88715-`** — replace with the
  final tag after merge + image build.
- **`init-repos.sh` pins `SCROLL_COMMIT=c4e1641`** — it will silently build the OLD code
  unless bumped to the merge commit. The workflow's `branch` input only selects the
  devops branch, not the scroll code.
- The workflow runs `make prover` (`--features cuda`) = **CPU bundle SNARK**
  (~33 min/bundle). For GPU SNARK (~6 s) it must build `make prover_halo2gpu`.
- Runtime image needs solc **0.8.24** (monorepo `build/dockerfiles/prover.Dockerfile`
  is the reference; the devops Dockerfile still had 0.8.19 at time of writing).

## 5. Coordinator side

- `min_prover_version: v4.8.0` — provers older than v4.8.0 are **rejected at login**
  (deliberate: the v0.9.0 `StarkProof` wire format — `proof`/`user_pvs_proof` — is
  incompatible; old provers would otherwise accept tasks, fail verification, and get
  penalized).
- `agg_vk.bin` (batch circuit aggregation VK) is **mandatory** in the verifier assets
  dir: the coordinator **refuses to start** without it (deliberate fail-fast so an
  assets misconfiguration can't be misreported as prover failure).
  `setup_releases.sh` / helm #3520 both fetch it.
- Dead forks removed: feynman (openvm_13) / galileo entries are gone from
  `coordinator/conf/config.json` and the prover config template; the pre-v0.9.0
  `univ_task_compatibility_fix` path and the unused `OpenVMProof` type are deleted.
- Workspace version bumped 4.7.12 → **4.8.0** (`Cargo.toml`), Go tag `v4.7.16 → v4.8.0`.

## 6. Deploy sequencing

1. Merge scroll #1816 (optionally after zkvm-prover `v0.9.1` tag exists and the rev is
   switched from the raw commit to the tag).
2. Build + publish the prover image (after fixing §4 caveats); record the final tag.
3. **Drain or reset in-flight v0.8.0 batch/bundle tasks** — stored pre-v0.9.0 proofs do
   not deserialize under the new wire format.
4. Coordinator: deploy helm **#3520 first** (populates `/verifier/openvm-v0.9.0/`
   including `agg_vk.bin`), then **#3521**. Coordinator start-up validates the assets.
5. Provers: roll with the new image + config (ansible template from devops #1181; in
   `deployment/mainnet-galileov2.yaml` set the final `docker_tag` and bump
   `deploy_version` 5 → 6 — deliberately manual, noted in the PR).
   Coordinator and provers must switch **together** — the version gate locks out
   stragglers by design.
6. On-chain: switch the verifier routing via MVRV `updateVerifier` (contract ops).
7. Post-checks: prover logins report `v4.8.0-…`; first new bundle's digest1 is
   `0x006770fb…5655`; bundle SNARK phase is seconds, not tens of minutes (if it's
   minutes, the image was built without `halo2-gpu`).

## 7. Rollback

- Coordinator: helm values back to image `v4.7.17`, `min_prover_version: v4.5.38`,
  assets `openvm-v0.8.0` (v0.8.0 assets are still downloaded by #3520 — keep it that way).
- Provers: previous `docker_tag` + previous config template; no state migration needed
  (prover LevelDB is per-workspace).
- On-chain: MVRV `updateVerifier` back to the old wrapper for the affected range.

## 8. Open items (as of 2026-09-16)

- zkvm-prover `v0.9.1` tag at `bf887150` — lispc offered; needed to resolve the
  "pinned to raw commit" review minor.
- `devops` image workflow: `SCROLL_COMMIT` bump + halo2gpu build + solc 0.8.24
  (branch `zhuozhang/prover-halo2gpu-solc-0.8.24` covers the latter two; PR deferred).
- CI: rollup watcher `TestFunction` nil-`chainCfg` panic is a **pre-existing** flaky
  test (present on develop; the test passes `nil` as `chainCfg`) — not caused by #1816;
  re-run the job if it trips.
