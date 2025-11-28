.PHONY: clean setup_db test_tool all check_vars

include conf/.make.env
GOOSE_CMD?=goose
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
	@echo "Please link sepolia or cloak-xen as conf"
	exit 1

check_vars: | conf
	@if [ -z "$(BEGIN_BLOCK)" ] || [ -z "$(END_BLOCK)" ]; then \
        echo "Error: BEGIN_BLOCK and END_BLOCK must be defined"; \
        echo "Usage: make import_data BEGIN_BLOCK=<start_block> END_BLOCK=<end_block>"; \
        exit 1; \
    fi

migration_blocks: $(BLOCK_PRE_MIGRATIONS)
ifneq ($(strip $(BLOCK_PRE_MIGRATIONS)),)
	GOOSE_MIGRATION_DIR=conf ${GOOSE_CMD} up-to 100
endif

setup_db: clean
	docker compose up --detach
	@echo "Waiting for PostgreSQL to be ready..."
	@for i in $$(seq 1 30); do \
        if nc -z localhost 5432 >/dev/null 2>&1; then \
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

reset_db:
	${GOOSE_CMD} down-to 0
	${GOOSE_CMD} up

test_tool:
	go build -o $(PWD)/build/bin/e2e_tool ../../rollup/tests/integration_tool

build/bin/e2e_tool: test_tool

import_data: build/bin/e2e_tool check_vars migration_blocks
	build/bin/e2e_tool --config conf/config.json ${BEGIN_BLOCK} ${END_BLOCK}

reimport_data: reset_db import_data

coordinator_setup:
	SCROLL_FORK_NAME=${SCROLL_FORK_NAME} $(MAKE) -C ../../coordinator localsetup
	cp -f conf/genesis.json ../../coordinator/build/bin/conf