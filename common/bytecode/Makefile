.PHONY: all erc20 greeter scroll

all: erc20 greeter scroll

erc20:
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./erc20/ERC20Mock.json --pkg erc20 --out ./erc20/ERC20Mock.go

greeter:
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./greeter/Greeter.json --pkg greeter --out ./greeter/Greeter.go

scroll:
	#L1/gateways
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L1/gateways/L1ERC20Gateway.json         --pkg gateways --out ./scroll/L1/gateways/L1ERC20Gateway.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L1/gateways/L1ERC721Gateway.json        --pkg gateways --out ./scroll/L1/gateways/L1ERC721Gateway.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L1/gateways/L1ERC1155Gateway.json       --pkg gateways --out ./scroll/L1/gateways/L1ERC1155Gateway.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L1/gateways/L1GatewayRouter.json        --pkg gateways --out ./scroll/L1/gateways/L1GatewayRouter.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L1/gateways/L1StandardERC20Gateway.json --pkg gateways --out ./scroll/L1/gateways/L1StandardERC20Gateway.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L1/gateways/L1CustomERC20Gateway.json   --pkg gateways --out ./scroll/L1/gateways/L1CustomERC20Gateway.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L1/gateways/L1WETHGateway.json          --pkg gateways --out ./scroll/L1/gateways/L1WETHGateway.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L1/gateways/L1ETHGateway.json			--pkg gateways --out ./scroll/L1/gateways/L1ETHGateway.go
	#L1/rollup
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L1/rollup/L1MessageQueue.json 			--pkg rollup --out ./scroll/L1/rollup/L1MessageQueue.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L1/rollup/L2GasPriceOracle.json 		--pkg rollup --out ./scroll/L1/rollup/L2GasPriceOracle.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L1/rollup/ScrollChain.json 				--pkg rollup --out ./scroll/L1/rollup/ScrollChain.go
	#L1
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L1/L1ScrollMessenger.json 				--pkg l1 --out ./scroll/L1/L1ScrollMessenger.go
	#L2/gateways
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/gateways/L2CustomERC20Gateway.json    	--pkg gateways --out ./scroll/L2/gateways/L2CustomERC20Gateway.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/gateways/L2ERC20Gateway.json          	--pkg gateways --out ./scroll/L2/gateways/L2ERC20Gateway.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/gateways/L2ERC721Gateway.json         	--pkg gateways --out ./scroll/L2/gateways/L2ERC721Gateway.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/gateways/L2ERC1155Gateway.json        	--pkg gateways --out ./scroll/L2/gateways/L2ERC1155Gateway.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/gateways/L2GatewayRouter.json         	--pkg gateways --out ./scroll/L2/gateways/L2GatewayRouter.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/gateways/L2StandardERC20Gateway.json  	--pkg gateways --out ./scroll/L2/gateways/L2StandardERC20Gateway.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/gateways/L2WETHGateway.json           	--pkg gateways --out ./scroll/L2/gateways/L2WETHGateway.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/gateways/L2ETHGateway.json           	--pkg gateways --out ./scroll/L2/gateways/L2ETHGateway.go
	#L2/predeploys
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/predeploys/L1BlockContainer.json   	--pkg predeploys --out ./scroll/L2/predeploys/L1BlockContainer.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/predeploys/L1GasPriceOracle.json   	--pkg predeploys --out ./scroll/L2/predeploys/L1GasPriceOracle.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/predeploys/L2MessageQueue.json   	--pkg predeploys --out ./scroll/L2/predeploys/L2MessageQueue.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/predeploys/L2TxFeeVault.json   		--pkg predeploys --out ./scroll/L2/predeploys/L2TxFeeVault.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/predeploys/WETH9.json                --pkg predeploys --out ./scroll/L2/predeploys/WETH9.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/predeploys/Whitelist.json            --pkg predeploys --out ./scroll/L2/predeploys/Whitelist.go
	#L2
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/L2/L2ScrollMessenger.json --pkg l2 --out ./scroll/L2/L2ScrollMessenger.go
	#External
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/external/External.json --pkg external --out ./scroll/external/External.go
	#Libraries
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/libraries/ScrollStandardERC20.json 			--pkg libraries --out ./scroll/libraries/ScrollStandardERC20.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./scroll/libraries/ScrollStandardERC20Factory.json 	--pkg libraries --out ./scroll/libraries/ScrollStandardERC20Factory.go
