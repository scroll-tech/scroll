.PHONY: clean

build:
	GOBIN=$(PWD)/build/bin go build -o $(PWD)/build/bin/miner-api ./cmd

clean: ## Empty out the bin folder
	@rm -rf build/bin