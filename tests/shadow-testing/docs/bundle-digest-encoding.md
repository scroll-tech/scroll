# Bundle Verifier Digests: Montgomery (v0.8.0) vs Canonical (v0.9.0+)

The `ZkEvmVerifierPostFeynman` wrapper has two immutables — `verifierDigest1` /
`verifierDigest2` — that commit to the bundle circuit's verifying key. At
`finalizeBundlePostEuclidV2` time the wrapper inserts these two `bytes32`
words into the `instances` array it passes to the Plonk verifier. The Plonk
verifier only accepts the proof if those words equal the digest words embedded
in the proof itself (`instances` offsets **384–416 = digest1, 416–448 =
digest2**).

The catch: the digest files published on S3 are **not in the same field
encoding for every guest release**. Using them in the wrong encoding deploys a
wrapper whose immutables never match any real proof — every finalization
reverts with `VerificationFailed(0x439cc0cd)`.

## The Rule

| Guest release | S3 `bundle/digest_*.hex` encoding | Usable directly for deployment? |
|---------------|-----------------------------------|---------------------------------|
| **v0.8.0** (OpenVM 1.6.0) | **Montgomery form** (BN254) | ❌ — must convert to canonical first |
| **v0.9.0+** (OpenVM 2.0.0) | **Canonical form** | ✅ — use as-is |

The encoding problem was fixed in v0.9.0: from that release on, the published
digest files are already in the canonical form the Plonk verifier expects.

## Montgomery → Canonical Conversion

For v0.8.0 digests (and only those), convert before deploying:

```python
BN254_MOD = 21888242871839275222246405745257275088548364400416034343698204186575808495617
R = pow(2, 256, BN254_MOD)          # Montgomery radix
R_INV = pow(R, -1, BN254_MOD)

def montgomery_to_canonical(hex_str):
    return f"{(int(hex_str, 16) * R_INV) % BN254_MOD:064x}"
```

## Verified Example (Mainnet Production, v0.8.0)

Cross-checked three independent ways (S3 → conversion → on-chain wrapper → a
real production bundle proof), all consistent:

| Source | digest1 | digest2 |
|--------|---------|---------|
| S3 `scroll-zkvm/v0.8.0/bundle/digest_*.hex` (Montgomery) | `2bacecb92b61b7cf7cff726f230ee16a42b85eacac2fafa32d48d62f820023eb` | `2190d3ad4438e96758074b6e80d4999f75816dc2317a126b2acc7a02bc96f1e0` |
| → canonical (after conversion) | `0x00398b786b500ca759ca2de2aee9c73bd8e28f1c80b49e1c53bc060a9a649269` | `0x0021785a05e931b447c8d6463f4547f92081a92ee357af26e1c6f6ecfe373d67` |
| Mainnet production wrapper `0x808297224e86b1a6055B5F790a2cE07Ed611f955` immutables | `0x00398b78…` ✅ | `0x0021785a…` ✅ |
| Mainnet bundle #18246 proof `instances[384:448]` | `0x00398b78…` ✅ | `0x0021785a…` ✅ |

For comparison, v0.9.0 S3 digests (already canonical, no conversion):
`digest1 = 0x006770fb0f71f20718b614c8c5d3fb39482f0527806e67b0ea00ab5508245655`,
`digest2 = 0x00488b196ca5b2ea1fda4c960fe2e1b0b9715c0428a8ec6fb01a64684f8533f7`.

## How To Check Which Form You Have

1. **Against a deployed wrapper**:
   ```bash
   cast call "$WRAPPER" "verifierDigest1()(bytes32)" --rpc-url "$RPC"
   cast call "$WRAPPER" "verifierDigest2()(bytes32)" --rpc-url "$RPC"
   ```
   Compare with the S3 file. Equal → S3 file is canonical (v0.9.0 behavior).
   Equal only after Montgomery→canonical conversion → v0.8.0 behavior.

2. **Against a real proof** (ground truth): the digest words are always
   canonical inside the proof. From a coordinator/production DB:
   ```sql
   SELECT convert_from(proof, 'UTF8') FROM bundle WHERE index = <N>;
   ```
   decode `proof.instances` (base64) and read bytes `384:416` / `416:448`.

3. **Rule of thumb**: if `finalizeBundlePostEuclidV2` reverts with
   `VerificationFailed(0x439cc0cd)` but the Plonk verifier binary, public
   input hash, and proof are all individually correct — suspect digest
   encoding first.

## Notes

- `lib/03-deploy-verifier.sh` fetches `digest_*.hex` from S3 and deploys them
  **as-is**. That is correct for v0.9.0+ but would be wrong for a v0.8.0
  deployment (convert first, or extract from a proof's `instances`).
- The wrapper's `protocolVersion` (10 for GalileoV2 / codec V10) is orthogonal
  to the digest encoding issue.
- History: the discrepancy was root-caused during the early shadow-fork tests
  (wrapper `0xc323…` deployed with raw S3 digests failed with
  `VerificationFailed`); documented in commit `3bb5092c` which removed the old
  LESSONS_LEARNED chronicle — this file preserves the digest part.
