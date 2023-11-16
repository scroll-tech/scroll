.PHONY: lint
REPO_ROOT_DIR=./..
IMAGE_VERSION=latest

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

bridgehistoryapi-db-cli:
	go build -o $(PWD)/build/bin/bridgehistoryapi-db-cli ./cmd/db_cli
	
bridgehistoryapi-cross-message-fetcher:
	go build -o $(PWD)/build/bin/bridgehistoryapi-cross-message-fetcher ./cmd/cross_message_fetcher

bridgehistoryapi-server:
	go build -o $(PWD)/build/bin/bridgehistoryapi-server ./cmd/backend_server

db-docker:
	docker run --name bridgehistoryapi-history-db -p 5444:5432 -e POSTGRES_PASSWORD=123456 -e POSTGRES_DB=test -d postgres

bridgehistoryapi-docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/bridgehistoryapi-cross-message-fetcher:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/bridgehistoryapi-cross-message-fetcher.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/bridgehistoryapi-server:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/bridgehistoryapi-server.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/bridgehistoryapi-db-cli:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/build/dockerfiles/bridgehistoryapi-db-cli.Dockerfile