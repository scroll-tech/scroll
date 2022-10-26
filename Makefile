.PHONY: check update docker clean install

check: ## The code's format and security checks.
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

docker:
	docker build -t scroll_l1geth ./common/docker/l1geth/
	docker build -t scroll_l2geth ./common/docker/l2geth/

install: 
	make -C bridge mock_abi
	make -C bridge bridge
	make -C bridge docker
	make -C coordinator coordinator
	make -C coordinator docker

test: ## run test
	make -C roller test-prover
	make -C roller test-gpu-prover
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 4 $(PWD)/database/...
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 4 $(PWD)/bridge/...
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 4 $(PWD)/common/...
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 4 $(PWD)/coordinator/...

clean: ## Empty out the bin folder
	@rm -rf build/bin