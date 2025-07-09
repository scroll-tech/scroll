.PHONY: clean setup_db test_tool all check_vars

GOOSE_CMD?=goose


all: setup_db test_tool import_data

clean:
	docker compose down

check_vars:
	@if [ -z "$(BEGIN_BLOCK)" ] || [ -z "$(END_BLOCK)" ]; then \
        echo "Error: BEGIN_BLOCK and END_BLOCK must be defined"; \
        echo "Usage: make import_data BEGIN_BLOCK=<start_block> END_BLOCK=<end_block>"; \
        exit 1; \
    fi

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
	GOOSE_MIGRATION_DIR=./ ${GOOSE_CMD} up-to 100

test_tool:
	go build -o $(PWD)/build/bin/e2e_tool ../../rollup/tests/integration_tool

build/bin/e2e_tool: test_tool

import_data_euclid: build/bin/e2e_tool check_vars
	build/bin/e2e_tool --config ./config.json --codec 7 ${BEGIN_BLOCK} ${END_BLOCK}

import_data: build/bin/e2e_tool check_vars
	build/bin/e2e_tool --config ./config.json --codec 8 ${BEGIN_BLOCK} ${END_BLOCK}    