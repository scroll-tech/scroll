#!/usr/bin/env python3
"""Sync L1MessageQueueV2 rolling hashes from ETH mainnet into the Anvil fork.

Background (TROUBLESHOOTING Trap 23): bundles that pop L1 messages enqueued
*after* the Anvil fork block fail finalization with VerificationFailed
(0x439cc0cd), because ScrollChain reads getMessageRollingHash() from the
forked L1MessageQueueV2 and the rolling-hash mapping (storage slot 101) has
no entries for post-fork messages. The contract then builds a publicInput
whose messageQueueHash differs from the prover's, so the plonk verifier
rejects the proof.

This script copies getMessageRollingHash(i) from a mainnet RPC into the fork
via anvil_setStorageAt for every index the fork is missing, starting at the
fork's nextCrossDomainMessageIndex - 1 (the last entry that must exist).
Idempotent; safe to run on a cron.

Note the off-by-one: the contract's rolling hash at index i is the hash
*after* message i, so a bundle popping up to index N needs rh[N-1].

Usage: python3 sync-queue-hashes.py [--dry-run]
Env overrides: FORK_RPC, MAINNET_RPC, QUEUE_ADDR
"""

import json
import os
import subprocess
import sys
import urllib.request

from Crypto.Hash import keccak

FORK_RPC = os.environ.get("FORK_RPC", "http://localhost:18545")
MAINNET_RPC = os.environ.get(
    "MAINNET_RPC", "https://ethereum-rpc.publicnode.com"
)
QUEUE = os.environ.get("QUEUE_ADDR", "0x56971da63A3C0205184FEF096E9ddFc7A8C2D18a")
MAPPING_SLOT = 101  # messageRollingHashes (verified against live storage)
MAX_PER_RUN = 512  # safety bound per invocation


def rpc(url, method, params):
    req = urllib.request.Request(
        url,
        data=json.dumps(
            {"jsonrpc": "2.0", "id": 1, "method": method, "params": params}
        ).encode(),
        headers={"Content-Type": "application/json", "User-Agent": "shadow-sync/1.0"},
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        out = json.loads(resp.read())
    if "error" in out and out["error"]:
        raise RuntimeError(f"{method} failed: {out['error']}")
    return out["result"]


def rolling_hash(url, index):
    # getMessageRollingHash(uint256) selector = 0xc6172e1f
    data = "0xc6172e1f" + index.to_bytes(32, "big").hex()
    res = rpc(url, "eth_call", [{"to": QUEUE, "data": data}, "latest"])
    return bytes.fromhex(res[2:])


def next_index(url):
    # nextCrossDomainMessageIndex() selector = 0xfd0ad31e
    res = rpc(url, "eth_call", [{"to": QUEUE, "data": "0xfd0ad31e"}, "latest"])
    return int(res, 16)


def mapping_entry_slot(index):
    k = keccak.new(digest_bits=256)
    k.update(index.to_bytes(32, "big") + MAPPING_SLOT.to_bytes(32, "big"))
    return "0x" + k.hexdigest()


def set_storage(index, value32):
    subprocess.run(
        [
            "cast",
            "rpc",
            "anvil_setStorageAt",
            QUEUE,
            mapping_entry_slot(index),
            "0x" + value32.hex(),
            "--rpc-url",
            FORK_RPC,
        ],
        check=True,
        capture_output=True,
    )


def main():
    dry_run = "--dry-run" in sys.argv
    start = next_index(FORK_RPC) - 1  # last index guaranteed present on fork
    print(f"fork nextCrossDomainMessageIndex-1 = {start}")
    synced = 0
    i = start + 1
    while synced < MAX_PER_RUN:
        try:
            mainnet_hash = rolling_hash(MAINNET_RPC, i)
        except Exception as e:
            print(f"mainnet RPC error at index {i}: {e}; stopping")
            break
        if mainnet_hash == b"\x00" * 32:
            break  # not enqueued on mainnet yet
        fork_hash = rolling_hash(FORK_RPC, i)
        if fork_hash != mainnet_hash:
            print(f"index {i}: fork={fork_hash.hex() or '0'} -> mainnet={mainnet_hash.hex()}")
            if not dry_run:
                set_storage(i, mainnet_hash)
            synced += 1
        i += 1
    print(f"done; {'would sync' if dry_run else 'synced'} {synced} rolling hashes")

    # Keep the fork's enqueue cursor in line with mainnet: finalizing a bundle
    # that pops messages beyond the fork's nextCrossDomainMessageIndex reverts
    # with ErrorFinalizedIndexTooLarge (0x16465978). Slot 103 = 0x67.
    mainnet_next = next_index(MAINNET_RPC)
    fork_next = next_index(FORK_RPC)
    if mainnet_next > fork_next:
        print(f"nextCrossDomainMessageIndex: fork={fork_next} -> mainnet={mainnet_next}")
        if not dry_run:
            subprocess.run(
                [
                    "cast", "rpc", "anvil_setStorageAt", QUEUE, hex(103),
                    hex(mainnet_next), "--rpc-url", FORK_RPC,
                ],
                check=True,
                capture_output=True,
            )


if __name__ == "__main__":
    main()
