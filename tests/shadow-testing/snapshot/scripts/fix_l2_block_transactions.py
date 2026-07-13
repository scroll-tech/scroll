#!/usr/bin/env python3
"""
Fix missing l2_block.transactions data by fetching from L2 RPC.
"""

import json
import psycopg2
import requests
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import Optional

L2_RPC = "https://l2geth-rpc-proxy.mainnet.aws.scroll.io"
DB_DSN = "postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup"
BATCH_SIZE = 50
MAX_WORKERS = 20


def hex_to_int(hex_str: str) -> int:
    if hex_str is None:
        return 0
    return int(hex_str, 16)


def rpc_get_block(block_num: int) -> Optional[dict]:
    """Fetch full block with transactions from L2 RPC."""
    try:
        resp = requests.post(
            L2_RPC,
            json={
                "jsonrpc": "2.0",
                "method": "eth_getBlockByNumber",
                "params": [hex(block_num), True],
                "id": 1,
            },
            timeout=30,
        )
        resp.raise_for_status()
        result = resp.json().get("result")
        return result
    except Exception as e:
        print(f"Error fetching block {block_num}: {e}", file=sys.stderr)
        return None


def convert_tx_to_transaction_data(rpc_tx: dict) -> dict:
    """Convert RPC transaction JSON to TransactionData format."""
    tx_type = hex_to_int(rpc_tx.get("type", "0x0"))
    
    # Handle L1 message tx: nonce should be queueIndex, not tx.Nonce()
    if tx_type == 0x7E:
        nonce = hex_to_int(rpc_tx.get("queueIndex", "0x0"))
    else:
        nonce = hex_to_int(rpc_tx.get("nonce", "0x0"))
    
    # gasPrice: for legacy txs, use gasPrice; for EIP-1559, use effective gasPrice
    gas_price = rpc_tx.get("gasPrice")
    if gas_price is None:
        gas_price = rpc_tx.get("maxFeePerGas", "0x0")
    
    # gasTipCap / gasFeeCap
    gas_tip_cap = rpc_tx.get("maxPriorityFeePerGas", gas_price)
    gas_fee_cap = rpc_tx.get("maxFeePerGas", gas_price)
    
    # isCreate: true if 'to' is null
    to_addr = rpc_tx.get("to")
    is_create = to_addr is None
    
    # accessList: use null if empty or absent
    access_list = rpc_tx.get("accessList")
    if access_list == []:
        access_list = None
    
    # authorizationList
    auth_list = rpc_tx.get("authorizationList")
    if auth_list == []:
        auth_list = None
    
    # v: use yParity if available for EIP-1559, otherwise use v
    v = rpc_tx.get("v")
    if v is None:
        v = rpc_tx.get("yParity", "0x0")
    
    tx_data = {
        "type": tx_type,
        "nonce": nonce,
        "txHash": rpc_tx.get("hash", ""),
        "gas": hex_to_int(rpc_tx.get("gas", "0x0")),
        "gasPrice": gas_price,
        "gasTipCap": gas_tip_cap,
        "gasFeeCap": gas_fee_cap,
        "from": rpc_tx.get("from", ""),
        "to": to_addr,
        "chainId": rpc_tx.get("chainId", "0x82750"),
        "value": rpc_tx.get("value", "0x0"),
        "data": rpc_tx.get("input", "0x"),
        "isCreate": is_create,
        "accessList": access_list,
        "authorizationList": auth_list,
        "v": v,
        "r": rpc_tx.get("r", "0x0"),
        "s": rpc_tx.get("s", "0x0"),
    }
    
    return tx_data


def process_block(block_num: int) -> Optional[tuple]:
    """Fetch and convert transactions for a single block."""
    block = rpc_get_block(block_num)
    if block is None:
        return None
    
    rpc_txs = block.get("transactions", [])
    if not rpc_txs:
        tx_data = "[]"
    else:
        tx_list = [convert_tx_to_transaction_data(tx) for tx in rpc_txs]
        tx_data = json.dumps(tx_list, separators=(',', ':'))
    
    return (block_num, tx_data)


def update_blocks_in_db(blocks_data: list):
    """Update transactions for multiple blocks in shadow DB."""
    conn = psycopg2.connect(DB_DSN)
    cur = conn.cursor()
    
    try:
        for block_num, tx_data in blocks_data:
            cur.execute(
                "UPDATE l2_block SET transactions = %s WHERE number = %s",
                (tx_data, block_num)
            )
        conn.commit()
    except Exception as e:
        conn.rollback()
        print(f"DB update error: {e}", file=sys.stderr)
        raise
    finally:
        cur.close()
        conn.close()


def get_empty_blocks() -> list:
    """Get list of block numbers with empty transactions."""
    conn = psycopg2.connect(DB_DSN)
    cur = conn.cursor()
    cur.execute(
        "SELECT number FROM l2_block WHERE transactions = '' ORDER BY number"
    )
    blocks = [row[0] for row in cur.fetchall()]
    cur.close()
    conn.close()
    return blocks


def main():
    empty_blocks = get_empty_blocks()
    total = len(empty_blocks)
    print(f"Found {total} blocks with empty transactions")
    
    if total == 0:
        print("No empty transactions to fix.")
        return
    
    processed = 0
    failed_blocks = []
    
    for batch_start in range(0, total, BATCH_SIZE):
        batch = empty_blocks[batch_start:batch_start + BATCH_SIZE]
        print(f"Processing batch {batch_start//BATCH_SIZE + 1}/{(total-1)//BATCH_SIZE + 1}: blocks {batch[0]} to {batch[-1]}")
        
        blocks_data = []
        with ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
            futures = {executor.submit(process_block, bn): bn for bn in batch}
            for future in as_completed(futures):
                result = future.result()
                if result:
                    blocks_data.append(result)
                else:
                    failed_blocks.append(futures[future])
        
        if blocks_data:
            update_blocks_in_db(blocks_data)
            processed += len(blocks_data)
            print(f"  Updated {len(blocks_data)} blocks. Total processed: {processed}/{total}")
        
        # Small delay between batches to avoid rate limiting
        time.sleep(0.5)
    
    print(f"\nDone! Processed {processed}/{total} blocks.")
    if failed_blocks:
        print(f"Failed blocks: {failed_blocks}")


if __name__ == "__main__":
    main()
