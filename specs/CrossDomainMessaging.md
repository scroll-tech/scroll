# Cross Domain Messaging

Scroll has an arbitrary message passing bridge that enables the token transfers and allows dapps to communicate between layer 1 and layer 2. This means that dapps on layer 1 can trigger contract functions on layer 2, and vice versa. Next, we will explain how the messages are relayed between layer 1 and layer 2.

## Send Message from L1 to L2

<figure>
<img src="assets/L1-to-L2.png" alt="L1 to L2 workflow" style="width:80%">
<figcaption align = "center"><b>Figure 1. L1 to L2 message relay workflow</b></figcaption>
</figure>

On the L1, there are two approaches to send a message to L2: sending arbitrary messages via `L1ScrollMessenger` and sending enforced transactions via `EnforcedTxGateway`.
The difference between these two methods is that the sender of arbitrary message transactions is `L1ScrollMessenger` while the sender of enforced transactions is an EOA account.
In addition to the `L1ScrollMessenger`, we provide several standard token gateways to make users easier to deposit ETH and other standard tokens including ERC-20, ERC-677, ERC-721, and ERC-1155.
Basically gateways encode deposits to a message and send to `L1ScrollMessenger`.
You can find more details about token gateways in the [Deposit Tokens](Deposit.md).

As depicted in the Figure 1, both arbitrary messages and enforced transactions are appended to message queue stored in the `L1MessageQueue`.
`L1MessageQueue` provides two APIs `appendCrossDomainMessage` and `appendEnforcedTransaction`, which can be only called by `L1ScrollMessenger` and `EnforcedTxGateway` correspondingly.

```solidity
function appendCrossDomainMessage(address target, uint256 gasLimit, bytes calldata data) external;
function appendEnforcedTransaction(address sender, address target, uint256 value, uint256 gasLimit, bytes calldata data) external;
```

Both functions then construct a L1-initiated transaction with a new transaction type `L1MessageTx` introduced in the Scroll chain and computes the transaction hash (see more details in the [L1 Message Transaction](#l1-message-transaction)).
`L1MessageQueue` appends the transaction hash to the message queue, and emits the event `QueueTransaction(sender, target, value, queueIndex, gasLimit, calldata)`.
The difference between `appendCrossDomainMessage` and `appendEnforcedTransaction` when constructing the L1 message transactions is:
- `appendCrossDomainMessage` uses the [aliased](#address-alias) address of `msg.sender`, which must be the address of `L1ScrollMessenger`, as the transaction sender.
- `appendEnforcedTransaction` uses `sender` from the argument as the transaction sender. This allows users to enforce a withdrawal or transfer of ETH from their L2 accounts via L1 transactions.

After the transaction is successfully executed on the L1, the watcher in the Scroll sequencer that monitors the `L1MessageQueue` contract will detect the new `QueueTransaction` events from L1 blocks.
The sequencer then generates a new `L1MessageTx` transaction per event and adds to the L1 transaction queue in the sequencer.
Later when it's the block time, the sequencer will include the transactions from both L1 transaction queue and L2 mempool to construct a new L2 block.
Note that the L1 message transactions must be included sequentially based on the L1 message queue order in the `L1MessageQueue` contract.
`L1MessageTx` transactions always come first in the L2 blocks followed by L2 transactions.
Currently, we limit the number of `L1MessageTx` transactions in a L2 block to `MAX_NUM_L1_MESSAGES` (TBD, likely 20).

Next, we will expand the details about how to send arbitrary messages via `L1ScrollMessenger` and send enforced transaction via `EnforcedTxGateway`.

### Send Arbitrary Messages

The `L1ScrollMessenger` contract provides two functions to send arbitrary messages.
The only difference is that one allows users to specify a refund address different from the sender address.

```solidity
function sendMessage(
    address target,
    uint256 value,
    bytes memory message,
    uint256 gasLimit
) external payable;

function sendMessage(
    address target,
    uint256 value,
    bytes calldata message,
    uint256 gasLimit,
    address refundAddress
) external payable;
```

In the `sendMessage` function, it encodes the arguments to cross-domain calldata, where the message nonce is the next queue index of the L1 message queue. This calldata is going to be used in the `L1MessageTx` transaction.

```solidity
abi.encodeWithSignature(
    "relayMessage(address,address,uint256,uint256,bytes)",
    _sender,
    _target,
    _value,
    _messageNonce,
    _message
)
```

The function calculates the message relay fee (see calculation method in the [Cross Domain Message Relay Fee](#cross-domain-message-relay-fee)) and deposit the fee to a `feeVault` account.
If `value` in the `sendMessage` is not zero, `L1ScrollMessenger` will freeze `value` amount of ETH in the contract and refund the excess amount to the designated `refundAddress` or the transaction sender otherwise.
If the amount of ETH in the message cannot cover the fee and deposit amount, the transaction will fail.
`L1ScrollMessenger` then appends the cross-domain message to `L1MessageQueue` via `appendCrossDomainMessage` method.

Todo: describe the replay message in case of failure due to too little gas limit.

### Send Enforced Transaction

The `EnforcedTxGateway` contract provides two functions to send enforced transactions listed below.

```solidity
function sendTransaction(
    address target,
    uint256 value,
    uint256 gasLimit,
    bytes calldata data
) external payable;

function sendTransaction(
    address sender,
    address target,
    uint256 value,
    uint256 gasLimit,
    bytes calldata data,
    bytes memory signature,
    address refundAddress
) external payable;
```

In the first function, the sender of the generated L1 message transaction is the transaction sender.
On the other hand, the second function uses the passed `sender` address as the sender of the L1 message transaction.
This allows a third party to send an enforced transaction on behalf of the user and pay the relay fee.
Note that the second function requires to provide a signature of the generated L1 message transaction that can recover the same address as `sender`.
Both `sendTransaction` functions enforce the sender must be an EOA account.

Similar to arbitrary message relaying, `sendTransaction` estimates the message relay fee and deduct the fee to a `feeVault` account.
But differently, the `value` passed to the function indicates the amount of ETH to transfer from the L2 account.
Hence, the `msg.value` only needs to cover the message relay fee.
If the amount of ETH in the message cannot cover the fee, the transaction will fail.
The excess fee is refunded to the transaction sender in the first function and to the `refundAddress` in the second function.
At last, `EnforcedTxGateway` calls `L1MessageQueue.appendEnforcedTransaction` to append the transaction to the message queue.

### L1 Message Transaction

We introduce a new transaction type `L1MessageTx` in the Scroll chain for L1 initiated transactions.
The payload of `L1MessageTx` is defined below.
The `L1MessageTx` transaction type is `0x7E` and the encoding of `L1MessageTx` transactions is `0x7E || rlp([queue_index, gas_limit, target, value, data, sender])`.
Note that this transaction doesn't contain the signature because the transaction is constructed in the L1 contract which doesn't have access to the account secret key.
```go
type L1MessageTx struct {
	QueueIndex uint64          // The queue index of the message queue in L1 contract
	Gas        uint64          // gas limit
	To         *common.Address // can not be nil, we do not allow contract creation from L1
	Value      *big.Int
	Data       []byte
	Sender     common.Address
}
```

### Cross Domain Message Relay Fee

The contract `L2GasPriceOracle` deployed on the L1 computes the relay fee given the gas limit.
This contract stores the `l2BaseFee` in the contract, which is updated by a dedicated relayer run by Scroll currently.
The relay fee of L1-to-L2 messages is `gasLimit * l2BaseFee`.

### Address Alias


## Send Message from L2 to L1

<figure>
<img src="assets/L2-to-L1.png" alt="L2 to L1 workflow" style="width:80%">
<figcaption align = "center"><b>Figure 2. L2 to L1 message relay workflow</b></figcaption>
</figure>

Similar to sending message from L1 to L2, you should call `L2ScrollMessenger.sendMessage` first in layer 2. The `L2ScrollMessenger` contract will emit a `SentMessage` event, which is monitored by the Relayer. TBA

Different from L1 to L2 message relay, the Relayer won't relay the message until the batch that contains the message is finalized in the `ScrollChain` contract on the Layer 1. The Relayer will generate the withdraw and submit the proof to `ZKRollup` contract in layer 1 again. Finally, anyone can call `L1ScrollMessenger.relayMessageWithProof` with correct proof to execute the message in layer 1.

Currently, for the safety reason, we only allow privileged contracts to send cross domain messages. And only privileged accounts can call `L2ScrollMessenger.relayMessage`.

