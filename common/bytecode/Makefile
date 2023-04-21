.PHONY: all erc20 greeter

all: erc20 greeter

erc20:
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./erc20/ERC20Mock.json --pkg erc20 --out ./erc20/ERC20Mock.go

greeter:
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./greeter/Greeter.json --pkg greeter --out ./greeter/Greeter.go
