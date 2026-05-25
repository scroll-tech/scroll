.PHONY: lint
# NOTE: We use $$(CURDIR) instead of $$(PWD) throughout.
# CURDIR is Make's built-in variable that tracks the current Makefile's directory
# and respects 'make -C'. PWD is inherited from the shell and does NOT change with -C,
# which causes binaries to be built to wrong paths when invoked via make -C <subdir>.

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic -p 1 $(CURDIR)/...

lint: ## Lint the files - used for CI
	GOBIN=$(CURDIR)/build/bin go run ../build/lint.go