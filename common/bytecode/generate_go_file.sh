#!/bin/bash

#L1/gateways
abigen --combined-json ./L1/gateways/L1CustomERC20Gateway.json --pkg l1CustomERC20Gateway --out ./abi/L1/gateways/L1CustomERC20Gateway.go
abigen --combined-json ./L1/gateways/L1ERC20Gateway.json --pkg L1ERC20Gateway --out ./abi/L1/gateways/L1ERC20Gateway.go
abigen --combined-json ./L1/gateways/L1ERC721Gateway.json --pkg L1ERC721Gateway --out ./abi/L1/gateways/L1ERC721Gateway.go
abigen --combined-json ./L1/gateways/L1ERC1155Gateway.json --pkg L1ERC1155Gateway --out ./abi/L1/gateways/L1ERC1155Gateway.go
abigen --combined-json ./L1/gateways/L1GatewayRouter.json --pkg L1GatewayRouter --out ./abi/L1/gateways/L1GatewayRouter.go
abigen --combined-json ./L1/gateways/L1StandardERC20Gateway.json --pkg L1StandardERC20Gateway --out ./abi/L1/gateways/L1StandardERC20Gateway.go
abigen --combined-json ./L1/gateways/L1WETHGateway.json --pkg L1WETHGateway --out ./abi/L1/gateways/L1WETHGateway.go

#L1/rollup
abigen --combined-json ./L1/rollup/ZKRollup.json --pkg ZKRollup --out ./abi/L1/rollup/ZKRollup.go

#L1
abigen --combined-json ./L1/L1ScrollMessenger.json --pkg ZKRollup --out ./abi/L1/L1ScrollMessenger.go

#L2/gateways
abigen --combined-json ./L2/gateways/L2CustomERC20Gateway.json --pkg l2CustomERC20Gateway --out ./abi/L2/gateways/L2CustomERC20Gateway.go
abigen --combined-json ./L2/gateways/L2ERC20Gateway.json --pkg L2ERC20Gateway --out ./abi/L2/gateways/L2ERC20Gateway.go
abigen --combined-json ./L2/gateways/L2ERC721Gateway.json --pkg L2ERC721Gateway --out ./abi/L2/gateways/L2ERC721Gateway.go
abigen --combined-json ./L2/gateways/L2ERC1155Gateway.json --pkg L2ERC1155Gateway --out ./abi/L2/gateways/L2ERC1155Gateway.go
abigen --combined-json ./L2/gateways/L2GatewayRouter.json --pkg L2GatewayRouter --out ./abi/L2/gateways/L2GatewayRouter.go
abigen --combined-json ./L2/gateways/L2StandardERC20Gateway.json --pkg L2StandardERC20Gateway --out ./abi/L2/gateways/L2StandardERC20Gateway.go
abigen --combined-json ./L2/gateways/L2WETHGateway.json --pkg L2WETHGateway --out ./abi/L2/gateways/L2WETHGateway.go

#L2/predeploys
abigen --combined-json ./L2/predeploys/L2ToL1MessagePasser.json --pkg L2ToL1MessagePasser --out ./abi/L2/predeploys/L2ToL1MessagePasser.go
abigen --combined-json ./L2/predeploys/WETH9.json --pkg WETH9 --out ./abi/L2/predeploys/WETH9.go
abigen --combined-json ./L2/predeploys/Whitelist.json --pkg Whitelist --out ./abi/L2/predeploys/Whitelist.go

#L2
abigen --combined-json ./L2/L2ScrollMessenger.json --pkg L2ScrollMessenger --out ./abi/L2/L2ScrollMessenger.go
