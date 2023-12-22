# Lido's Scroll Bridge

The document details the implementation of the bridging of the ERC20 compatible tokens[^*] between Ethereum and Scroll chains.

It's the first step of Lido's integration into the Scroll protocol. The main goal of the current implementation is to be the strong foundation for the long-term goals of the Lido expansion in the Scroll chain. The long-run picture of the Lido's integration into L2s includes:

- Bridging of Lido's tokens from L1 to L2 chains
- Instant ETH staking on L2 chains with receiving stETH/wstETH on the corresponding L2 immediately
- Keeping UX on L2 as close as possible to the UX on Ethereum mainnet

At this point, the implementation must provide a scalable and reliable solution for Lido to bridge ERC20 compatible tokens between Scroll and the Ethereum chain.

[^*]: The current implementation might not support the non-standard functionality of the ERC20 tokens. For example, rebasable tokens or tokens with transfers fee will work incorrectly. In case your token implements some non-typical ERC20 logic, make sure it is compatible with the bridge before usage.

## Security surface overview

| Statement                                                                                                                                    | Answer                                                                                                                                                                                                                                                                             |
| -------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| It is possible to bridge wstETH forth and back using this bridge                                                                             | Yes                                                                                                                                                                                                                                                                                |
| The bridge using a canonical mechanism for message/value passing                                                                             | Yes                                                                                                                                                                                                                                                                                |
| The bridge is upgradeable                                                                                                                    | Yes                                                                                                                                                                                                                                                                                |
| Upgrade authority for the bridge                                                                                                             | TBA                                                                                                                                                                                                                                                                                |
| Emergency pause/cancel mechanisms and their authorities                                                                                      | TBA                                                                                                                                                                                                                                                                                |
| The bridged token support permits and ERC-1271                                                                                               | No, only permits                                                                                                                                                                                                                                                                   |
| Are the following things in the scope of this bridge deployment:                                                                             |                                                                                                                                                                                                                                                                                    |
| - Passing the (w)stETH/USD price feed                                                                                                        | No                                                                                                                                                                                                                                                                                 |
| - Passing Lido DAO governance decisions                                                                                                      | [Lido DAO Agent](https://etherscan.io/address/0x3e40D73EB977Dc6a537aF587D48316feE66E9C8c) representation [on Scroll (address TBD)] via [ScrollBridgeExecutor](https://github.com/scroll-tech/governance-crosschain-bridges/blob/scroll/contracts/bridges/ScrollBridgeExecutor.sol) |
| Bridges are complicated in that the transaction can succeed on one side and fail on the other. What's the handling mechanism for this issue? | TBA                                                                                                                                                                                                                                                                                |
| Is there a deployment script that sets all the parameters and authorities correctly?                                                         | No, we are upgrading from existing gateway, will need to involve multisig operation by Scroll                                                                                                                                                                                      |
| Is there a post-deploy check script that, given a deployment, checks that all parameters and authorities are set correctly?                  | No                                                                                                                                                                                                                                                                                 |

## Scroll's Bridging Flow

The default implementation of the Scroll bridging solution consists of two parts: `L1StandardERC20Gateway` and `L2StandardERC20Gateway`. These contracts allow bridging the ERC20 tokens between Ethereum and Scroll chains.

In the standard bridge, when ERC20 is deposited on L1 and transferred to the bridge contract it remains "locked" there while the equivalent amount is minted in the L2 token. For withdrawals, the opposite happens the L2 token amount is burned then the same amount of L1 tokens is transferred to the recipient.

The default Scroll bridge is suitable for the short-term goal of the Lido (bridging of the wstETH token into Scroll), but it complicates the achievement of the long-term goals. For example, implementation of the staking from L2's very likely will require extending the token and gateway implementations.

Additionally, Scroll provides functionality to implement the custom bridge solution utilizing the same cross-domain infrastructure as the Standard bridge. The only constraint for the custom bridge to be compatible with the default Scroll Gateway is the implementation of the `IL1ERC20Gateway` and `IL2ERC20Gateway` interfaces.

The rest of the document provides a technical specification of the bridge Lido will use to transfer tokens between Ethereum and Scroll chains.

## Lido's Bridge Implementation

The current implementation of the tokens bridge provides functionality to bridge the specified type of ERC20 compatible token between Ethereum and Scroll chains. Additionally, the bridge provides some administrative features, like the **temporary disabling of the deposits and withdrawals**. It's necessary when bridging must be disabled fast because of the malicious usage of the bridge or vulnerability in the contracts. Also, it might be helpful in the implementation upgrade process.

The technical implementation focuses on the following requirements for the contracts:

- **Scalability** - current implementation must provide the ability to be extended with new functionality in the future.
- **Simplicity** - implemented contracts must be clear, simple, and expressive for developers who will work with code in the future.
- **Gas efficiency** - implemented solution must be efficient in terms of gas costs for the end-user, but at the same time, it must not violate the previous requirement.

A high-level overview of the proposed solution might be found in the below diagram:

![](https://i.imgur.com/7UaVuto.png)

- [**`LidoGatewayManager`**](./LidoGatewayManager.sol) - contains administrative methods to retrieve and control the state of the bridging process.
- [**`LidoBridgeableTokens`**](./LidoBridgeableTokens.sol) - contains the logic for validation of tokens used in the bridging process.
- [**`L1LidoGateway`**](./L1LidoGateway.sol) - Ethereum's counterpart of the bridge to bridge registered ERC20 compatible tokens between Ethereum and Scroll chains.
- [**`L2LidoGateway`**](./L2LidoGateway.sol) - Scroll's counterpart of the bridge to bridge registered ERC20 compatible tokens between Ethereum and Scroll chains
- [**`ScrollStandardERC20`**](../libraries/token/ScrollStandardERC20.sol) - an implementation of the `ERC20` token with administrative methods to mint and burn tokens.
- [**`TransparentUpgradeableProxy`**](https://github.com/OpenZeppelin/openzeppelin-contracts/blob/master/contracts/proxy/transparent/TransparentUpgradeableProxy.sol) - the ERC1967 proxy with extra admin functionality.

## Scroll's Bridging Flow

The general process of bridging tokens via Scroll's Lido bridge can be found here: [ETH and ERC20 Token Bridge](https://docs.scroll.io/en/developers/l1-and-l2-bridging/eth-and-erc20-token-bridge/).

## Deployment Process

To reduce the gas costs for users, contracts `L1LidoGateway`, `L2LidoGateway`, and `ScrollStandardERC20` contracts use immutable variables as much as possible. But some of those variables are cross-referred. For example, `L1LidoGateway` has reference to `L2LidoGateway` and vice versa. As we use proxies, we can deploy proxies at first and without calling the `initialize` function from each gateway. Then call the `initialize` function with correct contract addresses.

Another option - pre-calculate the future address of the deployed contract offchain and deployed the implementation using pre-calculated addresses. But it is less fault-tolerant than the solution above.

## Integration Risks

As an additional link in the tokens flow chain, the Scroll protocol and bridges add points of failure. Below are the main risks of the current integration:

### Minting of uncollateralized L2 token

Such an attack might happen if an attacker obtains the right to call `L2LidoGateway.finalizeDepositERC20()` directly. In such a scenario, an attacker can mint uncollaterized tokens on L2 and initiate withdrawal later.

The best way to detect such an attack is an offchain monitoring of the minting and depositing/withdrawal events. Based on such events might be tracked following stats:

- `l1ERC20TokenBridgeBalance` - a total number of locked tokens on the L1 bridge contract
- `l2TokenTotalSupply` - total number of minted L2 tokens
- `l2TokenNotWithdrawn` - total number of burned L2 tokens which aren’t withdrawn from the L1 bridge

At any time following invariant must be satisfied: `l1ERC20TokenBridgeBalance == l2TokenTotalSupply + l2TokenNotWithdrawn`.

In the case of invariant violation, Lido will have a dispute period to suspend the L1 and L2 bridges. Disabled bridges forbid the minting of L2Token and withdrawal of minted tokens till the resolution of the issue.

### Attack to L1ScrollMessenger

According to the Scroll documentation, `L1ScrollMessenger`:

> The L1 Scroll Messenger contract sends messages from L1 to L2 and relays messages from L2 onto L1.

This contract is central in the L2-to-L1 communication process since all messages from L2 that verified by the zkevm proof are executed on behalf of this contract.

In case of a vulnerability in the `L1ScrollMessenger`, which allows the attacker to send arbitrary messages bypassing the the zkevm proof, an attacker can immediately drain tokens from the L1 bridge.

Additional risk creates the upgradeability of the `L1ScrollMessenger`. Exist a risk of an attack with the replacement of the implementation with some malicious functionality. Such an attack might be reduced to the above vulnerability and steal all locked tokens on the L1 bridge.

To respond quickly to such an attack, Lido can set up monitoring of the Proxy contract, which will ring the alarm in case of an implementation upgrade.
