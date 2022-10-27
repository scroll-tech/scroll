.PHONY: check update dev_docker clean

check: ## The code's format and security checks.
	make -C bridge lint
	make -C common lint
	make -C coordinator lint
	make -C database lint
	make -C roller lint
	make -C roller libprover

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

dev_docker:
	docker build -t scroll_l1geth ./common/docker/l1geth/
	docker build -t scroll_l2geth ./common/docker/l2geth/.

clean: ## Empty out the bin folder
	@rm -rf build/bin