# Follow Mode Makefile
# Fork at the CURRENT mainnet tip and follow mainnet indefinitely:
# baseline sync -> anvil -> verifier -> coordinator -> provers -> relayer
# + poll-sync/sweeper/monitor daemons. See scripts/10-follow-up.sh.
#
# Usage:
#   make follow          Bring up the full follow-mode stack
#   make follow-old      Bring up Phase 1 of the mid-run upgrade test
#                        (old stack = production release, production verifier)
#   make follow-upgrade  Mid-run upgrade: cut over to the NEW zk stack
#                        (configs/mainnet-next.json) WITHOUT resetting the
#                        fork/DB — finalization stays continuous
#   make follow-canary   Bring up Phase A of the CANARY parallel-upgrade test
#                        (no local proving; production proofs imported and
#                        finalized through the forked production wrapper)
#   make canary-upgrade  Canary t1: register the NEW wrapper at boundary N and
#                        start the new coordinator+provers alongside the old
#                        (configs/mainnet-next.json; nothing is reset)
#   make canary-rollback Roll back the canary upgrade: restore the old wrapper
#                        at N and apply quarantined production proofs >= N
#   make follow-status   Latest monitor snapshot + finalize lag
#   make follow-report   Throughput report for the current run
#   make re-fork         Re-fork Anvil at latest block (recovery)
#   make follow-stop     Stop the follow-mode stack

SCRIPT_DIR    := $(shell cd "$(dir $(lastword $(MAKEFILE_LIST)))" && pwd)
CONFIG_FILE   := $(SCRIPT_DIR)/configs/mainnet.json
NEXT_CONFIG   := $(SCRIPT_DIR)/configs/mainnet-next.json
# .work is shared by both modes and stays at tests/shadow-testing/.work.
WORK_DIR      := $(SCRIPT_DIR)/../.work

DB_DSN        := $(shell jq -r '.db.dsn' $(CONFIG_FILE) 2>/dev/null)

.PHONY: follow follow-old follow-upgrade follow-canary canary-upgrade canary-rollback follow-stop follow-status follow-report re-fork help

follow:
	$(SCRIPT_DIR)/scripts/10-follow-up.sh

follow-old:
	$(SCRIPT_DIR)/scripts/10-follow-up.sh --skip-verifier

follow-upgrade:
	$(SCRIPT_DIR)/scripts/20-upgrade.sh --next-config $(NEXT_CONFIG)

follow-canary:
	$(SCRIPT_DIR)/scripts/10-follow-up.sh --import-proofs

canary-upgrade:
	$(SCRIPT_DIR)/scripts/30-canary-upgrade.sh --next-config $(NEXT_CONFIG)

canary-rollback:
	$(SCRIPT_DIR)/scripts/31-canary-rollback.sh

follow-stop:
	$(SCRIPT_DIR)/scripts/11-follow-stop.sh

follow-status:
	@echo "📊 Last monitor snapshot:"
	@last=$$(grep '^{' $(WORK_DIR)/catchup-metrics.log 2>/dev/null | tail -1); \
	if [ -z "$$last" ]; then echo "  (no metrics yet)"; \
	else printf '%s\n' "$$last" | (jq . 2>/dev/null || python3 -m json.tool); fi
	@echo ""
	@echo "Finalize lag (shadow DB):"
	@psql "$(DB_DSN)" -Atq -c "SELECT 'max_bundle=' || COALESCE(MAX(index),0) || ' max_finalized=' || COALESCE(MAX(index) FILTER (WHERE rollup_status=5),0) || ' lag=' || (COALESCE(MAX(index),0) - COALESCE(MAX(index) FILTER (WHERE rollup_status=5),0)) FROM bundle;" 2>/dev/null || echo "  (DB not accessible)"

follow-report:
	SHADOW_REPORT_START=$$(grep '^SHADOW_REPORT_START=' $(WORK_DIR)/follow-run.env 2>/dev/null | cut -d= -f2-) python3 $(SCRIPT_DIR)/scripts/generate-catchup-report.py

re-fork:
	$(SCRIPT_DIR)/scripts/re-fork.sh

help:
	@echo "Follow Mode Makefile (fork current mainnet tip, follow indefinitely)"
	@echo ""
	@echo "  make follow                    Bring up the full follow-mode stack"
	@echo "  make follow-old                Phase 1 of the upgrade test (old stack ="
	@echo "                                 production release + production verifier)"
	@echo "  make follow-upgrade            Cut over to the NEW zk stack mid-run"
	@echo "                                 (configs/mainnet-next.json; fork/DB untouched)"
	@echo "  make follow-canary             Phase A of the canary parallel-upgrade test"
	@echo "                                 (imported production proofs, no local proving)"
	@echo "  make canary-upgrade            Canary t1: new wrapper + new stack alongside"
	@echo "                                 the old (configs/mainnet-next.json)"
	@echo "  make canary-rollback           Roll back the canary upgrade (old wrapper"
	@echo "                                 restored at N, quarantined proofs applied)"
	@echo "  make follow-status             Latest monitor snapshot + finalize lag"
	@echo "  make follow-report             Throughput report for the current run"
	@echo "  make re-fork                   Re-fork Anvil at latest block (recovery)"
	@echo "  make follow-stop               Stop the follow-mode stack"
	@echo ""
	@echo "For snapshot replay mode (historical fork, fixed bundle range), use:"
	@echo "  make -C ../snapshot help"
