#/bin/sh
set -uex

L2_COUNCIL_SAFE_ADDR=0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
L2_COUNCIL_TIMELOCK_ADDR=0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9
L2_SCROLL_SAFE_ADDR=0xa513E6E4b8f2a923D98304ec87F64353C4D5C853
L2_SCROLL_TIMELOCK_ADDR=0x8A791620dd6260079BF849Dc5567aDC3F2FdC318
L2_FORWARDER_ADDR=0xA51c1fc2f0D1a1b8494Ed1FE312d7C3a78Ed91C0
L2_TARGET_ADDR=0x0DCd1Bf9A1b36cE34237eEaFef220932846BCD82

# 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
L2_DEPLOYER_PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
ZERO_BYTES=0x0000000000000000000000000000000000000000

# sign tx hash for timelock schedule call 
ADMIN_CALLDATA=$(cast calldata "err()")
FORWARDER_CALLDATA=$(cast calldata "forward(address,bytes)" $L2_FORWARDER_ADDR $ADMIN_CALLDATA)
TIMELOCK_SCHEDULE_CALLDATA=$(cast calldata "schedule(address,uint256,bytes,bytes32,bytes32,uint256)" $L2_FORWARDER_ADDR 0 $FORWARDER_CALLDATA 0x0 0x0 0x0)

SAFE_TX_HASH=$(cast call -r http://localhost:1234 $L2_SCROLL_SAFE_ADDR "getTransactionHash(address,uint256,bytes,uint8,uint256,uint256,uint256,address,address,uint256)" \
$L2_SCROLL_TIMELOCK_ADDR 0 $TIMELOCK_SCHEDULE_CALLDATA 0 0 0 0 $ZERO_BYTES $ZERO_BYTES 0)

SAFE_SIG=$(cast wallet sign --private-key $L2_DEPLOYER_PRIVATE_KEY $SAFE_TX_HASH | awk '{print $2}')

# send safe tx to schedule the call
cast send -c 31337 --legacy --private-key $L2_DEPLOYER_PRIVATE_KEY -r http://localhost:1234 --gas-limit 1000000  $L2_SCROLL_SAFE_ADDR "execTransaction(address,uint256,bytes,uint8,uint256,uint256,uint256,address,address,bytes)" \
$L2_SCROLL_TIMELOCK_ADDR 0 $TIMELOCK_SCHEDULE_CALLDATA 0 0 0 0 $ZERO_BYTES $ZERO_BYTES $SAFE_SIG

# function encodeTransactionData(
#     address to,
#     uint256 value,
#     bytes calldata data,
#     Enum.Operation operation,
#     uint256 safeTxGas,
#     uint256 baseGas,
#     uint256 gasPrice,
#     address gasToken,
#     address refundReceiver,
#     uint256 _nonce

# function execTransaction(
#     address to,
#     uint256 value,
#     bytes calldata data,
#     Enum.Operation operation,
#     uint256 safeTxGas,
#     uint256 baseGas,
#     uint256 gasPrice,
#     address gasToken,
#     address payable refundReceiver,
#     bytes memory signatures

exit 0



















# /////////////// 2nd tx ///////////////

# sign tx hash for execute call
TIMELOCK_EXECUTE_CALLDATA=$(cast calldata "execute(address,uint256,bytes,bytes32,bytes32)" $L2_FORWARDER_ADDR 0 $FORWARDER_CALLDATA 0x0 0x0)
SAFE_TX_HASH_=$(cast call -r http://localhost:1234 $L2_SCROLL_SAFE_ADDR "getTransactionHash(address,uint256,bytes,uint8,uint256,uint256,uint256,address,address,uint256)" \
$L2_SCROLL_TIMELOCK_ADDR 0 $TIMELOCK_SCHEDULE_CALLDATA 0 0 0 0 $ZERO_BYTES $ZERO_BYTES 0)
SAFE_SIG=$(cast wallet sign --private-key $L2_DEPLOYER_PRIVATE_KEY $SAFE_TX_HASH | awk '{print $2}')

# send safe tx to execute the call
cast send -c 31337 --legacy --private-key $L2_DEPLOYER_PRIVATE_KEY -r http://localhost:1234 --gas-limit 1000000  $L2_SCROLL_SAFE_ADDR "execTransaction(address,uint256,bytes,uint8,uint256,uint256,uint256,address,address,bytes)" \
$L2_SCROLL_TIMELOCK_ADDR 0 $TIMELOCK_EXECUTE_CALLDATA 0 0 0 0 $ZERO_BYTES $ZERO_BYTES $SAFE_SIG

echo "DONE"