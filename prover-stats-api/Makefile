.PHONY: clean build test

build:
	GOBIN=$(PWD)/build/bin go build -o $(PWD)/build/bin/prover-stats-api ./cmd

clean: ## Empty out the bin folder
	@rm -rf build/bin

swag:
	swag init -g ./cmd/main.go

test:
	go test -v $(PWD)/...

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go
