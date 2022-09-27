.PHONY: docker

docker:
	docker compose -f docker-compose.yml build

lint:
	cargo fmt --all && cargo clippy -- -D warnings
	shfmt -i 2 -sr -w scripts/*.sh

start:
	./scripts/stop.sh
	./scripts/start.sh

stop:
	./scripts/stop.sh
