.PHONY: all dao erc20 greeter nft sushi uniswapv2 vote weth9

all:
	make dao erc20 greeter nft sushi uniswapv2 vote weth9

dao:
	abigen --combined-json ./dao/GovernorMock.json --pkg dao --out ./dao/GovernorMock.go

erc20:
	abigen --combined-json ./erc20/ERC20Mock.json --pkg erc20 --out ./erc20/ERC20Mock.go

greeter:
	abigen --combined-json ./greeter/Greeter.json --pkg greeter --out ./greeter/Greeter.go
	
nft:
	abigen --combined-json ./nft/ERC721Mock.json --pkg nft --out ./nft/ERC721Mock.go
	
sushi:
	abigen --combined-json ./sushi/sushi.json --pkg sushi --out ./sushi/sushi.go
	
vote:
	abigen --combined-json ./vote/VotesMock.json --pkg vote --out ./vote/VotesMock.go
	
uniswapv2:
	abigen --combined-json ./uniswap/factory/UniswapV2Factory.json 	--pkg factory 	--out ./uniswap/factory/UniswapV2Factory.go
	abigen --combined-json ./uniswap/multicall/Multicall.json 		--pkg multicall --out ./uniswap/multicall/Multicall.go
	abigen --combined-json ./uniswap/router/UniswapV2Router02.json 	--pkg router 	--out ./uniswap/router/UniswapV2Router02.go

weth9:
	abigen --combined-json ./uniswap/weth9/WETH9.json --pkg weth9 --out ./uniswap/weth9/WETH9.go