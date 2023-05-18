.PHONY: all erc20 greeter scroll clean

all: erc20 greeter scroll

erc20:
	abigen --sol erc20/ERC20Mock.sol --pkg erc20 --out erc20/ERC20Mock.go

greeter:
	abigen --combined-json ./greeter/Greeter.json --pkg greeter --out ./greeter/Greeter.go

scroll:
	#L1/gateways
	abigen --sol scroll/L1/gateways/L1ETHGateway.sol --pkg gateways --out scroll/L1/gateways/L1ETHGateway.go
	#L1/rollup
	abigen --sol ./scroll/L1/rollup/L1MessageQueue.sol 			--pkg rollup --out ./scroll/L1/rollup/L1MessageQueue.go
	abigen --sol ./scroll/L1/rollup/L2GasPriceOracle.sol 		--pkg rollup --out ./scroll/L1/rollup/L2GasPriceOracle.go
	abigen --sol ./scroll/L1/rollup/ScrollChain.sol 				--pkg rollup --out ./scroll/L1/rollup/ScrollChain.go
	#L1
	abigen --sol ./scroll/L1/L1ScrollMessenger.sol 				--pkg l1 --out ./scroll/L1/L1ScrollMessenger.go
	#L2/gateways
	abigen --sol ./scroll/L2/gateways/L2ETHGateway.sol           	--pkg gateways --out ./scroll/L2/gateways/L2ETHGateway.go
	#L2/predeploys
	abigen --sol ./scroll/L2/predeploys/L1BlockContainer.sol   	--pkg predeploys --out ./scroll/L2/predeploys/L1BlockContainer.go
	abigen --sol ./scroll/L2/predeploys/L1GasPriceOracle.sol   	--pkg predeploys --out ./scroll/L2/predeploys/L1GasPriceOracle.go
	abigen --sol ./scroll/L2/predeploys/L2MessageQueue.sol   	--pkg predeploys --out ./scroll/L2/predeploys/L2MessageQueue.go
	abigen --sol ./scroll/L2/predeploys/L2TxFeeVault.sol   		--pkg predeploys --out ./scroll/L2/predeploys/L2TxFeeVault.go
	abigen --sol ./scroll/L2/predeploys/WETH9.sol                --pkg predeploys --out ./scroll/L2/predeploys/WETH9.go
	abigen --sol ./scroll/L2/predeploys/Whitelist.sol            --pkg predeploys --out ./scroll/L2/predeploys/Whitelist.go
	#L2
	abigen --sol ./scroll/L2/L2ScrollMessenger.sol --pkg l2 --out ./scroll/L2/L2ScrollMessenger.go

clean:
	find ./ -type f -name "*.go" | xargs rm -r
