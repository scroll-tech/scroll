#!/bin/bash

# generates go bindings from contracts, to paste into abi/bridge_abi.go
# compile artifacts in /contracts folder with `forge build`` first 

# Only run if it is ran from repository root.
if [[ ! -d "cmd" ]]
then
	echo "You need to run this script from the repository root."
	exit
fi

if [ -d "contracts" ]; then
  echo "Directory contracts exists"
else
  echo "Creating directory contracts"
  mkdir -p contracts
fi

abi_name=("IL1GatewayRouter" "IL2GatewayRouter" "IL1ScrollMessenger" "IL2ScrollMessenger" "IScrollChain" "L1MessageQueue")
pkg_name=("l1_gateway" "l2_gateway" "l1_messenger" "l2_messenger" "scrollchain" "l1_message_queue")
gen_name=("L1GatewayRouter" "L2GatewayRouter" "L1ScrollMessenger" "L2ScrollMessenger" "IScrollChain" "L1MessageQueue")

for i in "${!abi_name[@]}"; do
  mkdir -p tmp
  abi="tmp/${abi_name[$i]}.json"
  cat ../scroll-contracts/artifacts/src/${abi_name[$i]}.sol/${abi_name[$i]}.json | jq '.abi' > $abi
  pkg="${pkg_name[$i]}_abi"
  out="contracts/${pkg}/${gen_name[$i]}.go"
  echo "generating ${out} from ${abi}"
  mkdir -p contracts/$pkg
  abigen --abi=$abi --pkg=$pkg --out=$out
  awk '{sub("github.com/ethereum","github.com/scroll-tech")}1' $out > temp && mv temp $out
done

rm -rf tmp