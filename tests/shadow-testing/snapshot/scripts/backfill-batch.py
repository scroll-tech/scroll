#!/usr/bin/env python3
"""Backfill a specific batch (and its chunks/l2_blocks) from mainnet to shadow DB."""

import os
import sys

import psycopg2

SRC_DSN = os.environ.get(
    "MAINNET_DSN",
    "postgresql://mainnet_infra_team_read_only:AuexDUuaarskbG6tr9CH9gXsJqp4at67mddAbMrt@localhost:15432/mainnet_rollup",
)
DST_DSN = os.environ.get(
    "SHADOW_DSN",
    "postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup",
)


def get_columns(cur, table):
    cur.execute(
        "SELECT column_name FROM information_schema.columns "
        "WHERE table_schema = 'public' AND table_name = %s ORDER BY ordinal_position",
        (table,),
    )
    return [r[0] for r in cur.fetchall()]


def copy_row_range(src_dsn, dst_dsn, table, pk_col, pk_min, pk_max, conflict_clause):
    src = psycopg2.connect(src_dsn)
    dst = psycopg2.connect(dst_dsn)
    try:
        src_cur = src.cursor()
        dst_cur = dst.cursor()
        cols = get_columns(dst_cur, table)
        col_sql = ", ".join(cols)
        placeholders = ", ".join(["%s"] * len(cols))
        insert_sql = f"INSERT INTO {table} ({col_sql}) VALUES ({placeholders}) {conflict_clause}"

        src_cur.execute(
            f"SELECT {col_sql} FROM {table} WHERE {pk_col} >= %s AND {pk_col} <= %s ORDER BY {pk_col}",
            (pk_min, pk_max),
        )
        rows = src_cur.fetchall()
        if not rows:
            print(f"No rows for {table} in range {pk_min}..{pk_max}")
            return 0
        dst_cur.executemany(insert_sql, rows)
        inserted = dst_cur.rowcount
        dst.commit()
        print(f"{table}: inserted {inserted} rows in range {pk_min}..{pk_max}")
        return inserted
    finally:
        src.close()
        dst.close()


def main():
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <batch_index>")
        sys.exit(1)
    batch_index = int(sys.argv[1])

    src = psycopg2.connect(SRC_DSN)
    src_cur = src.cursor()
    src_cur.execute(
        "SELECT start_chunk_index, end_chunk_index FROM batch WHERE index = %s",
        (batch_index,),
    )
    row = src_cur.fetchone()
    src.close()
    if not row:
        print(f"Batch {batch_index} not found in mainnet")
        sys.exit(1)
    start_chunk, end_chunk = row

    # Copy batch itself.
    copy_row_range(
        SRC_DSN,
        DST_DSN,
        "batch",
        "index",
        batch_index,
        batch_index,
        "ON CONFLICT (index) WHERE deleted_at IS NULL DO NOTHING",
    )

    # Copy chunks for this batch.
    copy_row_range(
        SRC_DSN,
        DST_DSN,
        "chunk",
        "index",
        start_chunk,
        end_chunk,
        "ON CONFLICT (index) WHERE deleted_at IS NULL DO NOTHING",
    )

    # Copy l2_block rows referenced by those chunks.
    src = psycopg2.connect(SRC_DSN)
    src_cur = src.cursor()
    src_cur.execute(
        "SELECT MIN(start_block_number), MAX(end_block_number) FROM chunk WHERE index >= %s AND index <= %s",
        (start_chunk, end_chunk),
    )
    min_block, max_block = src_cur.fetchone()
    src.close()
    if min_block is not None:
        copy_row_range(
            SRC_DSN,
            DST_DSN,
            "l2_block",
            "number",
            min_block,
            max_block,
            "ON CONFLICT (number) WHERE deleted_at IS NULL DO NOTHING",
        )
        copy_row_range(
            SRC_DSN,
            DST_DSN,
            "l1_message",
            "height",
            min_block,
            max_block,
            "ON CONFLICT (queue_index) WHERE deleted_at IS NULL DO NOTHING",
        )


if __name__ == "__main__":
    main()
