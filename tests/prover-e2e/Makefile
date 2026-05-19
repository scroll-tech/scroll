.PHONY: clean setup_db test_tool all check_vars gen-config
# NOTE: We use $(CURDIR) instead of $(PWD) throughout.
# CURDIR is Make's built-in variable that tracks the current Makefile's directory
# and respects 'make -C'. PWD is inherited from the shell and does NOT change with -C,
# which causes binaries to be built to wrong paths when invoked via make -C <subdir>.

# Load test scenario settings (BEGIN_BLOCK, END_BLOCK, SCROLL_FORK_NAME, SCROLL_ZKVM_VERSION)
include conf/.make.env

# Load DB_PORT from .env (single source of truth for DB port)
include .env
export DB_PORT

GOOSE_CMD?=goose
GOOSE_DRIVER=postgres
GOOSE_DBSTRING=postgresql://dev:dev@localhost:${DB_PORT}/scroll?sslmode=disable
GOOSE_MIGRATION_DIR=../../database/migrate/migrations
export GOOSE_DRIVER GOOSE_DBSTRING GOOSE_MIGRATION_DIR

ifndef BEGIN_BLOCK
$(error BEGIN_BLOCK is not set. Define it in .make.env or pass BEGIN_BLOCK=<start_block>)
endif
ifndef END_BLOCK
$(error END_BLOCK is not set. Define it in .make.env or pass END_BLOCK=<end_block>)
endif

BLOCK_PRE_MIGRATIONS := $(wildcard conf/*.sql)
.OPTIONAL: $(BLOCK_PRE_MIGRATIONS)

all: setup_db test_tool import_data

clean:
	docker compose down

conf:
	@echo "Please link one of the scenario dirs (e.g., mainnet-galileoV2) as conf"
	exit 1

check_vars: | conf
	@if [ -z "$(BEGIN_BLOCK)" ] || [ -z "$(END_BLOCK)" ]; then \
        echo "Error: BEGIN_BLOCK and END_BLOCK must be defined"; \
        echo "Usage: make import_data BEGIN_BLOCK=<start_block> END_BLOCK=<end_block>"; \
        exit 1; \
    fi

# Generate config files with actual DB_PORT substituted from .env.
# Source configs use __DB_PORT__ placeholder; this target replaces it.
gen-config:
	@echo "Generating config files with DB_PORT=$(DB_PORT)..."
	@for f in conf/config.json conf/config.template.json; do \
		if [ -f "$$f" ]; then \
			sed -i 's/__DB_PORT__/$(DB_PORT)/g' "$$f"; \
			echo "  Updated $$f"; \
		fi; \
	done

migration_blocks: $(BLOCK_PRE_MIGRATIONS)
ifneq ($(strip $(BLOCK_PRE_MIGRATIONS)),)
		GOOSE_MIGRATION_DIR=conf ${GOOSE_CMD} up-to 100
endif

setup_db: clean gen-config
	docker compose up --detach
	@echo "Waiting for PostgreSQL to be ready..."
	@for i in $$(seq 1 30); do \
        if nc -z localhost $(DB_PORT) >/dev/null 2>&1; then \
            echo "PostgreSQL port is open!"; \
            sleep 2; \
            break; \
        fi; \
        echo "Waiting for PostgreSQL to start... ($$i/30)"; \
        sleep 2; \
        if [ $$i -eq 30 ]; then \
            echo "Timed out waiting for PostgreSQL to start"; \
            exit 1; \
        fi; \
    done
	${GOOSE_CMD} up

reset_db: gen-config
	${GOOSE_CMD} down-to 0
	${GOOSE_CMD} up

test_tool:
	go build -o $(CURDIR)/build/bin/e2e_tool ../../rollup/tests/integration_tool

build/bin/e2e_tool: test_tool

import_data: build/bin/e2e_tool check_vars migration_blocks
	build/bin/e2e_tool --config conf/config.json ${BEGIN_BLOCK} ${END_BLOCK}

reimport_data: reset_db import_data

coordinator_setup: gen-config
	SCROLL_ZKVM_VERSION=${SCROLL_ZKVM_VERSION} SCROLL_FORK_NAME=${SCROLL_FORK_NAME} $(MAKE) -C ../../coordinator localsetup
	cp -f conf/genesis.json ../../coordinator/build/bin/conf
# Also substitute DB_PORT in the coordinator config that was just copied
	@sed -i 's/__DB_PORT__/$(DB_PORT)/g' ../../coordinator/build/bin/conf/config.template.json
	@if [ -f ../../coordinator/build/bin/conf/config.json ]; then \
		sed -i 's/__DB_PORT__/$(DB_PORT)/g' ../../coordinator/build/bin/conf/config.json; \
	fi
