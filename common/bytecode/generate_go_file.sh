#!/bin/bash

#L1/gateways
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L1/gateways/L1CustomERC20Gateway.json    --pkg gateways --out ./L1/gateways/L1CustomERC20Gateway.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L1/gateways/L1ERC20Gateway.json          --pkg gateways --out ./L1/gateways/L1ERC20Gateway.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L1/gateways/L1ERC721Gateway.json         --pkg gateways --out ./L1/gateways/L1ERC721Gateway.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L1/gateways/L1ERC1155Gateway.json        --pkg gateways --out ./L1/gateways/L1ERC1155Gateway.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L1/gateways/L1GatewayRouter.json         --pkg gateways --out ./L1/gateways/L1GatewayRouter.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L1/gateways/L1StandardERC20Gateway.json  --pkg gateways --out ./L1/gateways/L1StandardERC20Gateway.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L1/gateways/L1WETHGateway.json           --pkg gateways --out ./L1/gateways/L1WETHGateway.go

#L1/rollup
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L1/rollup/ZKRollup.json --pkg rollup --out ./L1/rollup/ZKRollup.go

#L1
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L1/L1ScrollMessenger.json --pkg l1 --out ./L1/L1ScrollMessenger.go

#L2/gateways
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L2/gateways/L2CustomERC20Gateway.json    --pkg gateways --out ./L2/gateways/L2CustomERC20Gateway.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L2/gateways/L2ERC20Gateway.json          --pkg gateways --out ./L2/gateways/L2ERC20Gateway.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L2/gateways/L2ERC721Gateway.json         --pkg gateways --out ./L2/gateways/L2ERC721Gateway.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L2/gateways/L2ERC1155Gateway.json        --pkg gateways --out ./L2/gateways/L2ERC1155Gateway.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L2/gateways/L2GatewayRouter.json         --pkg gateways --out ./L2/gateways/L2GatewayRouter.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L2/gateways/L2StandardERC20Gateway.json  --pkg gateways --out ./L2/gateways/L2StandardERC20Gateway.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L2/gateways/L2WETHGateway.json           --pkg gateways --out ./L2/gateways/L2WETHGateway.go

#L2/predeploys
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L2/predeploys/L2ToL1MessagePasser.json   --pkg predeploys --out ./L2/predeploys/L2ToL1MessagePasser.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L2/predeploys/WETH9.json                 --pkg predeploys --out ./L2/predeploys/WETH9.go
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L2/predeploys/Whitelist.json             --pkg predeploys --out ./L2/predeploys/Whitelist.go

#L2
go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./L2/L2ScrollMessenger.json --pkg l2 --out ./L2/L2ScrollMessenger.go
