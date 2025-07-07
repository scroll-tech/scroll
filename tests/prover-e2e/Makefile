.PHONY: clean setup_db test_tool

GOOSE_CMD?=goose
BEGIN_BLOCK?=10405848
END_BLOCK?=10405858

all: setup_db test_tool import_data

clean:
	docker compose down

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

import_data: test_tool
	build/bin/e2e_tool --config ./config.json --codec 7 ${BEGIN_BLOCK} ${END_BLOCK}