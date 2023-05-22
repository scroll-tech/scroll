.PHONY: all erc20 greeter scroll clean

# TODO: After `feat/upgrade_abigen` branch merged, change to use develop branch again.
abigen="go run github.com/scroll-tech/go-ethereum/cmd/abigen@14a21cc835017ee2c17aaab52c4454b0d49ec7ba"

all: erc20 greeter scroll

erc20:
	abigen --sol erc20/ERC20Mock.sol 		--pkg erc20 --out erc20/ERC20Mock.go --contract ERC20Mock

greeter:
	abigen --sol ./greeter/Greeter.sol 	--pkg greeter --out ./greeter/Greeter.go --contract Greeter

scroll:
	#L1/gateways
	abigen --sol scroll/L1/gateways/L1ETHGateway.sol 				--pkg gateways --out scroll/L1/gateways/L1ETHGateway.go --contract L1ETHGateway
	#L1/rollup
	abigen --sol ./scroll/L1/rollup/L1MessageQueue.sol 				--pkg rollup --out ./scroll/L1/rollup/L1MessageQueue.go --contract L1MessageQueue
	abigen --sol ./scroll/L1/rollup/L2GasPriceOracle.sol 			--pkg rollup --out ./scroll/L1/rollup/L2GasPriceOracle.go --contract L2GasPriceOracle
	abigen --sol ./scroll/L1/rollup/ScrollChain.sol 				--pkg rollup --out ./scroll/L1/rollup/ScrollChain.go --contract ScrollChain
	#L1
	abigen --sol ./scroll/L1/L1ScrollMessenger.sol 					--pkg l1 --out ./scroll/L1/L1ScrollMessenger.go --contract L1ScrollMessenger
	#L2/gateways
	abigen --sol ./scroll/L2/gateways/L2ETHGateway.sol           	--pkg gateways --out ./scroll/L2/gateways/L2ETHGateway.go --contract L2ETHGateway
	#L2/predeploys
	abigen --sol ./scroll/L2/predeploys/L1BlockContainer.sol   		--pkg predeploys --out ./scroll/L2/predeploys/L1BlockContainer.go --contract L1BlockContainer
	abigen --sol ./scroll/L2/predeploys/L1GasPriceOracle.sol   		--pkg predeploys --out ./scroll/L2/predeploys/L1GasPriceOracle.go --contract L1GasPriceOracle
	abigen --sol ./scroll/L2/predeploys/L2MessageQueue.sol   		--pkg predeploys --out ./scroll/L2/predeploys/L2MessageQueue.go --contract L2MessageQueue
	abigen --sol ./scroll/L2/predeploys/L2TxFeeVault.sol   			--pkg predeploys --out ./scroll/L2/predeploys/L2TxFeeVault.go --contract L2TxFeeVault
	#L2
	abigen --sol ./scroll/L2/L2ScrollMessenger.sol 				--pkg l2 --out ./scroll/L2/L2ScrollMessenger.go --contract L2ScrollMessenger

clean:
	find ./ -type f -name "*.go" | xargs rm -r
