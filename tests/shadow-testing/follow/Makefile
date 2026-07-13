# Follow Mode Makefile
# Fork at the CURRENT mainnet tip and follow mainnet indefinitely:
# baseline sync -> anvil -> verifier -> coordinator -> provers -> relayer
# + poll-sync/sweeper/monitor daemons. See scripts/10-follow-up.sh.
#
# Usage:
#   make follow          Bring up the full follow-mode stack
#   make follow-status   Latest monitor snapshot + finalize lag
#   make follow-report   Throughput report for the current run
#   make re-fork         Re-fork Anvil at latest block (recovery)
#   make follow-stop     Stop the follow-mode stack

SCRIPT_DIR    := $(shell cd "$(dir $(lastword $(MAKEFILE_LIST)))" && pwd)
CONFIG_FILE   := $(SCRIPT_DIR)/configs/mainnet.json
# .work is shared by both modes and stays at tests/shadow-testing/.work.
WORK_DIR      := $(SCRIPT_DIR)/../.work

DB_DSN        := $(shell jq -r '.db.dsn' $(CONFIG_FILE) 2>/dev/null)

.PHONY: follow follow-stop follow-status follow-report re-fork help

follow:
	$(SCRIPT_DIR)/scripts/10-follow-up.sh

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
	@echo "  make follow-status             Latest monitor snapshot + finalize lag"
	@echo "  make follow-report             Throughput report for the current run"
	@echo "  make re-fork                   Re-fork Anvil at latest block (recovery)"
	@echo "  make follow-stop               Stop the follow-mode stack"
	@echo ""
	@echo "For snapshot replay mode (historical fork, fixed bundle range), use:"
	@echo "  make -C ../snapshot help"
