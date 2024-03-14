docker ps -a --format '{{.Names}}' | grep -v 'scroll_test_container' | xargs -r docker rm -f
rm -Rf ./consensus/beacondata ./consensus/validatordata ./consensus/genesis.ssz
rm -Rf ./execution/geth
