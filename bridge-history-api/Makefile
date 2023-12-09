.PHONY: lint
REPO_ROOT_DIR=./..
IMAGE_VERSION=latest

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

bridgehistoryapi-db-cli:
	go build -o $(PWD)/build/bin/bridgehistoryapi-db-cli ./cmd/db_cli
	
bridgehistoryapi-fetcher:
	go build -o $(PWD)/build/bin/bridgehistoryapi-fetcher ./cmd/fetcher

bridgehistoryapi-api:
	go build -o $(PWD)/build/bin/bridgehistoryapi-api ./cmd/api

redis-docker:
	-docker stop bridgehistoryapi-redis
	-docker rm bridgehistoryapi-redis
	docker run --name bridgehistoryapi-redis -d -p 6379:6379 redis:latest

reset-db-docker:
	-docker stop bridgehistoryapi-history-db
	-docker rm bridgehistoryapi-history-db
	docker run --name bridgehistoryapi-history-db -p 5444:5432 -e POSTGRES_PASSWORD=123456 -e POSTGRES_DB=test -d postgres
	go build -o $(PWD)/build/bin/bridgehistoryapi-db-cli ./cmd/db_cli & sleep 2
	$(PWD)/build/bin/bridgehistoryapi-db-cli reset

bridgehistoryapi-docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/bridgehistoryapi-fetcher:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/bridgehistoryapi-fetcher.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/bridgehistoryapi-api:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/bridgehistoryapi-api.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/bridgehistoryapi-db-cli:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/bridgehistoryapi-db-cli.Dockerfile