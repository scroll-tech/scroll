#!/usr/bin/env python3
"""
Fetch L2 block headers from RPC and populate l2_block table in shadow DB.

The coordinator needs l2_block records to format chunk tasks (for block hashes
and hardfork name resolution). This script fetches blocks in batches and
inserts them into the shadow database.

Usage:
    python3 fetch-l2-blocks.py --rpc https://mainnet-rpc.scroll.io \
        --db "postgresql://<user>:<password>@localhost:5433/shadow_rollup" \
        --start-block 26000000 --end-block 27000000

After running, link blocks to chunks:
    UPDATE l2_block lb
    SET chunk_hash = c.hash
    FROM chunk c
    WHERE lb.number >= c.start_block_number
      AND lb.number <= c.end_block_number;
"""

import argparse
import sys
import time
import concurrent.futures
from typing import Optional

import requests
import psycopg2
from psycopg2.extras import execute_values


def fetch_block_batch(rpc_url: str, block_numbers: list[int]) -> list[dict]:
    """Fetch multiple blocks via batch JSON-RPC request."""
    payload = [
        {
            "jsonrpc": "2.0",
            "method": "eth_getBlockByNumber",
            "params": [hex(num), False],
            "id": i,
        }
        for i, num in enumerate(block_numbers)
    ]

    try:
        resp = requests.post(rpc_url, json=payload, headers={"Content-Type": "application/json"}, timeout=60)
        resp.raise_for_status()
        results = resp.json()

        blocks = []
        for result in results:
            if "error" in result:
                print(f"  Error fetching block: {result['error']}", file=sys.stderr)
                continue
            block = result.get("result")
            if block is None:
                continue
            blocks.append(block)
        return blocks
    except Exception as e:
        print(f"  Request failed: {e}", file=sys.stderr)
        return []


def insert_blocks(db_url: str, blocks: list[dict]) -> int:
    """Insert blocks into l2_block table."""
    if not blocks:
        return 0

    rows = []
    for block in blocks:
        try:
            number = int(block["number"], 16)
            hash_val = block["hash"]
            parent_hash = block["parentHash"]
            timestamp = int(block["timestamp"], 16)
            gas_used = int(block["gasUsed"], 16)
            rows.append((number, hash_val, parent_hash, timestamp, gas_used))
        except (KeyError, ValueError) as e:
            print(f"  Skipping malformed block: {e}", file=sys.stderr)
            continue

    if not rows:
        return 0

    conn = psycopg2.connect(db_url)
    try:
        with conn.cursor() as cur:
            execute_values(
                cur,
                """
                INSERT INTO l2_block (number, hash, parent_hash, timestamp, gas_used)
                VALUES %s
                ON CONFLICT (number) DO UPDATE SET
                    hash = EXCLUDED.hash,
                    parent_hash = EXCLUDED.parent_hash,
                    timestamp = EXCLUDED.timestamp,
                    gas_used = EXCLUDED.gas_used
                """,
                rows,
            )
        conn.commit()
        return len(rows)
    finally:
        conn.close()


def get_existing_block_range(db_url: str) -> tuple[Optional[int], Optional[int]]:
    """Get min/max block numbers already in the DB."""
    conn = psycopg2.connect(db_url)
    try:
        with conn.cursor() as cur:
            cur.execute("SELECT MIN(number), MAX(number) FROM l2_block")
            return cur.fetchone()
    finally:
        conn.close()


def main():
    parser = argparse.ArgumentParser(description="Fetch L2 blocks into shadow DB")
    parser.add_argument("--rpc", required=True, help="L2 RPC endpoint URL")
    parser.add_argument("--db", required=True, help="Shadow DB connection string")
    parser.add_argument("--start-block", type=int, required=True, help="First block to fetch")
    parser.add_argument("--end-block", type=int, required=True, help="Last block to fetch")
    parser.add_argument("--batch-size", type=int, default=100, help="RPC batch size (default: 100)")
    parser.add_argument("--workers", type=int, default=4, help="Concurrent workers (default: 4)")
    parser.add_argument("--delay", type=float, default=0.1, help="Delay between batches in seconds (default: 0.1)")
    parser.add_argument("--skip-existing", action="store_true", help="Skip blocks already in DB")
    args = parser.parse_args()

    existing_min, existing_max = get_existing_block_range(args.db)
    print(f"Existing blocks in DB: {existing_min or 'none'} to {existing_max or 'none'}")

    start = args.start_block
    end = args.end_block

    if args.skip_existing and existing_min is not None:
        # Only fetch gaps or new blocks
        # Simple approach: just fetch the requested range, ON CONFLICT will handle it
        pass

    total_blocks = end - start + 1
    print(f"Fetching {total_blocks} blocks from {start} to {end} via {args.rpc}")

    fetched = 0
    failed = 0

    # Generate batch ranges
    ranges = []
    current = start
    while current <= end:
        batch_end = min(current + args.batch_size - 1, end)
        ranges.append(list(range(current, batch_end + 1)))
        current = batch_end + 1

    with concurrent.futures.ThreadPoolExecutor(max_workers=args.workers) as executor:
        futures = {
            executor.submit(fetch_block_batch, args.rpc, block_nums): block_nums
            for block_nums in ranges
        }

        for future in concurrent.futures.as_completed(futures):
            block_nums = futures[future]
            try:
                blocks = future.result()
                if blocks:
                    inserted = insert_blocks(args.db, blocks)
                    fetched += inserted
                    print(f"  Blocks {block_nums[0]}-{block_nums[-1]}: inserted {inserted}/{len(blocks)}")
                else:
                    failed += len(block_nums)
                    print(f"  Blocks {block_nums[0]}-{block_nums[-1]}: FAILED")
            except Exception as e:
                failed += len(block_nums)
                print(f"  Blocks {block_nums[0]}-{block_nums[-1]}: ERROR {e}")

            time.sleep(args.delay)

    print(f"\nDone! Fetched: {fetched}, Failed: {failed}")
    print("\nNext step: link blocks to chunks:")
    print("""
    UPDATE l2_block lb
    SET chunk_hash = c.hash
    FROM chunk c
    WHERE lb.number >= c.start_block_number
      AND lb.number <= c.end_block_number;
    """)


if __name__ == "__main__":
    main()
