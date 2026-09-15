# Transaction signers

Each rollup sender (`gas_oracle_sender`, `commit_sender`, `finalize_sender`) is
configured with its own `*_signer_config` block in the relayer config. The
`signer_type` field selects how transactions are signed:

| `signer_type` | Key location | Blob tx (EIP-4844) support |
| ------------- | ------------ | -------------------------- |
| `PrivateKey`  | In process / mounted secret | yes |
| `RemoteSigner`| web3signer (`eth_signTransaction`) | no |
| `AWSKMS`      | AWS KMS (never leaves KMS) | yes |

`commit_sender` submits batches as blob transactions, so it must use
`PrivateKey` or `AWSKMS`.

## AWSKMS

The `AWSKMS` signer keeps the private key inside AWS KMS. The service only sends
the 32-byte transaction signing hash to KMS and assembles the signature locally,
so all transaction types — including blob transactions — are supported.

```json
"commit_sender_signer_config": {
  "signer_type": "AWSKMS",
  "aws_kms_signer_config": {
    "key_id": "arn:aws:kms:us-east-1:123456789012:key/abcd-...",
    "region": "us-east-1",
    "signer_address": "0x1C5A77d9FA7eF466951B2F01F724BCa3A5820b63"
  }
}
```

- **`key_id`** — id, alias, or ARN of an asymmetric KMS key with key spec
  `ECC_SECG_P256K1` and key usage `SIGN_VERIFY`.
- **`region`** — optional; falls back to the ambient AWS config
  (`AWS_REGION`, shared config, instance/IRSA role) when empty. Startup fails if
  no region resolves from either.
- **`signer_address`** — **required**. The expected Ethereum address of the key.
  It is validated at startup against the address derived from the KMS public key,
  so a misconfigured `key_id` fails fast instead of signing from an unexpected
  account.

AWS credentials are resolved from the standard AWS SDK chain (IRSA, instance
role, or environment) — never put credentials in the config file.

### Recommended IAM setup

- Use a **separate KMS key per sender** so `kms:Sign` can be scoped and audited
  (via CloudTrail) per workload.
- Grant each workload only `kms:Sign` and `kms:GetPublicKey` on its key.
- KMS protects the key material, not the spending policy: keep the hot-wallet
  pattern (low balances, alerting, rotation). A compromised service can still
  request signatures, so consider transaction value/rate guards as a follow-up.

### Provisioning the key

There are two ways to get a key into KMS. Whichever you use, the signer never
needs the raw private key — it derives the Ethereum address from the KMS public
key (`keccak256(pubkey)[12:]`) at startup, and you put that address in
`signer_address`. To get it, take the `PublicKey` returned by
`aws kms get-public-key` (base64 SPKI DER), whose last 65 bytes are the
uncompressed point `0x04 || X || Y`, and keccak256-hash the 64 bytes after the
`0x04` prefix; the address is the last 20 bytes of that hash. Alternatively, start
the relayer with a placeholder `signer_address`: startup fails with an error that
names the address derived from the KMS key, which you then put in the config.

**Fund the address** before the sender goes live. Once configured correctly, the
relayer logs `initialized AWS KMS signer` with the key id and address, so you can
confirm which account a workload signs from.

**Option A — generate in KMS (recommended).** The private key is created inside
the KMS HSM and is non-exportable: it provably never exists outside KMS.

```bash
aws kms create-key \
  --key-spec ECC_SECG_P256K1 \
  --key-usage SIGN_VERIFY \
  --origin AWS_KMS \
  --description "scroll commit_sender"
# optional friendlier reference:
aws kms create-alias --alias-name alias/scroll-commit-sender --target-key-id <key-id>
```

Use this for new senders — fund the freshly derived address.

**Option B — import an existing private key (`Origin=EXTERNAL`).** Use this only
when you must preserve an already-funded address. KMS supports importing key
material for asymmetric keys, but the key existed in plaintext outside KMS at
least once, which is exactly the exposure the KMS signer otherwise eliminates —
so prefer Option A unless address continuity is a hard requirement. Create the
key with `--origin EXTERNAL`, then `get-parameters-for-import` /
`import-key-material` to upload the wrapped secp256k1 private key.

**Rotation.** KMS does not auto-rotate asymmetric keys, and a key's address
cannot change in place. Rotating means creating a new key (Option A), pointing
`signer_address`/`key_id` at it, and sweeping the old balance to the new address
— the old KMS key signs that sweep itself while still enabled, so no key access
is needed; disable/schedule-delete it only after the sweep confirms. Because a
non-exportable key can't leak, rotation is primarily an IAM-compromise response
rather than routine hygiene.
