REPO_ROOT_DIR=./.
IMAGE_VERSION=latest

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

backend-db-cli:
	go build -o $(PWD)/build/bin/backend-db-cli ./cmd/db_cli
	
backend-cross-msg-fetcher:
	go build -o $(PWD)/build/bin/backend-cross-msg-fetcher ./cmd/cross_msg_fetcher

backend-server:
	go build -o $(PWD)/build/bin/backend-server ./cmd/backend_server

db-docker:
	docker run --name backend-history-db -p 5444:5432 -e POSTGRES_PASSWORD=123456 -e POSTGRES_DB=test -d postgres

backend-docker:
	DOCKER_BUILDKIT=1 docker build -t scrolltech/backend-cross-msg-fetcher:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/docker/dockerfiles/backend-cross-msg-fetcher.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/backend-server:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/docker/dockerfiles/backend-server.Dockerfile
	DOCKER_BUILDKIT=1 docker build -t scrolltech/backend-db-cli:${IMAGE_VERSION} ${REPO_ROOT_DIR}/ -f ${REPO_ROOT_DIR}/docker/dockerfiles/backend-db-cli.Dockerfile