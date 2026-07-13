#!/usr/bin/env python3
"""
Sync batch.withdraw_root from the batch proof metadata.

In shadow-test setups, fetch-l2-blocks.py cannot read withdraw_root from a
standard L2 RPC and stores 0x0...0 in l2_block.withdraw_root. That placeholder
propagates to chunk.withdraw_root and batch.withdraw_root, but the prover
correctly computes the real withdraw root while generating batch proofs and
stores it in batch.proof -> metadata.batch_info.withdraw_root.

This script reads each batch's proof JSON and overwrites the top-level
batch.withdraw_root column with the value from the proof metadata. Run it after
the coordinator has verified batch proofs and before starting the relayer.

Usage:
    DB_DSN="postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup" \
        python3 09-sync-batch-withdraw-roots.py --batch-range 517766:517795
"""

import argparse
import json
import os
import sys

import psycopg2
from psycopg2.extras import RealDictCursor


def main():
    parser = argparse.ArgumentParser(description="Sync batch.withdraw_root from proof metadata")
    parser.add_argument("--batch-range", required=True, help="Batch index range, e.g. 517766:517795")
    parser.add_argument("--db-dsn", default=os.environ.get("DB_DSN"), help="PostgreSQL DSN")
    parser.add_argument("--dry-run", action="store_true", help="Print changes without applying")
    args = parser.parse_args()

    if not args.db_dsn:
        print("ERROR: --db-dsn or DB_DSN environment variable required", file=sys.stderr)
        sys.exit(1)

    start, end = map(int, args.batch_range.split(":"))

    conn = psycopg2.connect(args.db_dsn)
    cur = conn.cursor(cursor_factory=RealDictCursor)

    cur.execute(
        "SELECT index, withdraw_root, encode(proof, 'escape') AS proof_text FROM batch "
        "WHERE index BETWEEN %s AND %s ORDER BY index",
        (start, end),
    )
    rows = cur.fetchall()

    updates = []
    for row in rows:
        batch_index = row["index"]
        old_wr = row["withdraw_root"]
        proof_text = row["proof_text"]

        if not proof_text:
            print(f"Batch {batch_index}: no proof, skipped")
            continue

        try:
            proof = json.loads(proof_text)
            new_wr = proof["metadata"]["batch_info"]["withdraw_root"]
        except (json.JSONDecodeError, KeyError) as exc:
            print(f"Batch {batch_index}: failed to parse proof metadata: {exc}")
            continue

        if old_wr.lower() == new_wr.lower():
            continue

        updates.append((new_wr, batch_index, old_wr))

    if not updates:
        print("No updates needed")
        return

    print(f"Will update {len(updates)} batch(es)")
    for new_wr, batch_index, old_wr in updates:
        print(f"  batch {batch_index}: {old_wr} -> {new_wr}")

    if args.dry_run:
        return

    cur.executemany("UPDATE batch SET withdraw_root = %s WHERE index = %s", [(u[0], u[1]) for u in updates])
    conn.commit()
    print(f"Updated {len(updates)} batch(es)")

    cur.close()
    conn.close()


if __name__ == "__main__":
    main()
