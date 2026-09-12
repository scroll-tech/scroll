#!/usr/bin/env bash
# Sweep stale "proving" rows left behind when a prover dies mid-task.
# The coordinator does not reliably time out chunk/batch tasks, and a single
# stale row blocks the whole finalize queue (chunks -> batch -> bundle -> finalize).
#
# total_attempts is also reset: the coordinator skips tasks with
# total_attempts >= 5 (see chunk.go GetUnassignedChunk), so an outage that
# burns all attempts (e.g. Trap 22 block-hash failures) would permanently
# starve the task even after proving_status is reset.
#
# Thresholds (generous vs. observed proof times):
#   chunk  ~5 min  -> reset after 30 min
#   batch  ~1 min  -> reset after 30 min
#   bundle up to ~90 min (30-batch bundles) -> reset after 180 min
#
# Usage: ./sweep-stale-proving.sh [DSN]
set -euo pipefail

DSN="${1:-${SHADOW_DSN:-postgresql://postgres:shadow_pass@localhost:5433/shadow_rollup}}"

psql "$DSN" -Atq <<'SQL'
UPDATE chunk  SET proving_status = 1, active_attempts = 0, total_attempts = 0
  WHERE proving_status = 2 AND updated_at < now() - interval '30 minutes';
UPDATE batch  SET proving_status = 1, active_attempts = 0, total_attempts = 0
  WHERE proving_status = 2 AND updated_at < now() - interval '30 minutes';
UPDATE bundle SET proving_status = 1, active_attempts = 0, total_attempts = 0
  WHERE proving_status = 2 AND updated_at < now() - interval '180 minutes';
SQL
