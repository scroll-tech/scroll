.PHONY: lint

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 4 $(PWD)/...

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go
