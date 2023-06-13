#/bin/sh
set -uex

export L2_DEPLOYER_PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 

PORT=1234

# deploys a local instance of the contracts
anvil --port $PORT &
P1=$!

while ! lsof -i :$PORT
do
    echo "...waiting for anvil"
    sleep 1
done
echo "started anvil"

forge script ./foundry/DeployL2AdminContracts.s.sol:DeployL2AdminContracts --rpc-url http://localhost:1234 --legacy --broadcast -vvvv

./encode.sh

echo "deployment success"