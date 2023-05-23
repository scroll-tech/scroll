.PHONY: lint clean

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(PWD)/...

lint: ## Lint the files - used for CI
	GOBIN=$(PWD)/build/bin go run ../build/lint.go
	cd libzkp/impl && cargo fmt --all -- --check && cargo clippy --release -- -D warnings

clean: ## Clean exited containers.
	docker ps -a | grep Exited | awk '{print $1}' | xargs docker stop && docker container prune -f