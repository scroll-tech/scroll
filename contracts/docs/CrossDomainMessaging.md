# Cross Domain Messaging

Like other layer 2 protocol, Scroll allow dapps to communicate between layer 1 and layer 2. More specifically, dapps on layer 1 can trigger contract functions in layer 2, and vice versa.

## Message Between L1 and L2

The Scroll protocol implements two core contracts `L1ScrollMessenger` and `L2ScrollMessenger` to enable cross domain messaging. The only entry to send cross domain message is to call the following function:

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

The function is attached in both messenger in layer 1 and layer 2. After that, the Sequencer or relayer will handle the rest part for you. We will explain the detailed workflow in the following docs.

### Send Message from L1 to L2

As described above, the first step is to call `L1ScrollMessenger.sendMessage` in layer 1. Inside the `sendMessage` function, the `L1ScrollMessenger` contract will call into `L1MessageQueue.appendCrossDomainMessage` to append the cross domain message. Then `L1MessageQueue` will emit a `QueueTransaction` event, which is monitored by the Relayer. The Relayer will wait for the confirmation of the blocks in Layer 1. Currently, the Relayer wait for the blocks to become `safe`. After that, the Relayer will initiate a transaction in layer 2, calling function `L2ScrollMessenger.relayMessage` and finally, the message is executed in layer 2.

The execution in layer 2 may be failed due to out of gas problem. In such case, one can call `L1ScrollMessenger.replayMessage` to replace the message with a larger gas limit. And the Relayer will follow the steps and execute the message again in layer 2.

In the next version, we will replace the Relayer by the L2 seqeuncer to include the L1 messages as a new type of transaciton

### Send Message from L2 to L1

Similar to sending message from L1 to L2, you should call `L2ScrollMessenger.sendMessage` first in layer 2. The `L2ScrollMessenger` contract will emit a `SentMessage` event, which is monitored by the Relayer. TBA

Different from L1 to L2 message relay, the Relayer won't relay the message until the batch that contains the message is finalized in the `ScrollChain` contract on the Layer 1. The Relayer will generate the withdraw and submit the proof to `ZKRollup` contract in layer 1 again. Finally, anyone can call `L1ScrollMessenger.relayMessageWithProof` with correct proof to execute the message in layer 1.

Currently, for the safety reason, we only allow privileged contracts to send cross domain messages. And only privileged accounts can call `L2ScrollMessenger.relayMessage`.

## Fee For Sending Message

to be discussed.
