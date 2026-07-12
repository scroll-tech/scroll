#!/usr/bin/env python3
"""
Collect shadow-proving pipeline metrics and append to a log.

Run every hour via cron during the 48-hour catch-up test.
"""

import json
import os
import subprocess
import sys
from datetime import datetime, timezone

import psycopg2
from psycopg2.extras import RealDictCursor

LOG_FILE = os.environ.get("SHADOW_METRICS_LOG", "/home/scroll/zzhang/scroll/tests/shadow-testing/.work/catchup-metrics.log")
DB_DSN = os.environ.get("DB_DSN", "postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup")
RPC = os.environ.get("ANVIL_RPC", "http://localhost:18545")
WORK_DIR = "/home/scroll/zzhang/scroll/tests/shadow-testing/.work"


def run_psql(sql):
    conn = psycopg2.connect(DB_DSN)
    cur = conn.cursor(cursor_factory=RealDictCursor)
    cur.execute(sql)
    row = cur.fetchone()
    cur.close()
    conn.close()
    return dict(row) if row else {}


def anvil_block():
    try:
        out = subprocess.run(
            ["cast", "block-number", "--rpc-url", RPC],
            capture_output=True, text=True, timeout=30
        )
        return int(out.stdout.strip())
    except Exception as e:
        return f"error: {e}"


def tail_errors(path, n=20):
    if not os.path.exists(path):
        return []
    try:
        out = subprocess.run(
            ["tail", "-n", str(n), path],
            capture_output=True, text=True, timeout=10
        )
        lines = out.stdout.strip().split("\n")
        return [l for l in lines if "ERROR" in l or "error" in l.lower()][-5:]
    except Exception:
        return []


def process_health():
    wanted = ["anvil", "coordinator_api", "coordinator_cron", "rollup_relayer"]
    health = {}
    try:
        out = subprocess.run(
            ["pgrep", "-a", "-f", "anvil|coordinator_api|coordinator_cron|rollup_relayer|prover"],
            capture_output=True, text=True, timeout=10
        )
        lines = out.stdout.strip().split("\n") if out.stdout.strip() else []
        for name in wanted:
            health[name] = any(name in line for line in lines)
        health["provers"] = sum(1 for line in lines if "prover" in line and "coordinator" not in line)
    except Exception as e:
        health["error"] = str(e)
    return health


def main():
    now = datetime.now(timezone.utc).isoformat()
    block = anvil_block()
    health = process_health()

    summary = run_psql("""
        SELECT
            (SELECT MAX(index) FROM bundle) AS max_bundle,
            (SELECT COUNT(*) FROM bundle WHERE rollup_status = 5) AS finalized_bundles,
            (SELECT COUNT(*) FROM bundle WHERE proving_status = 4) AS proved_bundles,
            (SELECT COUNT(*) FROM bundle WHERE rollup_status IN (3, 4) AND proving_status = 4) AS pending_finalize,
            (SELECT COUNT(*) FROM bundle WHERE proving_status = 1) AS unproved_bundles,
            (SELECT COUNT(*) FROM batch WHERE proving_status = 4) AS proved_batches,
            (SELECT COUNT(*) FROM batch WHERE proving_status = 1) AS pending_batches,
            (SELECT COUNT(*) FROM chunk WHERE proving_status = 4) AS proved_chunks,
            (SELECT COUNT(*) FROM chunk WHERE proving_status = 1) AS pending_chunks,
            (SELECT ROUND(AVG(proof_time_sec)::numeric, 2) FROM bundle WHERE proving_status = 4 AND proof_time_sec > 0) AS avg_bundle_proof_time,
            (SELECT ROUND(AVG(proof_time_sec)::numeric, 2) FROM batch WHERE proving_status = 4 AND proof_time_sec > 0) AS avg_batch_proof_time,
            (SELECT ROUND(AVG(proof_time_sec)::numeric, 2) FROM chunk WHERE proving_status = 4 AND proof_time_sec > 0) AS avg_chunk_proof_time
    """)

    # Convert Decimals to float for JSON serialization
    for k, v in list(summary.items()):
        if v is None:
            summary[k] = 0
        elif hasattr(v, "to_eng_string"):  # Decimal
            try:
                summary[k] = float(v)
            except Exception:
                summary[k] = str(v)
        else:
            summary[k] = int(v)

    prover_errors = {}
    for i in range(4):
        log_path = os.path.join(WORK_DIR, f"prover-{i}", "prover.log")
        prover_errors[f"prover-{i}"] = tail_errors(log_path, 50)

    relayer_errors = tail_errors(os.path.join(WORK_DIR, "relayer-mainnet-v4.log"), 50)
    coordinator_errors = tail_errors(os.path.join(WORK_DIR, "coordinator-api.log"), 50)

    record = {
        "timestamp": now,
        "anvil_block": block,
        "process_health": health,
        "summary": summary,
        "recent_errors": {
            "prover": prover_errors,
            "relayer": relayer_errors,
            "coordinator": coordinator_errors,
        },
    }

    os.makedirs(os.path.dirname(LOG_FILE), exist_ok=True)
    with open(LOG_FILE, "a") as f:
        f.write(json.dumps(record) + "\n")

    print(f"[{now}] max_bundle={summary.get('max_bundle')} finalized={summary.get('finalized_bundles')} pending_batches={summary.get('pending_batches')}")


if __name__ == "__main__":
    main()
