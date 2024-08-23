.PHONY: fmt dev_docker build_test_docker run_test_docker clean update

L2GETH_TAG=scroll-v5.6.3

help: ## Display this help message
	@grep -h \
		-E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
update: ## Update dependencies
	go work sync
	cd $(PWD)/bridge-history-api/ && go get -u github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy
	cd $(PWD)/common/ && go get -u github.com/scroll-tech/go-ethereum@${L2GETH_TAG}&& go mod tidy
	cd $(PWD)/coordinator/ && go get -u github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy
	cd $(PWD)/database/ && go get -u github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy
	cd $(PWD)/rollup/ && go get -u github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy
	cd $(PWD)/tests/integration-test/ && go get -u github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy

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

	goimports -local $(PWD)/bridge-history-api/ -w .
	goimports -local $(PWD)/common/ -w .
	goimports -local $(PWD)/coordinator/ -w .
	goimports -local $(PWD)/database/ -w .
	goimports -local $(PWD)/rollup/ -w .
	goimports -local $(PWD)/tests/integration-test/ -w .

dev_docker: ## Build docker images for development/testing usages
	docker pull postgres
	docker build -t scroll_l1geth --platform linux/amd64 ./common/testcontainers/docker/l1geth/
	docker build -t scroll_l2geth --platform linux/amd64 ./common/testcontainers/docker/l2geth/

clean: ## Empty out the bin folder
	@rm -rf build/bin
