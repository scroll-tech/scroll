#!/usr/bin/env python3
"""
Sync Scroll mainnet rollup metadata into the local shadow DB.

This copies chunk/batch/bundle rows (plus the l2_block/l1_message data they
reference) from the production/mainnet read replica to the shadow database.
When paired with a relayer that has its local chunk/batch/bundle proposers
disabled, this lets the shadow fork follow the real mainnet cadence instead
of synthesizing bundles at an unrealistic rate.

Baseline sync uses psql COPY ... TO STDOUT | COPY ... FROM STDIN pipelines.
Incremental polling uses simple INSERT ... ON CONFLICT because the delta is small.

For chunk/batch/bundle we deliberately do NOT copy proof columns; the shadow
fork is meant to exercise the local v0.9.0 prover, not reuse mainnet proofs.

Canary parallel-upgrade mode (opt-in via --import-proofs or SYNC_PROOFS=1):
mainnet bundle proofs ARE imported, into a shadow-private quarantine table
`remote_bundle_proof`, and applied to local bundle rows below a boundary batch
(Phase A of the canary upgrade test finalizes the backlog with production
proofs instead of local proving). Default mode (flag off) is unchanged.
"""

import argparse
import importlib.util
import json
import logging
import os
import subprocess
import time
import urllib.request
from pathlib import Path

import psycopg2

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s",
)
log = logging.getLogger("sync-mainnet-db")

# sync-queue-hashes.py (Trap 23) is loaded as a module so each poll cycle can
# also mirror L1MessageQueueV2 rolling hashes onto the Anvil fork. It lives in
# the shared lib/ directory (follow/scripts -> follow -> shadow-testing).
_sqh_spec = importlib.util.spec_from_file_location(
    "sync_queue_hashes",
    Path(__file__).resolve().parent.parent.parent / "lib" / "sync-queue-hashes.py",
)
_sqh = importlib.util.module_from_spec(_sqh_spec)
_sqh_spec.loader.exec_module(_sqh)

SRC_DSN = os.environ.get(
    "MAINNET_DSN",
    "postgresql://mainnet_infra_team_read_only:AuexDUuaarskbG6tr9CH9gXsJqp4at67mddAbMrt@localhost:15432/mainnet_rollup",
)
DST_DSN = os.environ.get(
    "SHADOW_DSN",
    "postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup",
)
ANVIL_RPC = os.environ.get("ANVIL_RPC", "http://localhost:18545")
SCROLL_CHAIN = os.environ.get(
    "SCROLL_CHAIN", "0xa13BAF47339d63B743e7Da8741db5456DAc1E556"
)

# ── Canary proof import (opt-in: --import-proofs / SYNC_PROOFS=1) ────────────
# .work is shared by both modes and stays at tests/shadow-testing/.work
# (follow/scripts -> follow -> shadow-testing). Overridable for testability.
WORK_DIR = Path(
    os.environ.get(
        "WORK_DIR", Path(__file__).resolve().parent.parent.parent / ".work"
    )
)
# Boundary file written by 30-canary-upgrade.sh: PROOF_IMPORT_MAX_END_BATCH=N.
# Missing file/key = infinity (Phase A applies every quarantined proof).
CANARY_ENV_FILE = Path(os.environ.get("CANARY_ENV_FILE", WORK_DIR / "canary.env"))
# Persistent cursor (last remote bundle index scanned) so proof import is
# restart-safe; upserts are idempotent, so duplicates are harmless.
PROOF_CURSOR_FILE = Path(
    os.environ.get("PROOF_CURSOR_FILE", WORK_DIR / "proof-import.cursor")
)

PK_COLUMNS = {
    "chunk": "index",
    "batch": "index",
    "bundle": "index",
}

# l2_block and l1_message are excluded from incremental poll sync because
# querying MAX() on the 181 GB mainnet l2_block table frequently times out.
# They are copied during the initial baseline sync instead.

# Poll mode never bulk-backfills: deltas larger than this (or an empty shadow
# table) require a baseline sync instead.
MAX_POLL_DELTA = int(os.environ.get("MAX_POLL_DELTA", "5000"))

CONFLICT_CLAUSES = {
    "l2_block": "ON CONFLICT (number) WHERE deleted_at IS NULL DO NOTHING",
    "l1_message": "ON CONFLICT (queue_index) WHERE deleted_at IS NULL DO NOTHING",
    "chunk": "ON CONFLICT (index) WHERE deleted_at IS NULL DO NOTHING",
    "batch": "ON CONFLICT (index) WHERE deleted_at IS NULL DO NOTHING",
    "bundle": "ON CONFLICT (index) DO NOTHING",
}

# Columns that should not be copied from mainnet for chunk/batch/bundle,
# so that the shadow fork reproves them locally with v0.9.0.
PROOF_RELATED_COLS = {
    "chunk": {
        "proof",
        "proving_status",
        "prover_assigned_at",
        "proved_at",
        "proof_time_sec",
        "total_attempts",
        "active_attempts",
    },
    "batch": {
        "proof",
        "proving_status",
        "chunk_proofs_status",
        "prover_assigned_at",
        "proved_at",
        "proof_time_sec",
        "total_attempts",
        "active_attempts",
    },
    "bundle": {
        "proof",
        "proving_status",
        "batch_proofs_status",
        "proved_at",
        "proof_time_sec",
        "total_attempts",
        "active_attempts",
    },
}


def dsn_to_env(dsn):
    """Return libpq env vars for a DSN so psql does not log credentials."""
    parts = psycopg2.extensions.parse_dsn(dsn)
    env = {}
    if "host" in parts:
        env["PGHOST"] = parts["host"]
    if "port" in parts:
        env["PGPORT"] = str(parts["port"])
    if "user" in parts:
        env["PGUSER"] = parts["user"]
    if "password" in parts:
        env["PGPASSWORD"] = parts["password"]
    if "dbname" in parts:
        env["PGDATABASE"] = parts["dbname"]
    return env


def connect(dsn):
    return psycopg2.connect(dsn)


def connect_env_cursor(pg_env):
    """Open a psycopg2 cursor from libpq-style env vars (PGHOST etc.)."""
    conn = psycopg2.connect(
        host=pg_env.get("PGHOST"),
        port=pg_env.get("PGPORT"),
        user=pg_env.get("PGUSER"),
        password=pg_env.get("PGPASSWORD"),
        dbname=pg_env.get("PGDATABASE"),
    )
    return conn.cursor()


def get_watermarks(cur):
    res = {}
    for table, pk in PK_COLUMNS.items():
        cur.execute(f"SELECT MAX({pk}) FROM {table}")
        row = cur.fetchone()
        res[table] = row[0] if row and row[0] is not None else 0
    return res


def get_fork_committed_index(rpc_url=ANVIL_RPC, contract=SCROLL_CHAIN, fallback=None):
    """Query the Anvil fork's miscData.lastCommittedBatchIndex via JSON-RPC.

    miscData()(uint64 lastCommittedBatchIndex, uint64 lastFinalizedBatchIndex,
                uint32 lastFinalizeTimestamp, uint8 flags, uint88 reserved)
    selector = keccak256("miscData()")[:4] = 0x06582acb (verified with `cast sig`).
    Returns fallback (or None) when the RPC is unreachable.
    """
    payload = json.dumps(
        {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "eth_call",
            "params": [{"to": contract, "data": "0x06582acb"}, "latest"],
        }
    ).encode()
    try:
        req = urllib.request.Request(
            rpc_url, data=payload, headers={"Content-Type": "application/json"}
        )
        with urllib.request.urlopen(req, timeout=10) as resp:
            result = json.loads(resp.read())["result"]
        # first return word = lastCommittedBatchIndex
        return int(result[2:66], 16)
    except Exception as e:
        log.warning("Failed to query fork lastCommittedBatchIndex: %s", e)
        return fallback


def fixup_rollup_status(dst_cur, table, lo, hi, fork_committed):
    """Reset mainnet rollup lifecycle state on freshly synced rows.

    New rows arrive with mainnet's rollup_status/commit/finalize columns, but the
    shadow fork runs its own rollup:
      * batches at or below the fork's committed boundary are already committed
        on the fork -> rollup_status = 3 (committed, pending finalization)
      * batches above the boundary must be committed by the shadow relayer
        -> rollup_status = 1 (pending commit)
      * bundles are always pending finalization -> rollup_status = 1
    Scoped to (lo, hi] so rows already advanced by the shadow relayer are
    never touched.
    """
    if fork_committed is None:
        log.warning(
            "Skipping rollup_status fixup for %s (%s..%s): fork committed index unknown",
            table,
            lo,
            hi,
        )
        return
    if table == "batch":
        dst_cur.execute(
            """
            UPDATE batch
            SET rollup_status = 3,
                commit_tx_hash = NULL, committed_at = NULL,
                finalize_tx_hash = NULL, finalized_at = NULL,
                oracle_status = 1, oracle_tx_hash = NULL
            WHERE index > %s AND index <= %s AND index <= %s
              AND rollup_status <> 3
            """,
            (lo, hi, fork_committed),
        )
        n_committed = dst_cur.rowcount
        dst_cur.execute(
            """
            UPDATE batch
            SET rollup_status = 1,
                commit_tx_hash = NULL, committed_at = NULL,
                finalize_tx_hash = NULL, finalized_at = NULL,
                oracle_status = 1, oracle_tx_hash = NULL
            WHERE index > %s AND index <= %s AND index > %s
              AND rollup_status <> 1
            """,
            (lo, hi, fork_committed),
        )
        n_pending = dst_cur.rowcount
        if n_committed or n_pending:
            log.info(
                "batch rollup_status fixup (%s..%s]: %d committed, %d pending (fork boundary %d)",
                lo,
                hi,
                n_committed,
                n_pending,
                fork_committed,
            )
    elif table == "bundle":
        dst_cur.execute(
            """
            UPDATE bundle
            SET rollup_status = 1, finalize_tx_hash = NULL, finalized_at = NULL
            WHERE index > %s AND index <= %s AND rollup_status <> 1
            """,
            (lo, hi),
        )
        if dst_cur.rowcount:
            log.info(
                "bundle rollup_status fixup (%s..%s]: %d reset to pending",
                lo,
                hi,
                dst_cur.rowcount,
            )


def get_columns(src_cur, table):
    src_cur.execute(
        "SELECT column_name FROM information_schema.columns "
        "WHERE table_schema = 'public' AND table_name = %s ORDER BY ordinal_position",
        (table,),
    )
    return [r[0] for r in src_cur.fetchall()]


def copy_table_pipe(src_env, dst_env, table, where_clause, columns=None):
    """Copy a table slice from mainnet into the shadow DB via a psql pipe.

    Idempotent: rows are piped into a staging table first, then merged with
    INSERT ... ON CONFLICT DO NOTHING so re-running a baseline over a shadow
    DB that already has data (and local proving/rollup state) never fails on
    duplicate keys and never clobbers existing rows.
    """
    log.info("copy %s WHERE %s", table, where_clause)
    src_env_vars = {k: v for k, v in src_env.items() if k.startswith("PG")}
    dst_env_vars = {k: v for k, v in dst_env.items() if k.startswith("PG")}
    staging = f"staging_{table}"

    if columns:
        col_sql = ", ".join(columns)
        copy_out_cmd = f"COPY (SELECT {col_sql} FROM public.{table} WHERE {where_clause}) TO STDOUT WITH (FORMAT text)"
        copy_in_cmd = f"COPY public.{staging} ({col_sql}) FROM STDIN WITH (FORMAT text)"
    else:
        col_sql = None
        copy_out_cmd = f"COPY (SELECT * FROM public.{table} WHERE {where_clause}) TO STDOUT WITH (FORMAT text)"
        copy_in_cmd = f"COPY public.{staging} FROM STDIN WITH (FORMAT text)"

    env_in = os.environ.copy()
    env_in.update(dst_env_vars)

    # Recreate the staging table (no constraints/indexes: source duplicates
    # must not abort the COPY; dedupe happens at merge time). INCLUDING
    # DEFAULTS so columns excluded from the copy (proof-related) get their
    # defaults instead of violating NOT NULL.
    prep = subprocess.run(
        ["psql", "--quiet", "--set", "ON_ERROR_STOP=1", "--command",
         f"DROP TABLE IF EXISTS public.{staging}; "
         f"CREATE TABLE public.{staging} (LIKE public.{table} INCLUDING DEFAULTS);"],
        env=env_in,
        capture_output=True,
    )
    if prep.returncode != 0:
        raise RuntimeError(
            f"psql staging prep failed for {table}: {prep.stderr.decode(errors='replace')}"
        )

    copy_out = [
        "psql",
        "--quiet",
        "--set", "ON_ERROR_STOP=1",
        "--command",
        copy_out_cmd,
    ]
    copy_in = [
        "psql",
        "--quiet",
        "--set", "ON_ERROR_STOP=1",
        "--command",
        copy_in_cmd,
    ]

    env_out = os.environ.copy()
    env_out.update(src_env_vars)

    with subprocess.Popen(
        copy_out,
        env=env_out,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    ) as out_proc:
        in_proc = subprocess.Popen(
            copy_in,
            env=env_in,
            stdin=out_proc.stdout,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        out_proc.stdout.close()
        in_stdout, in_stderr = in_proc.communicate()
        out_stdout, out_stderr = out_proc.communicate()

    if in_proc.returncode != 0:
        raise RuntimeError(
            f"psql COPY IN failed for {staging}: rc={in_proc.returncode} stderr={in_stderr.decode(errors='replace')}"
        )
    if out_proc.returncode != 0:
        raise RuntimeError(
            f"psql COPY OUT failed for {table}: rc={out_proc.returncode} stderr={out_stderr.decode(errors='replace')}"
        )

    # Merge into the real table, skipping rows that already exist.
    select_cols = col_sql if col_sql else ", ".join(get_columns(connect_env_cursor(dst_env), table))
    merge = subprocess.run(
        ["psql", "--quiet", "--set", "ON_ERROR_STOP=1", "--command",
         f"INSERT INTO public.{table} ({select_cols}) SELECT {select_cols} FROM public.{staging} "
         f"{CONFLICT_CLAUSES[table]}; "
         f"DROP TABLE public.{staging};"],
        env=env_in,
        capture_output=True,
    )
    if merge.returncode != 0:
        raise RuntimeError(
            f"psql merge failed for {table}: {merge.stderr.decode(errors='replace')}"
        )
    log.info("copy %s complete", table)


def copy_range(src, dst_cur, table, pk, min_val, max_val, page_size=500, columns=None):
    """Incremental copy for polling: small deltas, executemany is fine."""
    if columns is None:
        cols = get_columns(src.cursor(), table)
    else:
        cols = columns
    col_sql = ", ".join(cols)
    placeholders = ", ".join(["%s"] * len(cols))
    insert_sql = f"INSERT INTO {table} ({col_sql}) VALUES ({placeholders}) {CONFLICT_CLAUSES[table]}"
    where = f"WHERE {pk} > %s AND {pk} <= %s"

    with src.cursor() as count_cur:
        count_cur.execute(f"SELECT COUNT(*) FROM {table} {where}", (min_val, max_val))
        total = count_cur.fetchone()[0]
    if total == 0:
        log.info("No rows to copy for %s in range %s..%s", table, min_val, max_val)
        return 0
    log.info("Copying up to %d rows for %s (range %s..%s)", total, table, min_val, max_val)

    inserted = 0
    with src.cursor(name=f"sync_{table}_cur") as cur:
        cur.itersize = page_size
        cur.execute(f"SELECT {col_sql} FROM {table} {where} ORDER BY {pk}", (min_val, max_val))
        while True:
            rows = cur.fetchmany(page_size)
            if not rows:
                break
            dst_cur.executemany(insert_sql, rows)
            inserted += dst_cur.rowcount
            if inserted % (page_size * 10) == 0:
                log.info("  %s: inserted %d/%d ...", table, inserted, total)
    log.info("Finished %s: inserted %d new rows", table, inserted)
    return inserted


def update_bundle_seq(cur):
    cur.execute(
        "SELECT setval('bundle_index_seq', GREATEST(coalesce(max(index),0),1)+1) FROM bundle"
    )


def sync_parent_links(dst_cur):
    """Backfill parent links on synced rows.

    Mainnet sets chunk.batch_hash when its batch proposer assigns chunks,
    and batch.bundle_hash on bundle proposal. Chunks/batches copied by poll
    sync before that assignment keep the NULL link forever (ON CONFLICT DO
    NOTHING never updates them), which silently blocks the coordinator:
    batches need chunk.batch_hash for the chunk_proofs_status promotion,
    bundles need batch.bundle_hash for batch_proofs_status. Re-derive the
    links every cycle from the parent rows' index ranges instead.
    """
    dst_cur.execute(
        "UPDATE chunk c SET batch_hash = b.hash FROM batch b "
        "WHERE c.index BETWEEN b.start_chunk_index AND b.end_chunk_index "
        "AND b.index > (SELECT COALESCE(MAX(index), 0) - 500 FROM batch) "
        "AND (c.batch_hash IS NULL OR c.batch_hash = '' OR c.batch_hash <> b.hash)"
    )
    if dst_cur.rowcount:
        log.info("chunk: linked batch_hash on %d rows", dst_cur.rowcount)
    dst_cur.execute(
        "UPDATE batch b SET bundle_hash = u.hash FROM bundle u "
        "WHERE b.index BETWEEN u.start_batch_index AND u.end_batch_index "
        "AND u.index > (SELECT COALESCE(MAX(index), 0) - 500 FROM bundle) "
        "AND (b.bundle_hash IS NULL OR b.bundle_hash = '' OR b.bundle_hash <> u.hash)"
    )
    if dst_cur.rowcount:
        log.info("batch: linked bundle_hash on %d rows", dst_cur.rowcount)


def sync_l2_blocks(src, dst_cur):
    """Copy l2_block rows for recent shadow chunks that have none.

    The coordinator needs l2_block (linked via chunk_hash) to format chunk
    prover tasks; without it every task fails with "failed to fetch block
    hashes of a chunk". Poll mode cannot watermark l2_block by MAX(number)
    (the 181 GB mainnet table times out), so instead we look for recent
    shadow chunks with no blocks and copy their block range from mainnet.

    Note the blocks usually already exist in the shadow DB (imported by
    block number during baseline) but with chunk_hash NULL, in which case
    the INSERT is a no-op due to ON CONFLICT — the linkage must be
    established with an UPDATE from the chunk rows themselves.
    """
    dst_cur.execute(
        "UPDATE l2_block b SET chunk_hash = c.hash FROM chunk c "
        "WHERE b.number BETWEEN c.start_block_number AND c.end_block_number "
        "AND c.index > (SELECT COALESCE(MAX(index), 0) - 2000 FROM chunk) "
        "AND (b.chunk_hash IS NULL OR b.chunk_hash <> c.hash)"
    )
    if dst_cur.rowcount:
        log.info("l2_block: linked chunk_hash on %d existing rows", dst_cur.rowcount)
    dst_cur.execute(
        "SELECT MIN(start_block_number), MAX(end_block_number) FROM chunk c "
        "WHERE c.index > (SELECT COALESCE(MAX(index), 0) - 2000 FROM chunk) "
        "AND NOT EXISTS (SELECT 1 FROM l2_block b WHERE b.chunk_hash = c.hash)"
    )
    lo, hi = dst_cur.fetchone()
    if lo is None:
        return 0
    # Include the parent block (start - 1) for hash-chain continuity.
    return copy_range(src, dst_cur, "l2_block", "number", max(lo - 2, 0), hi)


def ensure_proof_table(dst_cur):
    """Create the shadow-private quarantine table for imported mainnet proofs.

    `proof`/`proved_at` mirror the bundle table's column types (BYTEA /
    TIMESTAMP(0)) so applying them is a plain assignment. Not a goose
    migration: this table is canary-mode local state, created on demand.
    """
    dst_cur.execute(
        """
        CREATE TABLE IF NOT EXISTS remote_bundle_proof (
            bundle_hash     TEXT PRIMARY KEY,
            end_batch_index BIGINT NOT NULL,
            proof           BYTEA,
            proved_at       TIMESTAMP(0),
            synced_at       TIMESTAMPTZ NOT NULL DEFAULT now()
        )
        """
    )


def read_proof_cursor():
    """Last remote bundle index already scanned for proved bundles."""
    try:
        return int(PROOF_CURSOR_FILE.read_text().strip())
    except Exception:
        return 0


def write_proof_cursor(value):
    """Persist the proof-import cursor. Call only AFTER dst.commit()."""
    PROOF_CURSOR_FILE.parent.mkdir(parents=True, exist_ok=True)
    PROOF_CURSOR_FILE.write_text(f"{value}\n")


def read_proof_boundary():
    """Upper bound (exclusive) on end_batch_index for applying proofs.

    Read every cycle from the canary env file so flipping the boundary at the
    upgrade point needs no poll-sync restart. Missing file/key = infinity.
    """
    try:
        for line in CANARY_ENV_FILE.read_text().splitlines():
            if line.startswith("PROOF_IMPORT_MAX_END_BATCH="):
                return int(line.split("=", 1)[1].strip())
    except Exception:
        pass
    return None


def import_remote_proofs(src, dst_cur):
    """One proof-import cycle (canary mode only). Returns the new cursor.

    1) Upsert newly-proved remote bundles (proving_status=4) into the
       remote_bundle_proof quarantine table, advancing the persistent cursor.
    2) Apply quarantined proofs to local bundle rows below the boundary:
       only rows with proof IS NULL and rollup_status<>5 are touched, so
       relayer-finalized rows and locally-proved rows are never clobbered.
    """
    ensure_proof_table(dst_cur)
    cursor = read_proof_cursor()

    upsert_sql = (
        "INSERT INTO remote_bundle_proof (bundle_hash, end_batch_index, proof, proved_at) "
        "VALUES (%s, %s, %s, %s) ON CONFLICT (bundle_hash) DO NOTHING"
    )
    upserted = 0
    # Lookback: mainnet creates the bundle row first and proves it minutes
    # later, so a plain "index > cursor" high-watermark permanently skips any
    # bundle that was still unproven when the cursor moved past it. Re-scan a
    # small trailing window instead; ON CONFLICT DO NOTHING makes re-upserts
    # free.
    LOOKBACK = 200
    with src.cursor(name="sync_proof_cur") as cur:
        cur.itersize = 50
        cur.execute(
            "SELECT hash, end_batch_index, proof, proved_at FROM bundle "
            "WHERE proving_status = 4 AND index > %s ORDER BY index",
            (max(cursor - LOOKBACK, 0),),
        )
        while True:
            rows = cur.fetchmany(50)
            if not rows:
                break
            dst_cur.executemany(upsert_sql, rows)
            upserted += dst_cur.rowcount
    # Advance the cursor past PROVED bundles only — never past an unproven
    # row, or its later-arriving proof would fall behind the cursor (the
    # lookback above bounds how far the query reaches back, so the cursor
    # must not outrun unproven rows by more than LOOKBACK either; proved-only
    # advancement keeps that invariant).
    with src.cursor() as cur:
        cur.execute(
            "SELECT COALESCE(MAX(index), %s) FROM bundle WHERE proving_status = 4",
            (cursor,),
        )
        new_cursor = cur.fetchone()[0]
    if upserted:
        log.info("proof import: upserted %d remote proofs (cursor %d -> %d)",
                 upserted, cursor, new_cursor)

    boundary = read_proof_boundary()
    apply_sql = (
        "UPDATE bundle b SET proof = r.proof, proving_status = 4, proved_at = r.proved_at "
        "FROM remote_bundle_proof r "
        "WHERE b.hash = r.bundle_hash AND b.rollup_status <> 5 AND b.proof IS NULL"
    )
    if boundary is None:
        dst_cur.execute(apply_sql)
    else:
        dst_cur.execute(apply_sql + " AND b.end_batch_index < %s", (boundary,))
    if dst_cur.rowcount:
        log.info(
            "proof import: applied %d proofs to local bundles (boundary=%s)",
            dst_cur.rowcount,
            boundary if boundary is not None else "+inf",
        )
    return new_cursor


def baseline_sync(src_dsn, dst_dsn, start_batch, import_proofs=False):
    src = connect(src_dsn)
    dst = connect(dst_dsn)
    src_cur = src.cursor()
    dst_cur = dst.cursor()

    src_env = dsn_to_env(src_dsn)
    dst_env = dsn_to_env(dst_dsn)

    try:
        src_cur.execute("SELECT MAX(index) FROM batch")
        end_batch = src_cur.fetchone()[0]
        log.info("Baseline sync: batches %d -> %d", start_batch, end_batch)

        # Determine columns to copy for chunk/batch/bundle (exclude proof-related fields).
        sync_columns = {}
        for table in ("chunk", "batch", "bundle"):
            cols = get_columns(dst_cur, table)
            sync_columns[table] = [c for c in cols if c not in PROOF_RELATED_COLS.get(table, set())]

        # 1) batches
        copy_table_pipe(
            src_env,
            dst_env,
            "batch",
            f"index > {start_batch} AND index <= {end_batch}",
            columns=sync_columns["batch"],
        )

        # 2) chunks referenced by those batches (include parent chunk for continuity)
        src_cur.execute(
            "SELECT MIN(start_chunk_index), MAX(end_chunk_index) FROM batch "
            "WHERE index > %s AND index <= %s",
            (start_batch, end_batch),
        )
        min_chunk, max_chunk = src_cur.fetchone()
        if min_chunk is not None:
            copy_table_pipe(
                src_env,
                dst_env,
                "chunk",
                f"index > {min_chunk - 1} AND index <= {max_chunk}",
                columns=sync_columns["chunk"],
            )

            # 3) l2_block rows referenced by those chunks (include parent block)
            src_cur.execute(
                "SELECT MIN(start_block_number), MAX(end_block_number) FROM chunk "
                "WHERE index > %s AND index <= %s",
                (min_chunk - 1, max_chunk),
            )
            min_block, max_block = src_cur.fetchone()
            if min_block is not None:
                copy_table_pipe(
                    src_env,
                    dst_env,
                    "l2_block",
                    f"number > {min_block - 1} AND number <= {max_block}",
                )
                # l1_message is referenced by height, not number. Use block range as approximation.
                copy_table_pipe(
                    src_env,
                    dst_env,
                    "l1_message",
                    f"height > {min_block - 1} AND height <= {max_block}",
                )

        # 4) bundles that overlap the batch range
        src_cur.execute(
            "SELECT MIN(index), MAX(index) FROM bundle "
            "WHERE start_batch_index <= %s AND end_batch_index >= %s",
            (end_batch, start_batch + 1),
        )
        min_bundle, max_bundle = src_cur.fetchone()
        if min_bundle is not None:
            copy_table_pipe(
                src_env,
                dst_env,
                "bundle",
                f"index > {min_bundle - 1} AND index <= {max_bundle}",
                columns=sync_columns["bundle"],
            )

        # Reset mainnet rollup lifecycle state on freshly synced rows; the
        # shadow fork runs its own commit/finalize sequence.
        fork_committed = get_fork_committed_index()
        fixup_rollup_status(dst_cur, "batch", start_batch, end_batch, fork_committed)
        if min_bundle is not None:
            fixup_rollup_status(dst_cur, "bundle", min_bundle - 1, max_bundle, fork_committed)

        update_bundle_seq(dst_cur)
        proof_cursor = None
        if import_proofs:
            proof_cursor = import_remote_proofs(src, dst_cur)
        dst.commit()
        if proof_cursor is not None:
            write_proof_cursor(proof_cursor)
        log.info("Baseline sync complete")
    except Exception:
        dst.rollback()
        raise
    finally:
        src_cur.close()
        dst_cur.close()
        src.close()
        dst.close()


def poll_sync(src_dsn, dst_dsn, interval, import_proofs=False):
    src = connect(src_dsn)
    dst = connect(dst_dsn)

    def _ensure_cursors():
        nonlocal src, dst
        try:
            src.cursor().execute("SELECT 1")
            dst.cursor().execute("SELECT 1")
        except Exception as e:
            log.warning("Connection check failed, reconnecting: %s", e)
            try:
                src.close()
            except Exception:
                pass
            try:
                dst.close()
            except Exception:
                pass
            src = connect(src_dsn)
            dst = connect(dst_dsn)

    # Pre-compute columns to copy for chunk/batch/bundle once per run.
    dst_cur_for_cols = dst.cursor()
    sync_columns = {}
    for table in ("chunk", "batch", "bundle"):
        cols = get_columns(dst_cur_for_cols, table)
        sync_columns[table] = [c for c in cols if c not in PROOF_RELATED_COLS.get(table, set())]
    dst_cur_for_cols.close()

    try:
        fork_committed = None
        while True:
            _ensure_cursors()
            src_cur = src.cursor()
            dst_cur = dst.cursor()
            try:
                fork_committed = get_fork_committed_index(fallback=fork_committed)
                src_wm = get_watermarks(src_cur)
                dst_wm = get_watermarks(dst_cur)
                log.info("mainnet %s, shadow %s", src_wm, dst_wm)

                for table, pk in PK_COLUMNS.items():
                    delta = src_wm[table] - dst_wm[table]
                    if delta <= 0:
                        continue
                    if dst_wm[table] == 0 or delta > MAX_POLL_DELTA:
                        # Never bulk-backfill from poll mode: an empty or far-
                        # behind watermark means the DB was just reset, and
                        # copying millions of historical rows here would bury
                        # the coordinator in ancient tasks. Baseline sync
                        # (--init-from-batch) owns backfill.
                        log.warning(
                            "poll sync skipping %s: shadow watermark %s, mainnet %s "
                            "(delta %d > %d or empty) — run baseline sync instead",
                            table, dst_wm[table], src_wm[table], delta, MAX_POLL_DELTA,
                        )
                        continue
                    copy_range(
                        src,
                        dst_cur,
                        table,
                        pk,
                        dst_wm[table],
                        src_wm[table],
                        columns=sync_columns[table],
                    )
                    if table in ("batch", "bundle"):
                        fixup_rollup_status(
                            dst_cur,
                            table,
                            dst_wm[table],
                            src_wm[table],
                            fork_committed,
                        )

                update_bundle_seq(dst_cur)
                sync_parent_links(dst_cur)
                sync_l2_blocks(src, dst_cur)
                proof_cursor = None
                if import_proofs:
                    proof_cursor = import_remote_proofs(src, dst_cur)
                dst.commit()
                if proof_cursor is not None:
                    write_proof_cursor(proof_cursor)
                log.info("Poll cycle complete; sleeping %ds", interval)
            except Exception:
                dst.rollback()
                raise
            finally:
                src_cur.close()
                dst_cur.close()
            # Trap 23: mirror post-fork L1 queue rolling hashes onto the fork
            # so bundles popping new L1 messages can be finalized. Failures
            # here must never kill the DB sync loop.
            try:
                _sqh.run_once(log_fn=lambda m: log.info("queue-hash: %s", m))
            except Exception as e:
                log.warning("queue-hash sync failed: %s", e)
            time.sleep(interval)
    finally:
        src.close()
        dst.close()


def main():
    parser = argparse.ArgumentParser(description="Sync mainnet rollup metadata to shadow DB")
    parser.add_argument(
        "--init-from-batch",
        type=int,
        help="Baseline: copy mainnet batches from this index up to the current mainnet tip",
    )
    parser.add_argument(
        "--poll-interval",
        type=int,
        default=60,
        help="Polling interval in seconds (default 60)",
    )
    parser.add_argument(
        "--import-proofs",
        action="store_true",
        default=os.environ.get("SYNC_PROOFS", "") == "1",
        help="Canary mode: also import proved mainnet bundle proofs into the "
        "remote_bundle_proof quarantine table and apply them to local bundles "
        "below the canary boundary (env SYNC_PROOFS=1 works too)",
    )
    args = parser.parse_args()

    if args.init_from_batch is not None:
        baseline_sync(SRC_DSN, DST_DSN, args.init_from_batch, import_proofs=args.import_proofs)
    else:
        log.info("Starting poll mode with interval=%ds", args.poll_interval)
        poll_sync(SRC_DSN, DST_DSN, args.poll_interval, import_proofs=args.import_proofs)


if __name__ == "__main__":
    main()
