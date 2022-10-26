.PHONY: clean go-imports

go-imports:
	$(GOBIN)/goimports -local database/ -w . && $(GOBIN)/goimports -local roller/ -w . && $(GOBIN)/goimports -local coordinator/ -w . && $(GOBIN)/goimports -local bridge/ -w . && $(GOBIN)/goimports -local common/ -w .
work-sync:
	go work sync

test-geth-docker:
	docker build -t scroll_l1geth ./common/docker/l1geth/ && docker build -t scroll_l2geth ./common/docker/l2geth/

clean: ## Empty out the bin folder
	@rm -rf build/bin
