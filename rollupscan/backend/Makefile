.PHONY: docker

docker:
	docker compose -f docker-compose.yml build

import_mock_data:
	./scripts/import_mock_data.sh

lint:
	cargo fmt --all && cargo clippy -- -D warnings
	shfmt -i 2 -sr -w scripts/*.sh

login_db:
	psql postgres://scroll:scroll2022@localhost:5434/scroll

start:
	./scripts/stop.sh
	./scripts/start.sh

stop:
	./scripts/stop.sh
