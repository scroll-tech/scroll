# Snapshot Replay Mode Makefile
# Historical fork + fixed bundle range (env/prove/finalize pipeline).
# Usage:
#   make all CONFIG=mainnet BUNDLE_RANGE=17297:17301
#   make env  CONFIG=mainnet BUNDLE_RANGE=17297:17301
#   make prove CONFIG=mainnet BUNDLE_RANGE=17297:17301
#   make finalize CONFIG=mainnet BUNDLE_RANGE=17297:17301
#   make stop

CONFIG        ?= mainnet
BUNDLE_RANGE  ?= 17297:17301
SCRIPT_DIR    := $(shell cd "$(dir $(lastword $(MAKEFILE_LIST)))" && pwd)
CONFIG_FILE   := $(SCRIPT_DIR)/configs/$(CONFIG).json
# .work is shared by both modes and stays at tests/shadow-testing/.work.
WORK_DIR      := $(SCRIPT_DIR)/../.work
LIB_DIR       := $(SCRIPT_DIR)/../lib

# Extract values from JSON config (requires jq)
FORK_URL      := $(shell jq -r '.fork.url' $(CONFIG_FILE) 2>/dev/null)
FORK_BLOCK    := $(shell jq -r '.fork.block_number' $(CONFIG_FILE) 2>/dev/null)
ANVIL_RPC     := $(shell jq -r '.fork.anvil_rpc' $(CONFIG_FILE) 2>/dev/null)
DB_DSN        := $(shell jq -r '.db.dsn' $(CONFIG_FILE) 2>/dev/null)
GENESIS       := $(shell jq -r '.genesis' $(CONFIG_FILE) 2>/dev/null)
SCROLL_CHAIN  := $(shell jq -r '.contracts.scroll_chain' $(CONFIG_FILE) 2>/dev/null)
L1_MSG_QUEUE  := $(shell jq -r '.contracts.l1_message_queue_v2' $(CONFIG_FILE) 2>/dev/null)
ROLLUP_VERIF  := $(shell jq -r '.contracts.rollup_verifier' $(CONFIG_FILE) 2>/dev/null)
DEPLOYED_VERIF:= $(shell jq -r '.contracts.deployed_verifier' $(CONFIG_FILE) 2>/dev/null)
OWNER         := $(shell jq -r '.contracts.owner' $(CONFIG_FILE) 2>/dev/null)
PROVER_EOA    := $(shell jq -r '.accounts.prover_eoa' $(CONFIG_FILE) 2>/dev/null)
COMMIT_EOA    := $(shell jq -r '.accounts.commit_eoa' $(CONFIG_FILE) 2>/dev/null)
LAST_FINALIZED:= $(shell jq -r '.reset.last_finalized_batch_index' $(CONFIG_FILE) 2>/dev/null)
LAST_COMMITTED:= $(shell jq -r '.reset.last_committed_batch_index // empty' $(CONFIG_FILE) 2>/dev/null)
NEXT_QUEUE    := $(shell jq -r '.reset.next_unfinalized_queue_index' $(CONFIG_FILE) 2>/dev/null)
CODEC_VERSION := $(shell jq -r '.reset.codec_version' $(CONFIG_FILE) 2>/dev/null)

# ─── Validation ──────────────────────────────────────────────────────────────
.PHONY: check-deps

check-deps:
	@command -v jq >/dev/null 2>&1 || { echo "jq is required"; exit 1; }
	@command -v cast >/dev/null 2>&1 || { echo "cast (Foundry) is required"; exit 1; }
	@test -f $(CONFIG_FILE) || { echo "Config file not found: $(CONFIG_FILE)"; exit 1; }

# ─── Environment Setup ───────────────────────────────────────────────────────
.PHONY: env anvil-up db-reset

env: check-deps anvil-up db-reset
	@echo ""
	@echo "✅ Environment ready for $(CONFIG)"
	@echo "   Anvil RPC: $(ANVIL_RPC)"
	@echo "   Bundle range: $(BUNDLE_RANGE)"

anvil-up: check-deps
	@echo "🚀 Setting up Anvil fork ($(CONFIG))..."
	$(LIB_DIR)/01-setup-anvil.sh \
		--fork-url "$(FORK_URL)" \
		--fork-block "$(FORK_BLOCK)" \
		--anvil-rpc "$(ANVIL_RPC)" \
		--state-file "$(WORK_DIR)/anvil-$(CONFIG).state.json" \
		--last-finalized "$(LAST_FINALIZED)" \
		--last-committed "$(LAST_COMMITTED)" \
		--next-queue "$(NEXT_QUEUE)" \
		--deployed-verifier "$(DEPLOYED_VERIF)" \
		--prover-eoa "$(PROVER_EOA)" \
		--commit-eoa "$(COMMIT_EOA)" \
		--owner "$(OWNER)" \
		--scroll-chain "$(SCROLL_CHAIN)" \
		--l1-msg-queue "$(L1_MSG_QUEUE)" \
		--rollup-verifier "$(ROLLUP_VERIF)"

db-reset: check-deps
	@echo "🗄️  Preparing shadow DB..."
	$(SCRIPT_DIR)/scripts/02-prepare-db.sh \
		--db-dsn "$(DB_DSN)" \
		--bundle-range "$(BUNDLE_RANGE)"

# ─── Proving Phase ────────────────────────────────────────────────────────────
.PHONY: prove prover-up wait-proofs

prove: prover-up wait-proofs

prover-up:
	@echo "🔬 Starting provers..."
	$(LIB_DIR)/04-prover-up.sh \
		--config "$(CONFIG)" \
		--bundle-range "$(BUNDLE_RANGE)"

wait-proofs:
	@echo "⏳ Waiting for proofs to complete..."
	$(SCRIPT_DIR)/scripts/05-wait-for-proofs.sh \
		--db-dsn "$(DB_DSN)" \
		--bundle-range "$(BUNDLE_RANGE)"

# ─── Finalization Phase ──────────────────────────────────────────────────────
.PHONY: finalize relayer-up wait-finalize

finalize: relayer-up wait-finalize

relayer-up:
	@echo "📤 Starting relayer..."
	$(LIB_DIR)/06-run-relayer.sh \
		--config "$(CONFIG)"

wait-finalize:
	@echo "⏳ Waiting for finalization..."
	$(SCRIPT_DIR)/scripts/07-wait-for-finalize.sh \
		--anvil-rpc "$(ANVIL_RPC)" \
		--scroll-chain "$(SCROLL_CHAIN)" \
		--bundle-range "$(BUNDLE_RANGE)"

# ─── Sepolia-specific Pipeline ───────────────────────────────────────────────
.PHONY: sepolia-all

sepolia-all: env prove finalize
	@echo ""
	@echo "🎉 Sepolia shadow test complete!"
	@echo "   Bundles: $(BUNDLE_RANGE)"
	@echo "   Anvil RPC: $(ANVIL_RPC)"

# ─── Full Pipeline ────────────────────────────────────────────────────────────
.PHONY: all

all: env prove finalize
	@echo ""
	@echo "🎉 Shadow test complete for $(CONFIG)!"
	@echo "   Bundles: $(BUNDLE_RANGE)"
	@echo "   Anvil RPC: $(ANVIL_RPC)"

# ─── Status & Verification ───────────────────────────────────────────────────
.PHONY: status verify

status: check-deps
	@echo "📊 Current status:"
	@echo ""
	@echo "Anvil lastFinalizedBatchIndex:"
	@cast call "$(SCROLL_CHAIN)" "lastFinalizedBatchIndex()(uint256)" --rpc-url "$(ANVIL_RPC)" 2>/dev/null || echo "  (Anvil not running)"
	@echo ""
	@echo "Bundle proving status:"
	@psql "$(DB_DSN)" -c "SELECT index, proving_status, rollup_status FROM bundle WHERE index BETWEEN $(subst :, AND ,$(BUNDLE_RANGE)) ORDER BY index;" 2>/dev/null || echo "  (DB not accessible)"

verify: check-deps
	@echo "🔍 Verifying on-chain finalization..."
	$(SCRIPT_DIR)/scripts/07-wait-for-finalize.sh \
		--anvil-rpc "$(ANVIL_RPC)" \
		--scroll-chain "$(SCROLL_CHAIN)" \
		--bundle-range "$(BUNDLE_RANGE)" \
		--verify-only

# ─── Docker-Orchestrated Pipeline ──────────────────────────────────────────────
.PHONY: docker-all docker-env docker-prove docker-finalize docker-stop

docker-all:
	@echo "🐳 Docker-orchestrated full pipeline..."
	$(SCRIPT_DIR)/scripts/08-docker-orchestrate.sh \
		--config $(CONFIG) \
		--bundle-range $(BUNDLE_RANGE) \
		--phase all

docker-env:
	@echo "🐳 Docker environment setup..."
	$(SCRIPT_DIR)/scripts/08-docker-orchestrate.sh \
		--config $(CONFIG) \
		--bundle-range $(BUNDLE_RANGE) \
		--phase env

docker-prove:
	@echo "🐳 Docker proving phase..."
	$(SCRIPT_DIR)/scripts/08-docker-orchestrate.sh \
		--config $(CONFIG) \
		--bundle-range $(BUNDLE_RANGE) \
		--phase prove

docker-finalize:
	@echo "🐳 Docker finalization phase..."
	$(SCRIPT_DIR)/scripts/08-docker-orchestrate.sh \
		--config $(CONFIG) \
		--bundle-range $(BUNDLE_RANGE) \
		--phase finalize

docker-stop:
	@echo "🛑 Stopping Docker services..."
	-docker compose -f $(SCRIPT_DIR)/../docker-compose.yml down 2>/dev/null || true
	@if [ -f $(WORK_DIR)/anvil-$(CONFIG).pid ]; then \
		kill $$(cat $(WORK_DIR)/anvil-$(CONFIG).pid) 2>/dev/null || true; \
		rm -f $(WORK_DIR)/anvil-$(CONFIG).pid; \
	fi
	@echo "Done."

# ─── Cleanup ─────────────────────────────────────────────────────────────────
.PHONY: stop clean

stop: docker-stop
	@echo "🛑 Stopping bare-metal services..."
	@if [ -f $(WORK_DIR)/anvil.pid ]; then \
		kill $$(cat $(WORK_DIR)/anvil.pid) 2>/dev/null || true; \
		rm -f $(WORK_DIR)/anvil.pid; \
	fi
	@echo "Done."

clean: stop
	@echo "🧹 Cleaning up..."
	rm -rf $(WORK_DIR)/*
	@echo "Done."

# ─── Help ────────────────────────────────────────────────────────────────────
.PHONY: help

help:
	@echo "Snapshot Replay Mode Makefile (historical fork, fixed bundle range)"
	@echo ""
	@echo "  make all      CONFIG=<name> BUNDLE_RANGE=<start:end>  Full pipeline"
	@echo "  make env      CONFIG=<name> BUNDLE_RANGE=<start:end>  Setup Anvil + DB"
	@echo "  make prove    CONFIG=<name> BUNDLE_RANGE=<start:end>  Prove bundles"
	@echo "  make finalize CONFIG=<name> BUNDLE_RANGE=<start:end>  Finalize bundles"
	@echo "  make status   CONFIG=<name> BUNDLE_RANGE=<start:end>  Check status"
	@echo "  make verify   CONFIG=<name> BUNDLE_RANGE=<start:end>  Verify on-chain"
	@echo ""
	@echo "Docker Targets (recommended):"
	@echo "  make docker-all      CONFIG=<name> BUNDLE_RANGE=<start:end>  Full docker pipeline"
	@echo "  make docker-env      CONFIG=<name> BUNDLE_RANGE=<start:end>  Setup env (postgres + anvil + import)"
	@echo "  make docker-prove    CONFIG=<name> BUNDLE_RANGE=<start:end>  Prove (coordinator + prover)"
	@echo "  make docker-finalize CONFIG=<name> BUNDLE_RANGE=<start:end>  Finalize (relayer)"
	@echo "  make docker-stop                                        Stop docker services"
	@echo ""
	@echo "  make stop                                              Stop all services"
	@echo "  make clean                                             Stop + cleanup"
	@echo ""
	@echo "Configs: $(shell ls $(SCRIPT_DIR)/configs/*.json 2>/dev/null | xargs -n1 basename | sed 's/.json//g' | xargs)"
	@echo ""
	@echo "Example (docker):"
	@echo "  make docker-all CONFIG=mainnet BUNDLE_RANGE=17302:17305"
	@echo ""
	@echo "Example (bare-metal):"
	@echo "  make all CONFIG=mainnet BUNDLE_RANGE=17297:17301"
	@echo ""
	@echo "For follow mode (live mainnet), use: make -C ../follow help"
