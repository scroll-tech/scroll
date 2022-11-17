.PHONY: check update dev_docker clean

help: ## Display this help message
	@grep -h \
		-E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

lint: ## The code's format and security checks.
	make -C bridge lint
	make -C common lint
	make -C coordinator lint
	make -C database lint
	make -C roller lint

update: ## update dependencies
	go work sync
	cd $(PWD)/bridge/ && go mod tidy
	cd $(PWD)/common/ && go mod tidy
	cd $(PWD)/coordinator/ && go mod tidy
	cd $(PWD)/database/ && go mod tidy
	cd $(PWD)/roller/ && go mod tidy
	goimports -local $(PWD)/bridge/ -w .
	goimports -local $(PWD)/common/ -w .
	goimports -local $(PWD)/coordinator/ -w .
	goimports -local $(PWD)/database/ -w .
	goimports -local $(PWD)/roller/ -w .

dev_docker: ## build docker images for development/testing usages
	docker build -t scroll_l1geth ./common/docker/l1geth/
	docker build -t scroll_l2geth ./common/docker/l2geth/

clean: ## Empty out the bin folder
	@rm -rf build/bin
