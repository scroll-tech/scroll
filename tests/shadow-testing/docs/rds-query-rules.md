# Querying the Remote Mainnet DB (RDS via tunnel) — Rules

> Moved here from root `AGENTS.md` 2026-09-11 (single home for DB-access discipline).
> Production read-only DB: `192.168.1.108:15432/mainnet_rollup` (RDS tunnel; DSN in `local-secrets.md`). Table sizes as of 2026-08.

## Partial indexes — always predicate with `deleted_at IS NULL`

- **Every `l2_block` index is partial: `WHERE deleted_at IS NULL`.** Any query on `l2_block` (and `chunk`) without that predicate **cannot use any index** and seq-scans the whole heap (62 GB for `l2_block`) — even innocent-looking ones like `SELECT MAX(number) FROM l2_block`. Always write `... WHERE deleted_at IS NULL AND ...`. (Same pattern on `chunk`; its indexes are partial too.)

## Never run unbounded `count(*)` on `l2_block`

- Plain `count(*)` = parallel seq scan ≈ **2 minutes** (33.5M rows / 62 GB heap, 183 GB incl. TOAST). Even `count(*) WHERE deleted_at IS NULL` (parallel index-only scan) ran **>5 minutes** — the index itself is multi-GB. Just don't count this table.
- For row-count estimates use the planner stats instead — instant:

  ```sql
  SELECT reltuples::bigint FROM pg_class WHERE relname = 'l2_block';
  ```

- Bounded counts are fine when they carry the partial-index predicate plus a range on the indexed column: `SELECT count(*) FROM l2_block WHERE deleted_at IS NULL AND number > X AND number <= Y;` (an 8.4k-block range ≈ 8 s). `MAX(number) ... WHERE deleted_at IS NULL` is instant.
- `chunk` / `batch` / `bundle` are small enough (heaps ≤ 5 GB) that `count(*) WHERE deleted_at IS NULL` returns in seconds — but still always include the predicate, or you seq-scan.

## Session cost

- Each `psql` invocation pays ~0.7 s TLS+SCRAM setup and ~80 ms RTT per query (the tunnel forwards to a remote RDS; the LAN hop itself is <1 ms). Batch multiple statements into one session (one `psql` with several `-c`, or interactive) instead of one invocation per query.
- The follow-mode poll-sync daemon only issues indexed queries (`MAX(index)`, range-bound `COPY`, small proof windows), so it is unaffected — this page is for ad-hoc/manual queries.
- Related: follow-mode Trap 41 (`deleted_at IS NULL` discipline inside `sync-mainnet-db.py`).
