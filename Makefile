.PHONY: fmt dev_docker build_test_docker run_test_docker clean update

L2GETH_TAG=scroll-v5.1.6

help: ## Display this help message
	@grep -h \
		-E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
update:
	go work sync
	cd $(PWD)/bridge-history-api/ && go get -u github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy
	cd $(PWD)/common/ && go get -u github.com/scroll-tech/go-ethereum@${L2GETH_TAG}&& go mod tidy
	cd $(PWD)/coordinator/ && go get -u github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy
	cd $(PWD)/database/ && go get -u github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy
	cd $(PWD)/prover/ && go get -u github.com/scroll-tech/go-ethereum@${L2GETH_TAG}&& go mod tidy
	cd $(PWD)/rollup/ && go get -u github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy
	cd $(PWD)/tests/integration-test/ && go get -u github.com/scroll-tech/go-ethereum@${L2GETH_TAG} && go mod tidy

lint: ## Run code formatting and linting
	@make -C rollup lint
	@make -C common lint
	@make -C coordinator lint
	@make -C database lint
	@make -C prover lint
	@make -C bridge-history-api lint

fmt: ## Format the code using goimports
	go work sync
	@for module in bridge-history-api common coordinator database prover rollup tests/integration-test; do \
		cd $(PWD)/$$module/ && go mod tidy; \
	done
	@goimports -local $(PWD)/ -w .

dev_docker: ## Build Docker images for development/testing
	docker build -t scroll_l1geth ./common/docker/l1geth/
	docker build -t scroll_l2geth ./common/docker/l2geth/

build_test_docker: ## Build Docker image for local testing on M1/M2 Silicon Mac
	docker build -t scroll_test_image -f ./build/dockerfiles/local_testing.Dockerfile $$(mktemp -d)

run_test_docker: ## Run Docker image for local testing on M1/M2 Silicon Mac
	docker run -it --rm --name scroll_test_container --network=host -v /var/run/docker.sock:/var/run/docker.sock -v $(PWD):/go/src/app scroll_test_image

clean: ## Remove the bin folder
	@rm -rf build/bin