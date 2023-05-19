.PHONY: all erc20 greeter scroll clean

all: erc20 greeter scroll

erc20:
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol erc20/ERC20Mock.sol 		--pkg erc20 --out erc20/ERC20Mock.go

greeter:
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol ./greeter/Greeter.sol 	--pkg greeter --out ./greeter/Greeter.go

scroll:
	#L1/gateways
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol scroll/L1/ethgateway/L1ETHGateway.sol 				--pkg ethgateway --out scroll/L1/ethgateway/L1ETHGateway.go
	#L1/rollup
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol ./scroll/L1/messagequeue/L1MessageQueue.sol 			--pkg messagequeue --out ./scroll/L1/messagequeue/L1MessageQueue.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol ./scroll/L1/priceoracle/L2GasPriceOracle.sol 			--pkg priceoracle --out ./scroll/L1/priceoracle/L2GasPriceOracle.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol ./scroll/L1/scrollchain/ScrollChain.sol 				--pkg scrollchain --out ./scroll/L1/scrollchain/ScrollChain.go
	#L1
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol ./scroll/L1/L1ScrollMessenger.sol 				--pkg l1 --out ./scroll/L1/L1ScrollMessenger.go
	#L2/gateways
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol ./scroll/L2/ethgateway/L2ETHGateway.sol           	--pkg ethgateway --out ./scroll/L2/ethgateway/L2ETHGateway.go
	#L2/predeploys
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol ./scroll/L2/container/L1BlockContainer.sol   	--pkg container --out ./scroll/L2/container/L1BlockContainer.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol ./scroll/L2/priceoracle/L1GasPriceOracle.sol   	--pkg priceoracle --out ./scroll/L2/priceoracle/L1GasPriceOracle.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol ./scroll/L2/messagequeue/L2MessageQueue.sol   		--pkg messagequeue --out ./scroll/L2/messagequeue/L2MessageQueue.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol ./scroll/L2/feevault/L2TxFeeVault.sol   		--pkg feevault --out ./scroll/L2/feevault/L2TxFeeVault.go
	#L2
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --sol ./scroll/L2/L2ScrollMessenger.sol 				--pkg l2 --out ./scroll/L2/L2ScrollMessenger.go

clean:
	find ./ -type f -name "*.go" | xargs rm -r
