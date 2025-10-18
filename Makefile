.PHONY: fmt dev_docker build_test_docker run_test_docker clean update

L2GETH_TAG=scroll-v5.9.7

help: ## Display this help message
	@grep -h \
		-E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
update: ## Update dependencies
	go work sync
	cd $(PWD)/bridge-history-api/ && go get github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy
	cd $(PWD)/common/ && go get github.com/scroll-tech/go-ethereum@${L2GETH_TAG}&& go mod tidy
	cd $(PWD)/coordinator/ && go get github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy
	cd $(PWD)/database/ && go get github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy
	cd $(PWD)/rollup/ && go get github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy
	cd $(PWD)/tests/integration-test/ && go get github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy

lint: ## The code's format and security checks
	make -C rollup lint
	make -C common lint
	make -C coordinator lint
	make -C database lint
	make -C bridge-history-api lint

fmt: ## Format the code
	go work sync
	cd $(PWD)/bridge-history-api/ && go mod tidy
	cd $(PWD)/common/ && go mod tidy
	cd $(PWD)/coordinator/ && go mod tidy
	cd $(PWD)/database/ && go mod tidy
	cd $(PWD)/rollup/ && go mod tidy
	cd $(PWD)/tests/integration-test/ && go mod tidy

	goimports -local scroll-tech/bridge-history-api/ -w .
	goimports -local scroll-tech/common/ -w .
	goimports -local scroll-tech/coordinator/ -w .
	goimports -local scroll-tech/database/ -w .
	goimports -local scroll-tech/rollup/ -w .
	goimports -local scroll-tech/tests/integration-test/ -w .

dev_docker: ## Build docker images for development/testing usages
	docker pull postgres
	docker build -t scroll_l1geth --platform linux/amd64 ./common/testcontainers/docker/l1geth/
	docker build -t scroll_l2geth --platform linux/amd64 ./common/testcontainers/docker/l2geth/

clean: ## Empty out the bin folder
	@rm -rf build/bin
