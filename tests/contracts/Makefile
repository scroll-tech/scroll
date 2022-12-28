.PHONY: abi

abi:
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./dao/GovernorMock.json  					--pkg dao 		--out ./dao/GovernorMock.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./erc20/ERC20Mock.json 					--pkg erc20 	--out ./erc20/ERC20Mock.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./greeter/Greeter.json 					--pkg greeter 	--out ./greeter/Greeter.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./nft/ERC721Mock.json 						--pkg nft 		--out ./nft/ERC721Mock.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./sushi/sushi.json 						--pkg sushi 	--out ./sushi/sushi.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./vote/VotesMock.json 						--pkg vote 		--out ./vote/VotesMock.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./uniswap/factory/UniswapV2Factory.json 	--pkg factory 	--out ./uniswap/factory/UniswapV2Factory.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./uniswap/multicall/Multicall.json 		--pkg multicall --out ./uniswap/multicall/Multicall.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./uniswap/router/UniswapV2Router02.json 	--pkg router 	--out ./uniswap/router/UniswapV2Router02.go
	go run github.com/scroll-tech/go-ethereum/cmd/abigen --combined-json ./uniswap/weth9/WETH9.json 				--pkg weth9 	--out ./uniswap/weth9/WETH9.go
