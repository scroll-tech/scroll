# Scroll L1 Contract Addresses

> Auto-generated from genesis configs and on-chain queries.
> Last updated: 2026-05-31

## Mainnet (Ethereum L1)

| Contract | Address | Verified Source |
|----------|---------|-----------------|
| **ScrollChain Proxy** | `0xa13BAF47339d63B743e7Da8741db5456DAc1E556` | [Etherscan](https://etherscan.io/address/0xa13BAF47339d63B743e7Da8741db5456DAc1E556) |
| ScrollChain Implementation | `0x0a20703878e68E587c59204cc0EA86098B8c3bA7` | (from proxy slot) |
| **MultipleVersionRollupVerifier** | `0x4CEA3E866e7c57fD75CB0CA3E9F5f1151D4Ead3F` | [Etherscan](https://etherscan.io/address/0x4CEA3E866e7c57fD75CB0CA3E9F5f1151D4Ead3F) |
| L1MessageQueueV1 | `0x0d7E906BD9cAFa154b048cFa766Cc1E54E39AF9B` | genesis.json |
| L1MessageQueueV2 | `0x56971da63A3C0205184FEF096E9ddFc7A8C2D18a` | genesis.json |
| L2SystemConfig | `0x331A873a2a85219863d80d248F9e2978fE88D0Ea` | genesis.json |
| Scroll Owner | `0x798576400F7D662961BA15C6b3F3d813447a26a6` | `owner()` on-chain |

### Mainnet Verifier History (from on-chain)

| Version | Start Batch | Verifier Address | Type |
|---------|-------------|------------------|------|
| 7 | 364,588 | `0xc084a6De8b0F2742396572d6f110eC87ca9329bA` | legacy |
| 8 | 0 | `0xa8d4702Aa5c09AF5dD1323E1842a43789021F485` | pre-v0.8.0 |
| 8 | 0 | `0xc3230A4C89a5Ce0455414215e533de4D8849b3f8` | Anvil-deployed (wrong digests) |
| **10** | 0 | `0x0dE180164Dc571522457101F5c47B2eaB36d0A82` | **GalileoV2 (mainnet)** |

### Mainnet Batch Status (as of block ~25,213,000)

- `lastCommittedBatchIndex`: ~517,843
- `lastFinalizedBatchIndex`: 517,843
- `committedBatches(517809)`: `0xeadeee9af865c6d13df6b66a45b3f3f161e6211aeb7d86e075a645f0e6a58f9e`
- `committedBatches(517843)`: `0x40545c71ed8fdcaabc06ad64599e9fdd4a62c1e2fd599a6642f64d229f7762a6`

---

## Sepolia (Ethereum Testnet)

| Contract | Address | Source |
|----------|---------|--------|
| **ScrollChain Proxy** | `0x2D567EcE699Eabe5afCd141eDB7A4f2D0D6ce8a0` | genesis.json |
| **MultipleVersionRollupVerifier** | `0x8A360c7F6fca548507017DdeD732bFe7E078F963` | `verifier()` on Sepolia |
| L1MessageQueueV1 | `0xF0B2293F5D834eAe920c6974D50957A1732de763` | genesis.json |
| L1MessageQueueV2 | `0xA0673eC0A48aa924f067F1274EcD281A10c5f19F` | genesis.json |
| L2SystemConfig | `0xF444cF06A3E3724e20B35c2989d3942ea8b59124` | genesis.json |
| Scroll Owner | `0xbE57544Eaf3515E888614a464EC9e0ad38f73e37` | `owner()` on Sepolia |

### Sepolia Batch Status

- `lastFinalizedBatchIndex`: 127,878 (0x1f386)

---

## Cloak (Validium Testnet)

| Contract | Address | Source |
|----------|---------|--------|
| **ScrollChain Proxy** | `0x9110B582327f6de87d8f833Ef7FAcD38CB093f64` | genesis.json |

---

## How These Addresses Were Found

### ScrollChain Proxy
The address `0xa13BAF47339d63B743e7Da8741db5456DAc1E556` appears in multiple places:
- `tests/prover-e2e/mainnet-galileoV2/genesis.json`: `"scrollChainAddress"`
- `coordinator/build/bin/conf/genesis.json`
- `bridge-history-api/conf/config.json`
- `scroll-devnets/charts/shadow-fork/e2e-test/values.yaml`

**Critical verification step**: Initially, Anvil was mistakenly forking Scroll L2 (chainId=534352) instead of Ethereum L1. On Scroll L2, `0xa13B...` had no ScrollChain code. After correcting Anvil to fork Ethereum mainnet (chainId=1), the address correctly resolved to the ScrollChain proxy with:
- Implementation slot: `0x0a20703878e68E587c59204cc0EA86098B8c3bA7`
- `lastFinalizedBatchIndex()`: 517,828
- `owner()`: `0x798576400F7D662961BA15C6b3F3d813447a26a6`

### MultipleVersionRollupVerifier
- **Mainnet**: Queried via `cast call 0xa13BAF... "verifier()(address)"` on Ethereum mainnet RPC → `0x4CEA3E866e7c57fD75CB0CA3E9F5f1151D4Ead3F`
- **Sepolia**: Queried via `cast call 0x2D567... "verifier()(address)"` on Sepolia RPC → `0x8A360c7F6fca548507017DdeD732bFe7E078F963`
- Also referenced in `scroll-devnets/charts/shadow-fork/jobs/upgrade-contract.yaml`

### Other Addresses
All other L1 contract addresses are extracted from the corresponding `genesis.json` files in `tests/prover-e2e/<network>/genesis.json`.
