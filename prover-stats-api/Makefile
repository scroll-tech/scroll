.PHONY: clean build test

build:
	GOBIN=$(PWD)/build/bin go build -o $(PWD)/build/bin/prover-stats-api ./cmd

clean: ## Empty out the bin folder
	@rm -rf build/bin

test:
	go test -v $(PWD)/...