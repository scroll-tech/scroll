#!/bin/bash

# Only run if it is ran from repository root.
if [[ ! -d "cmd" ]]
then
	echo "You need to run this script from the repository root."
	exit
fi

if [ -d "bridge/contracts" ]; then
  echo "Directory bridge/contracts exists"
else
  echo "Creating directory bridge/contracts"
  mkdir -p bridge/contracts
fi

abi_name=("IL1GatewayRouter" "IL2GatewayRouter" "IL1ScrollMessenger" "IL2ScrollMessenger" "ZKRollup")
pkg_name=("l1_gateway" "l2_gateway" "l1_messenger" "l2_messenger" "rollup")
gen_name=("L1GatewayRouter" "L2GatewayRouter" "L1ScrollMessenger" "L2ScrollMessenger" "ZKRollup")

for i in "${!abi_name[@]}"; do
  abi="bridge/abi/${abi_name[$i]}.json"
  pkg="${pkg_name[$i]}"
  out="bridge/contracts/${pkg}/${gen_name[$i]}.go"
  echo "generating ${out} from ${abi}"
  mkdir -p bridge/contracts/$pkg
  abigen --abi=$abi --pkg=$pkg --out=$out
  awk '{sub("github.com/ethereum","github.com/scroll-tech")}1' $out > temp && mv temp $out
done