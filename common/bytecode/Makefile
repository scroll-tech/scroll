.PHONY: all erc20 greeter scroll clean

all: erc20 greeter scroll

erc20:
	abigen --combined-json ./erc20/ERC20Mock.json --pkg erc20 --out ./erc20/ERC20Mock.go

greeter:
	abigen --combined-json ./greeter/Greeter.json --pkg greeter --out ./greeter/Greeter.go

scroll:
	#L1/gateways
	abigen --combined-json ./scroll/L1/gateways/L1ETHGateway.json			--pkg gateways --out ./scroll/L1/gateways/L1ETHGateway.go
	#L1/rollup
	abigen --combined-json ./scroll/L1/rollup/L1MessageQueue.json 			--pkg rollup --out ./scroll/L1/rollup/L1MessageQueue.go
	abigen --combined-json ./scroll/L1/rollup/L2GasPriceOracle.json 		--pkg rollup --out ./scroll/L1/rollup/L2GasPriceOracle.go
	abigen --combined-json ./scroll/L1/rollup/ScrollChain.json 				--pkg rollup --out ./scroll/L1/rollup/ScrollChain.go
	#L1
	abigen --combined-json ./scroll/L1/L1ScrollMessenger.json 				--pkg l1 --out ./scroll/L1/L1ScrollMessenger.go
	#L2/gateways
	abigen --combined-json ./scroll/L2/gateways/L2ETHGateway.json           	--pkg gateways --out ./scroll/L2/gateways/L2ETHGateway.go
	#L2/predeploys
	abigen --combined-json ./scroll/L2/predeploys/L1BlockContainer.json   	--pkg predeploys --out ./scroll/L2/predeploys/L1BlockContainer.go
	abigen --combined-json ./scroll/L2/predeploys/L1GasPriceOracle.json   	--pkg predeploys --out ./scroll/L2/predeploys/L1GasPriceOracle.go
	abigen --combined-json ./scroll/L2/predeploys/L2MessageQueue.json   	--pkg predeploys --out ./scroll/L2/predeploys/L2MessageQueue.go
	abigen --combined-json ./scroll/L2/predeploys/L2TxFeeVault.json   		--pkg predeploys --out ./scroll/L2/predeploys/L2TxFeeVault.go
	abigen --combined-json ./scroll/L2/predeploys/WETH9.json                --pkg predeploys --out ./scroll/L2/predeploys/WETH9.go
	abigen --combined-json ./scroll/L2/predeploys/Whitelist.json            --pkg predeploys --out ./scroll/L2/predeploys/Whitelist.go
	#L2
	abigen --combined-json ./scroll/L2/L2ScrollMessenger.json --pkg l2 --out ./scroll/L2/L2ScrollMessenger.go
	#External
	abigen --combined-json ./scroll/external/External.json --pkg external --out ./scroll/external/External.go

clean:
	find ./ -type f -name "*.go" | xargs rm -r
