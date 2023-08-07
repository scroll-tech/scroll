.PHONY: all erc20 greeter scroll

all: erc20 greeter scroll

erc20:
	abigen --combined-json ./erc20/ERC20Mock.json --pkg erc20 --out ./erc20/ERC20Mock.go

greeter:
	abigen --combined-json ./greeter/Greeter.json --pkg greeter --out ./greeter/Greeter.go

scroll:
	# L1
	./build_abi.sh L1
	# L1/gateway
	./build_abi.sh L1/gateway
	# L1/rollup
	./build_abi.sh L1/rollup
	# L2
	./build_abi.sh L2
	# L2/gateway
	./build_abi.sh L2/gateway
	# L2/predeploys
	./build_abi.sh L2/predeploys
