#!/usr/bin/env python3
"""
Generate final report for the 48-hour catch-up test.

Reads tests/shadow-testing/.work/catchup-metrics.log and prints:
- start/end time and duration
- Anvil block progression
- bundle creation vs finalization counts and rate
- average proof times
- bottleneck/failure summary
"""

import json
import os
import sys
from datetime import datetime, timezone

LOG_FILE = os.environ.get("SHADOW_METRICS_LOG", "/home/scroll/zzhang/scroll/tests/shadow-testing/.work/catchup-metrics.log")


def main():
    if not os.path.exists(LOG_FILE):
        print(f"Metrics log not found: {LOG_FILE}")
        sys.exit(1)

    records = []
    with open(LOG_FILE) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                records.append(json.loads(line))
            except json.JSONDecodeError:
                continue

    # Optional window filter: only count records at/after SHADOW_REPORT_START
    # (ISO timestamp), so the report covers the current 48h test rather than
    # older runs that share the same log file.
    report_start = os.environ.get("SHADOW_REPORT_START")
    if report_start:
        cutoff = datetime.fromisoformat(report_start)
        if cutoff.tzinfo is None:
            cutoff = cutoff.replace(tzinfo=timezone.utc)
        records = [r for r in records if datetime.fromisoformat(r["timestamp"]) >= cutoff]

    if not records:
        print("No metrics records found.")
        sys.exit(1)

    start = datetime.fromisoformat(records[0]["timestamp"])
    end = datetime.fromisoformat(records[-1]["timestamp"])
    duration_hours = (end - start).total_seconds() / 3600

    first_summary = records[0]["summary"]
    last_summary = records[-1]["summary"]

    anvil_start = records[0].get("anvil_block", 0)
    anvil_end = records[-1].get("anvil_block", 0)

    bundles_created = last_summary.get("max_bundle", 0) - first_summary.get("max_bundle", 0)
    bundles_finalized = last_summary.get("finalized_bundles", 0) - first_summary.get("finalized_bundles", 0)

    # Pull average proof times from the latest record
    avg_chunk = last_summary.get("avg_chunk_proof_time")
    avg_batch = last_summary.get("avg_batch_proof_time")
    avg_bundle = last_summary.get("avg_bundle_proof_time")

    # Collect all errors seen across records. CoordinatorEmptyProofData is the
    # SDK's idle-poll message ("no task available") logged at ERROR level — it
    # is benign and floods the report, so count it separately.
    all_errors = {
        "prover-0": [],
        "prover-1": [],
        "prover-2": [],
        "prover-3": [],
        "relayer": [],
        "coordinator": [],
    }
    idle_polls = 0
    for r in records:
        recent = r.get("recent_errors", {})
        for prover, errs in recent.get("prover", {}).items():
            for e in errs:
                if "CoordinatorEmptyProofData" in e:
                    idle_polls += 1
                else:
                    all_errors.setdefault(prover, []).append(e)
        all_errors["relayer"].extend(recent.get("relayer", []))
        all_errors["coordinator"].extend(recent.get("coordinator", []))

    print("# 48-Hour Shadow Catch-Up Report")
    print(f"- Duration: {duration_hours:.2f} hours ({start.isoformat()} -> {end.isoformat()})")
    print(f"- Anvil blocks: {anvil_start} -> {anvil_end} ({anvil_end - anvil_start} blocks)")
    print()
    print("## Bundle Throughput")
    print(f"- Bundles created (max_bundle delta): {bundles_created}")
    print(f"- Bundles finalized: {bundles_finalized}")
    print(f"- Finalization / creation ratio: {(bundles_finalized / bundles_created * 100):.1f}%" if bundles_created else "- No bundles created")
    print(f"- Latest pending bundles (unproved): {last_summary.get('unproved_bundles')}")
    print(f"- Latest pending batches: {last_summary.get('pending_batches')}")
    print(f"- Latest pending chunks: {last_summary.get('pending_chunks')}")
    print()
    print("## Average Proof Times (from latest record)")
    print(f"- Chunk: {avg_chunk}s" if avg_chunk is not None else "- Chunk: N/A")
    print(f"- Batch: {avg_batch}s" if avg_batch is not None else "- Batch: N/A")
    print(f"- Bundle: {avg_bundle}s" if avg_bundle is not None else "- Bundle: N/A")
    print()
    print("## Failures / Bottlenecks")
    total_errors = sum(len(v) for v in all_errors.values())
    print(f"- Total error samples collected: {total_errors}")
    print(f"- Idle-poll samples suppressed (CoordinatorEmptyProofData, benign): {idle_polls}")
    for source, errs in all_errors.items():
        unique = sorted(set(errs))[:5]
        if unique:
            print(f"- {source} ({len(errs)} samples):")
            for e in unique:
                print(f"  - {e[:200]}")


if __name__ == "__main__":
    main()
