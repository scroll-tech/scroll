.PHONY: clean

build:
	GOBIN=$(PWD)/build/bin go build -o $(PWD)/build/bin/miner-api ./cmd

clean: ## Empty out the bin folder
	@rm -rf build/bin

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...