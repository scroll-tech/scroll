.PHONY: all erc20 greeter scroll scroll_sol clean

all: erc20 greeter scroll

erc20:
	go run github.com/scroll-tech/go-ethereum/cmd/abigen@develop --sol erc20/ERC20Mock.sol 		--pkg erc20 --out erc20/ERC20Mock.go --contract ERC20Mock

greeter:
	go run github.com/scroll-tech/go-ethereum/cmd/abigen@develop --sol ./greeter/Greeter.sol 	--pkg greeter --out ./greeter/Greeter.go --contract Greeter

# TODO: This cmd can just used in local env, and it's not suitable used in automatic process. Because bridge's update can't keep consistent with contracts' update.
scroll_sol:
	cd ../../contracts && forge install
	forge flatten --root ../../contracts  ../../contracts/src/L1/L1ScrollMessenger.sol 					| sed '/SPDX-License-Identifier/d' > ./scroll/L1/L1ScrollMessenger.sol
	#L1/gateways
	forge flatten --root ../../contracts ../../contracts/src/L1/gateways/L1ETHGateway.sol 				| sed '/SPDX-License-Identifier/d' > ./scroll/L1/gateways/L1ETHGateway.sol
	#L1/rollup
	forge flatten --root ../../contracts ../../contracts/src/L1/rollup/L1MessageQueue.sol 				| sed '/SPDX-License-Identifier/d' > ./scroll/L1/rollup/L1MessageQueue.sol
	forge flatten --root ../../contracts ../../contracts/src/L1/rollup/L2GasPriceOracle.sol 			| sed '/SPDX-License-Identifier/d' > ./scroll/L1/rollup/L2GasPriceOracle.sol
	forge flatten --root ../../contracts ../../contracts/src/L1/rollup/ScrollChain.sol 					| sed '/SPDX-License-Identifier/d' > ./scroll/L1/rollup/ScrollChain.sol
	#L1
	forge flatten --root ../../contracts ../../contracts/src/L1/L1ScrollMessenger.sol 					| sed '/SPDX-License-Identifier/d' > ./scroll/L1/L1ScrollMessenger.sol
	#L2/gateways
	forge flatten --root ../../contracts ../../contracts/src/L2/gateways/L2ETHGateway.sol           	| sed '/SPDX-License-Identifier/d' > ./scroll/L2/gateways/L2ETHGateway.sol
	#L2/predeploys
	forge flatten --root ../../contracts ../../contracts/src/L2/predeploys/L1BlockContainer.sol   		| sed '/SPDX-License-Identifier/d' > ./scroll/L2/predeploys/L1BlockContainer.sol
	forge flatten --root ../../contracts ../../contracts/src/L2/predeploys/L1GasPriceOracle.sol   		| sed '/SPDX-License-Identifier/d' > ./scroll/L2/predeploys/L1GasPriceOracle.sol
	forge flatten --root ../../contracts ../../contracts/src/L2/predeploys/L2MessageQueue.sol   		| sed '/SPDX-License-Identifier/d' > ./scroll/L2/predeploys/L2MessageQueue.sol
	forge flatten --root ../../contracts ../../contracts/src/L2/predeploys/L2TxFeeVault.sol   			| sed '/SPDX-License-Identifier/d' > ./scroll/L2/predeploys/L2TxFeeVault.sol
	#L2
	forge flatten --root ../../contracts ../../contracts/src/L2/L2ScrollMessenger.sol 					| sed '/SPDX-License-Identifier/d' > ./scroll/L2/L2ScrollMessenger.sol

scroll:
	#L1/gateways
	go run github.com/scroll-tech/go-ethereum/cmd/abigen@develop --sol scroll/L1/gateways/L1ETHGateway.sol 				--pkg gateways --out ./scroll/L1/gateways/L1ETHGateway.go --contract L1ETHGateway
	#L1/rollup
	go run github.com/scroll-tech/go-ethereum/cmd/abigen@develop --sol ./scroll/L1/rollup/L1MessageQueue.sol 			--pkg rollup --out ./scroll/L1/rollup/L1MessageQueue.go --contract L1MessageQueue
	go run github.com/scroll-tech/go-ethereum/cmd/abigen@develop --sol ./scroll/L1/rollup/L2GasPriceOracle.sol 			--pkg rollup --out ./scroll/L1/rollup/L2GasPriceOracle.go --contract L2GasPriceOracle
	go run github.com/scroll-tech/go-ethereum/cmd/abigen@develop --sol ./scroll/L1/rollup/ScrollChain.sol 				--pkg rollup --out ./scroll/L1/rollup/ScrollChain.go --contract ScrollChain
	#L1
	go run github.com/scroll-tech/go-ethereum/cmd/abigen@develop --sol ./scroll/L1/L1ScrollMessenger.sol 				--pkg l1 --out ./scroll/L1/L1ScrollMessenger.go --contract L1ScrollMessenger
	#L2/gateways
	go run github.com/scroll-tech/go-ethereum/cmd/abigen@develop --sol ./scroll/L2/gateways/L2ETHGateway.sol           	--pkg gateways --out ./scroll/L2/gateways/L2ETHGateway.go --contract L2ETHGateway
	#L2/predeploys
	go run github.com/scroll-tech/go-ethereum/cmd/abigen@develop --sol ./scroll/L2/predeploys/L1BlockContainer.sol   	--pkg predeploys --out ./scroll/L2/predeploys/L1BlockContainer.go --contract L1BlockContainer
	go run github.com/scroll-tech/go-ethereum/cmd/abigen@develop --sol ./scroll/L2/predeploys/L1GasPriceOracle.sol   	--pkg predeploys --out ./scroll/L2/predeploys/L1GasPriceOracle.go --contract L1GasPriceOracle
	go run github.com/scroll-tech/go-ethereum/cmd/abigen@develop --sol ./scroll/L2/predeploys/L2MessageQueue.sol   		--pkg predeploys --out ./scroll/L2/predeploys/L2MessageQueue.go --contract L2MessageQueue
	go run github.com/scroll-tech/go-ethereum/cmd/abigen@develop --sol ./scroll/L2/predeploys/L2TxFeeVault.sol   		--pkg predeploys --out ./scroll/L2/predeploys/L2TxFeeVault.go --contract L2TxFeeVault
	#L2
	go run github.com/scroll-tech/go-ethereum/cmd/abigen@develop --sol ./scroll/L2/L2ScrollMessenger.sol 				--pkg l2 --out ./scroll/L2/L2ScrollMessenger.go --contract L2ScrollMessenger

clean:
	find ./ -type f -name "*.go" | xargs rm -r
	find scroll/ -type f -name "*.sol" | xargs rm -f