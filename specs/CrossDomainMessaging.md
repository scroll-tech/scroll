# Cross Domain Messaging

Like other layer 2 protocol, Scroll builds an arbitrary message passing bridge that allows the token transfer and dapps to communicate between layer 1 and layer 2. In such way, dapps on layer 1 can trigger contract functions on layer 2, and vice versa.

In essence, the protocol implements two core contracts `L1ScrollMessenger` and `L2ScrollMessenger` to enable the cross domain messaging. The entry point to send cross domain messages is to call the `sendMessage` function in both contracts:

```solidity
function sendMessage(
    address _to,
    uint256 _value,
    bytes memory _message,
    uint256 _gasLimit
) external payable;

function sendMessage(
    address _to,
    uint256 _value,
    bytes calldata _message,
    uint256 _gasLimit,
    address _refundAddress
) external payable;
```

Though both L1 and L2 messenger contract support the same interface, the mechanism behind the message relay from L1 to L2 and from L2 to L1 works differently. The subsequent sections will describe the detailed workflow.

## Send Message from L1 to L2

![L1 to L2 workflow](assets/L1-to-L2.png)

On the L1, there are three entry points for users and dapps to send a message to L2.

- We provide a few standard gateway contracts to deposit Ether and several standard tokens such as ERC20, ERC-721, and ERC-1155. You can check out [Deposit](Deposit.md) to find out more details about these gateways. The gateways will encode the deposit to a message and send to `L1ScrollMessenger.sendMessage`.
- Users can directly use `L1ScrollMessenger.sendMessage` to send arbitrary messages to L2.
- Users can also use `L1MessageQueue.appendEnforcedTransaction` to send enforced transactions to L2.

**Send Arbitrary Messages**

In the `L1ScrollMessenger.sendMessage` function, it converts the message to cross domain calldata and passes it to `L1MessageQueue.appendCrossDomainMessage` to append to the `messageQueue`.
This function also estimates the cross domain message fee (more details in [Cross Domain Message Fee](#cross-domain-message-fee)) and deducts it from the transferred Ether. If the amount cannot cover the fee and value to transfer to L2, the transaction will fail.
All excess Ether will be refunded to the designated `refundAddress` or sender if not specified.

After entering into `L1MessageQueue` contract, the `appendCrossDomainMessage` function computes the transaction hash given the target address, calldata, and gas limit.
Note that the transaction hash computed in the contract is the same as the transaction hash of the corresponding L2 transaction for this message.
In addition, the `appendCrossDomainMessage` function can be only called by `L1ScrollMessenger` because the fee is deducted by the

**Send Enforced Transaction**

TBA

<!--
Inside the `sendMessage` function, the `L1ScrollMessenger` contract will call into `L1MessageQueue.appendCrossDomainMessage` to append the cross domain message. Then `L1MessageQueue` will emit a `QueueTransaction` event, which is monitored by the Relayer. The Relayer will wait for the confirmation of the blocks in Layer 1. Currently, the Relayer wait for the blocks to become `safe`. After that, the Relayer will initiate a transaction in layer 2, calling function `L2ScrollMessenger.relayMessage` and finally, the message is executed in layer 2.

The execution in layer 2 may be failed due to out of gas problem. In such case, one can call `L1ScrollMessenger.replayMessage` to replace the message with a larger gas limit. And the Relayer will follow the steps and execute the message again in layer 2.

In the next version, we will replace the Relayer by the L2 seqeuncer to include the L1 message transaction in the L2 blocks. -->

### L1 Message Transaction

### Cross Domain Message Fee

### Address Alias


## Send Message from L2 to L1

![L2 to L1 workflow](assets/L2-to-L1.png)

Similar to sending message from L1 to L2, you should call `L2ScrollMessenger.sendMessage` first in layer 2. The `L2ScrollMessenger` contract will emit a `SentMessage` event, which is monitored by the Relayer. TBA

Different from L1 to L2 message relay, the Relayer won't relay the message until the batch that contains the message is finalized in the `ScrollChain` contract on the Layer 1. The Relayer will generate the withdraw and submit the proof to `ZKRollup` contract in layer 1 again. Finally, anyone can call `L1ScrollMessenger.relayMessageWithProof` with correct proof to execute the message in layer 1.

Currently, for the safety reason, we only allow privileged contracts to send cross domain messages. And only privileged accounts can call `L2ScrollMessenger.relayMessage`.

