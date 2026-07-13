# Shadow Testing Makefile — thin dispatcher.
#
# The two test modes live in their own directories, each with its own
# Makefile, scripts, configs and GUIDE:
#
#   follow/    Follow mode (primary): fork the current mainnet tip and follow
#              mainnet bundle production in real time. Default acceptance test
#              for prover/guest upgrades.
#   snapshot/  Snapshot replay mode: fork a historical block and prove/finalize
#              a fixed bundle range. Incident reproduction, single-bundle
#              debugging, Sepolia testing, codec-migration checks.
#
# Shared scripts and the relayer config template live in lib/.
# .work/ (logs, pidfiles, anvil state) is shared by both modes and stays here.

CONFIG        ?= mainnet
BUNDLE_RANGE  ?= 17297:17301

# --- Follow mode -------------------------------------------------------------
.PHONY: follow-up follow-stop follow-status follow-report re-fork

follow-up:
	$(MAKE) -C follow follow

follow-stop:
	$(MAKE) -C follow follow-stop

follow-status:
	$(MAKE) -C follow follow-status

follow-report:
	$(MAKE) -C follow follow-report

re-fork:
	$(MAKE) -C follow re-fork

# --- Snapshot replay mode ----------------------------------------------------
.PHONY: snapshot-all snapshot-env snapshot-prove snapshot-finalize \
        snapshot-sepolia-all snapshot-status snapshot-verify \
        snapshot-docker-all snapshot-docker-env snapshot-docker-prove \
        snapshot-docker-finalize snapshot-docker-stop snapshot-stop snapshot-clean

snapshot-all:
	$(MAKE) -C snapshot all CONFIG=$(CONFIG) BUNDLE_RANGE=$(BUNDLE_RANGE)

snapshot-env:
	$(MAKE) -C snapshot env CONFIG=$(CONFIG) BUNDLE_RANGE=$(BUNDLE_RANGE)

snapshot-prove:
	$(MAKE) -C snapshot prove CONFIG=$(CONFIG) BUNDLE_RANGE=$(BUNDLE_RANGE)

snapshot-finalize:
	$(MAKE) -C snapshot finalize CONFIG=$(CONFIG) BUNDLE_RANGE=$(BUNDLE_RANGE)

snapshot-sepolia-all:
	$(MAKE) -C snapshot sepolia-all CONFIG=$(CONFIG) BUNDLE_RANGE=$(BUNDLE_RANGE)

snapshot-status:
	$(MAKE) -C snapshot status CONFIG=$(CONFIG) BUNDLE_RANGE=$(BUNDLE_RANGE)

snapshot-verify:
	$(MAKE) -C snapshot verify CONFIG=$(CONFIG) BUNDLE_RANGE=$(BUNDLE_RANGE)

snapshot-docker-all:
	$(MAKE) -C snapshot docker-all CONFIG=$(CONFIG) BUNDLE_RANGE=$(BUNDLE_RANGE)

snapshot-docker-env:
	$(MAKE) -C snapshot docker-env CONFIG=$(CONFIG) BUNDLE_RANGE=$(BUNDLE_RANGE)

snapshot-docker-prove:
	$(MAKE) -C snapshot docker-prove CONFIG=$(CONFIG) BUNDLE_RANGE=$(BUNDLE_RANGE)

snapshot-docker-finalize:
	$(MAKE) -C snapshot docker-finalize CONFIG=$(CONFIG) BUNDLE_RANGE=$(BUNDLE_RANGE)

snapshot-docker-stop:
	$(MAKE) -C snapshot docker-stop CONFIG=$(CONFIG)

snapshot-stop:
	$(MAKE) -C snapshot stop CONFIG=$(CONFIG)

snapshot-clean:
	$(MAKE) -C snapshot clean CONFIG=$(CONFIG)

# --- Help --------------------------------------------------------------------
.PHONY: help

help:
	@echo "Shadow Testing — two modes, each with its own directory:"
	@echo ""
	@echo "Follow Mode (primary — fork current mainnet tip, follow indefinitely):"
	@echo "  make follow-up                 Bring up the full follow-mode stack"
	@echo "  make follow-status             Latest monitor snapshot + finalize lag"
	@echo "  make follow-report             Throughput report for the current run"
	@echo "  make re-fork                   Re-fork Anvil at latest block (recovery)"
	@echo "  make follow-stop               Stop the follow-mode stack"
	@echo "  (or: make -C follow <target>; guide: follow/GUIDE.md)"
	@echo ""
	@echo "Snapshot Replay Mode (historical fork, fixed bundle range):"
	@echo "  make snapshot-all        CONFIG=<name> BUNDLE_RANGE=<start:end>  Full pipeline"
	@echo "  make snapshot-env        CONFIG=<name> BUNDLE_RANGE=<start:end>  Setup Anvil + DB"
	@echo "  make snapshot-prove      CONFIG=<name> BUNDLE_RANGE=<start:end>  Prove bundles"
	@echo "  make snapshot-finalize   CONFIG=<name> BUNDLE_RANGE=<start:end>  Finalize bundles"
	@echo "  make snapshot-status     CONFIG=<name> BUNDLE_RANGE=<start:end>  Check status"
	@echo "  make snapshot-verify     CONFIG=<name> BUNDLE_RANGE=<start:end>  Verify on-chain"
	@echo "  make snapshot-sepolia-all CONFIG=sepolia BUNDLE_RANGE=<start:end> Sepolia pipeline"
	@echo "  make snapshot-docker-{all,env,prove,finalize,stop}               Docker pipeline"
	@echo "  make snapshot-stop / snapshot-clean                              Stop / cleanup"
	@echo "  (or: make -C snapshot <target>; guide: snapshot/GUIDE.md)"
