.PHONY: abi lint

abi:
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./abi/dao/GovernorMock.json  					--pkg dao 		--out ./abi/dao/GovernorMock.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./abi/erc20/ERC20Mock.json 					--pkg erc20 	--out ./abi/erc20/ERC20Mock.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./abi/greeter/Greeter.json 					--pkg greeter 	--out ./abi/greeter/Greeter.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./abi/nft/ERC721Mock.json 						--pkg nft 		--out ./abi/nft/ERC721Mock.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./abi/sushi/sushi.json 						--pkg sushi 	--out ./abi/sushi/sushi.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./abi/vote/VotesMock.json 						--pkg vote 		--out ./abi/vote/VotesMock.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./abi/uniswap/factory/UniswapV2Factory.json 	--pkg factory 	--out ./abi/uniswap/factory/UniswapV2Factory.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./abi/uniswap/multicall/Multicall.json 		--pkg multicall --out ./abi/uniswap/multicall/Multicall.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./abi/uniswap/router/UniswapV2Router02.json 	--pkg router 	--out ./abi/uniswap/router/UniswapV2Router02.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./abi/uniswap/weth9/WETH9.json 				--pkg weth9 	--out ./abi/uniswap/weth9/WETH9.go

