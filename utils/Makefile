.PHONY: lint

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go
	
bump_l2geth:
	go get -v -u github.com/scroll-tech/go-ethereum/cmd/geth@zkrollup
	go mod tidy
