.PHONY: check update dev_docker clean

ZKP_VERSION=release-1220

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
	cd $(PWD)/bridge/ && go get -u github.com/scroll-tech/go-ethereum@staging && go mod tidy
	cd $(PWD)/common/ && go get -u github.com/scroll-tech/go-ethereum@staging && go mod tidy
	cd $(PWD)/coordinator/ && go get -u github.com/scroll-tech/go-ethereum@staging && go mod tidy
	cd $(PWD)/database/ && go get -u github.com/scroll-tech/go-ethereum@staging && go mod tidy
	cd $(PWD)/roller/ && go get -u github.com/scroll-tech/go-ethereum@staging && go mod tidy
	goimports -local $(PWD)/bridge/ -w .
	goimports -local $(PWD)/common/ -w .
	goimports -local $(PWD)/coordinator/ -w .
	goimports -local $(PWD)/database/ -w .
	goimports -local $(PWD)/roller/ -w .

dev_docker: ## build docker images for development/testing usages
	docker build -t scroll_l1geth ./common/docker/l1geth/
	docker build -t scroll_l2geth ./common/docker/l2geth/

test_zkp: ## Test zkp prove and verify, roller/prover generates the proof and coordinator/verifier verifies it
	mkdir -p test_params
	wget https://circuit-release.s3.us-west-2.amazonaws.com/circuit-release/${ZKP_VERSION}/test_params/params18 -O ./test_params/params18
	wget https://circuit-release.s3.us-west-2.amazonaws.com/circuit-release/${ZKP_VERSION}/test_params/params25 -O ./test_params/params25
	wget https://circuit-release.s3.us-west-2.amazonaws.com/circuit-release/${ZKP_VERSION}/test_seed -O test_seed
	rm -rf ./roller/assets/test_params && mv test_params ./roller/assets/ && mv test_seed ./roller/assets/
	cd ./roller && make test-gpu-prover
	rm -rf ./coordinator/assets/test_params && mv ./roller/assets/test_params ./coordinator/assets/
	cd ./coordinator && make test-gpu-verifier

clean: ## Empty out the bin folder
	@rm -rf build/bin
