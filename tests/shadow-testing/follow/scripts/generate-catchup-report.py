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
    # max_finalized_bundle is a monotonic watermark (shadow fork's finalized
    # bundle index); finalized_bundles per record is an interval count, so the
    # watermark delta is the robust total.
    bundles_finalized = last_summary.get("max_finalized_bundle", 0) - first_summary.get("max_finalized_bundle", 0)

    # Run metadata (written by 10-follow-up.sh) — used to split the report into
    # a catch-up phase (fork finalized index climbing to the mainnet bundle tip
    # recorded at run start) and a steady-state phase (everything after).
    run_env = {}
    env_path = os.environ.get(
        "SHADOW_FOLLOW_ENV",
        os.path.join(os.path.dirname(LOG_FILE), "follow-run.env"),
    )
    if os.path.exists(env_path):
        for line in open(env_path):
            if "=" in line:
                k, v = line.strip().split("=", 1)
                run_env[k] = v
    bundle_tip_at_start = run_env.get("MAINNET_BUNDLE_TIP_AT_START")
    bundle_tip_at_start = int(bundle_tip_at_start) if bundle_tip_at_start else None
    fork_hours_back = run_env.get("FORK_HOURS_BACK")

    catchup_end_idx = None
    if bundle_tip_at_start is not None:
        for i, r in enumerate(records):
            if r["summary"].get("max_finalized_bundle", 0) >= bundle_tip_at_start:
                catchup_end_idx = i
                break

    def phase_stats(recs):
        """Return (hours, bundles_created_delta, bundles_finalized_sum, avg proof times)."""
        if not recs:
            return 0.0, 0, 0, None, None, None
        hours = (datetime.fromisoformat(recs[-1]["timestamp"]) - datetime.fromisoformat(recs[0]["timestamp"])).total_seconds() / 3600
        created = recs[-1]["summary"].get("max_bundle", 0) - recs[0]["summary"].get("max_bundle", 0)
        finalized = recs[-1]["summary"].get("max_finalized_bundle", 0) - recs[0]["summary"].get("max_finalized_bundle", 0)

        def mean(key):
            vals = [r["summary"][key] for r in recs if r["summary"].get(key)]
            return round(sum(vals) / len(vals)) if vals else None

        return hours, created, finalized, mean("avg_chunk_proof_time"), mean("avg_batch_proof_time"), mean("avg_bundle_proof_time")

    catchup_stats = phase_stats(records[: catchup_end_idx + 1]) if catchup_end_idx is not None else None
    steady_stats = phase_stats(records[catchup_end_idx + 1 :]) if catchup_end_idx is not None else None

    # Mid-run upgrade split (written by 20-upgrade.sh): pre/post upgrade phases.
    upgrade_at_time = run_env.get("UPGRADE_AT_TIME")
    upgrade_at_batch = run_env.get("UPGRADE_AT_BATCH")
    upgrade_circuit = run_env.get("UPGRADE_CIRCUIT_VERSION")
    pre_upgrade_stats = post_upgrade_stats = None
    upgrade_dt = None
    if upgrade_at_time:
        try:
            upgrade_dt = datetime.fromisoformat(upgrade_at_time)
            if upgrade_dt.tzinfo is None:
                upgrade_dt = upgrade_dt.replace(tzinfo=timezone.utc)
            pre_recs = [r for r in records if datetime.fromisoformat(r["timestamp"]) < upgrade_dt]
            post_recs = [r for r in records if datetime.fromisoformat(r["timestamp"]) >= upgrade_dt]
            pre_upgrade_stats = phase_stats(pre_recs)
            post_upgrade_stats = phase_stats(post_recs)
        except ValueError:
            upgrade_dt = None

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
    if fork_hours_back is not None:
        print(f"- Fork: {fork_hours_back}h back, mainnet bundle tip at start: {bundle_tip_at_start}")
    print()
    if catchup_stats is not None:
        ch, cc, cf, c_chunk, c_batch, c_bundle = catchup_stats
        sh, sc, sf, s_chunk, s_batch, s_bundle = steady_stats
        print("## Catch-Up Phase (backlog at fork)")
        print(f"- Duration to catch up: {ch:.2f} hours")
        print(f"- Bundles finalized during catch-up: {cf}")
        print(f"- Avg proof times (interval means): chunk {c_chunk}s / batch {c_batch}s / bundle {c_bundle}s")
        print()
        print("## Steady-State Phase (following mainnet)")
        print(f"- Duration: {sh:.2f} hours")
        print(f"- Bundles created on mainnet: {sc}")
        print(f"- Bundles finalized on fork: {sf}")
        print(f"- Avg proof times (interval means): chunk {s_chunk}s / batch {s_batch}s / bundle {s_bundle}s")
        print()
    if upgrade_dt is not None:
        ph, pc, pf, p_chunk, p_batch, p_bundle = pre_upgrade_stats
        qh, qc, qf, q_chunk, q_batch, q_bundle = post_upgrade_stats
        print(f"## Mid-Run Upgrade (boundary batch {upgrade_at_batch}, circuit -> {upgrade_circuit})")
        print(f"- Upgrade at: {upgrade_dt.isoformat()}")
        print()
        print("### Pre-Upgrade Phase (old stack)")
        print(f"- Duration: {ph:.2f} hours")
        print(f"- Bundles finalized: {pf}")
        print(f"- Avg proof times (interval means): chunk {p_chunk}s / batch {p_batch}s / bundle {p_bundle}s")
        print()
        print("### Post-Upgrade Phase (new stack)")
        print(f"- Duration: {qh:.2f} hours")
        print(f"- Bundles created on mainnet: {qc}")
        print(f"- Bundles finalized on fork: {qf}")
        print(f"- Avg proof times (interval means): chunk {q_chunk}s / batch {q_batch}s / bundle {q_bundle}s")
        print()
    print("## Bundle Throughput (whole window)")
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
