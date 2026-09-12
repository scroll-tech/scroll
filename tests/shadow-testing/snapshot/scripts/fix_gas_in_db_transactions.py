#!/usr/bin/env python3
"""Normalize l2_block.transactions JSON for relayer/da-codec unmarshalling."""

import json
import psycopg2
import sys

DB_DSN = "postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup"
L1_MESSAGE_TX_TYPE = 0x7E


def hex_to_int(v):
    if isinstance(v, str) and v.startswith("0x"):
        return int(v, 16)
    return v


def int_to_hex(v):
    if isinstance(v, int):
        return f"0x{v:x}"
    return v


def fix_txs(txs):
    changed = False
    for tx in txs:
        # Numeric fields expected as integers by types.TransactionData.
        for field in ("gas", "nonce", "type"):
            v = tx.get(field)
            if isinstance(v, str):
                tx[field] = hex_to_int(v)
                changed = True

        # chainId must be a hex string for *hexutil.Big unmarshalling.
        chain_id = tx.get("chainId")
        if isinstance(chain_id, int):
            tx["chainId"] = int_to_hex(chain_id)
            changed = True

        # RPC uses 'input'; da-codec/types.TransactionData expects 'data'.
        if "data" not in tx and "input" in tx:
            tx["data"] = tx["input"]
            changed = True

        # RPC uses maxFeePerGas/maxPriorityFeePerGas; da-codec uses gasFeeCap/gasTipCap.
        if "gasFeeCap" not in tx and "maxFeePerGas" in tx:
            tx["gasFeeCap"] = tx["maxFeePerGas"]
            changed = True
        if "gasTipCap" not in tx and "maxPriorityFeePerGas" in tx:
            tx["gasTipCap"] = tx["maxPriorityFeePerGas"]
            changed = True

        # Ensure hex string fields are actually strings (some may be integers in DB).
        for field in ("gasPrice", "gasFeeCap", "gasTipCap", "value", "v", "r", "s",
                      "maxFeePerGas", "maxPriorityFeePerGas"):
            if field in tx and isinstance(tx[field], int):
                tx[field] = int_to_hex(tx[field])
                changed = True

        # L1 message txs: RPC stores the queue index in 'queueIndex' and leaves
        # 'nonce' as 0. da-codec reuses 'nonce' for the queue index.
        if tx.get("type") == L1_MESSAGE_TX_TYPE and "queueIndex" in tx:
            queue_index = tx["queueIndex"]
            if isinstance(queue_index, str):
                queue_index = hex_to_int(queue_index)
            if tx.get("nonce") != queue_index:
                tx["nonce"] = queue_index
                changed = True

    return changed


def main():
    start_block = int(sys.argv[1]) if len(sys.argv) > 1 else 33848707
    end_block = int(sys.argv[2]) if len(sys.argv) > 2 else 33851426

    conn = psycopg2.connect(DB_DSN)
    cur = conn.cursor()
    try:
        cur.execute(
            "SELECT number, transactions FROM l2_block WHERE number BETWEEN %s AND %s",
            (start_block, end_block),
        )
        rows = cur.fetchall()
        print(f"Found {len(rows)} blocks to inspect ({start_block}-{end_block})")

        updated = 0
        for number, tx_json in rows:
            if not tx_json:
                continue
            try:
                txs = json.loads(tx_json)
            except Exception as e:
                print(f"Skip block {number}: parse error {e}", file=sys.stderr)
                continue
            if not isinstance(txs, list):
                continue
            if fix_txs(txs):
                cur.execute(
                    "UPDATE l2_block SET transactions = %s WHERE number = %s",
                    (json.dumps(txs, separators=(",", ":")), number),
                )
                updated += 1
                if updated % 100 == 0:
                    conn.commit()
                    print(f"  updated {updated} blocks so far...")

        conn.commit()
        print(f"Done: updated {updated} blocks.")
    finally:
        cur.close()
        conn.close()


if __name__ == "__main__":
    main()
