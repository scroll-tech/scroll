# Deposit Tokens from L1 to L2

This section describes how users and developers deposit tokens from L1 to L2. The deposit transaction is initialized from L1 by calling into the gateway contracts.
We provide a few standard gateways for different types of tokens, listed in the table below.

| Gateway Contract         | Description                                                        |
|--------------------------|--------------------------------------------------------------------|
| `L1GatewayRouter`        | The gateway router supports the deposit of ETH and ERC20 tokens.   |
| `L1ETHGateway`           | The gateway to deposit ETH.                                        |
| `L1StandardERC20Gateway` | The gateway for standard ERC20 token deposits.                     |
| `L1CustomERC20Gateway`   | The gateway for custom ERC20 token deposits.                       |
| `L1WETHGateway`          | The gateway for Wrapped ETH deposits.                              |
| `L1ERC721Gateway`        | The gateway for ERC-721 token deposits.                            |
| `L1ERC1155Gateway`       | The gateway for ERC-1155 token deposits.                           |

The figure below depicts the deposit workflow from L1 to L2. Users call the gateways to initialize the token deposit. The deposit is then encoded to a message and sent to the `L1ScrollMessenger` and appended to the `L1MessageQueue`. Afterwards, the L2 sequencer picks up the new L1 transaction events and then include corresponding transactions in the L2 blocks to finalize the deposits on L2.
The subsequent sections describe the details of how different tokens are deposited.

![Deposit Workflow](assets/deposit.png)

## Deposit ETH

To deposit ETH from L1 to L2, one can use `L1GatewayRouter.depositETH` or `L1GatewayRouter.depositETHAndCall`:
```solidity
function depositETH(uint256 _amount, uint256 _gasLimit) external payable;

function depositETH(address _to, uint256 _amount, uint256 _gasLimit) public payable;

function depositETHAndCall(address _to, uint256 _amount, bytes calldata _data, uint256 _gasLimit) external payable;
```

This transaction will call into `L1ETHGateway` and then `L1EthGateway` will encode the deposit as a message sent to the `L1ScrollMessenger` contract.
The deposited ETH will be locked in the `L1ScrollMessenger` contract after relay fee is deducted from the total amount.
In addition, `depositETHAndCall` can transfer ETH and execute a contract call at the same time.

After the deposit transaction is finalized on the L1, the sequencer will then include a corresponding L2 transaction in the L2 block that transfers the same amount of ETH to the target address.
The L2 transaction calls `L2ScrollMessenger.relayMessage`, which is then routed to `L2ETHGateway.finalizeDepositETH` with the deposited ETH amount.
We allocated a sufficient amount of ETH to `L2ScrollMessenger` contract during the genesis so that `L2ScrollMessenger` can transfer native ETH without minting new ETH tokens.

## Deposit ERC20 Tokens

To deposit ERC20 tokens from L1 to L2, one can use `L1GatewayRouter.depositERC20` or `L1GatewayRouter.depositERC20AndCall`:

```solidity
function depositERC20(address _token, uint256 _amount, uint256 _gasLimit) external payable;

function depositERC20(address _token, address _to, uint256 _amount, uint256 _gasLimit) external payable;

function depositERC20AndCall(address _token, address _to, uint256 _amount, bytes memory _data, uint256 _gasLimit) public payable;
```

We use a similar design as [Arbitrum protocol](https://developer.offchainlabs.com/docs/bridging_assets#bridging-erc20-tokens). Several gateway contracts are used to bridge different kinds of ERC20 tokens, such as standard ERC20 tokens, custom ERC20 tokens, and Wrapped ETH token.
`L1GatewayRouter` records the mapping of ERC20 tokens to the corresponding ERC20 gateway on the L1.
`L1GatewayRouter` uses `StandardERC20Gateway` as the ERC20 gateway for a new ERC20 token by default unless otherwise specified.

We implement a `StandardERC20Gateway` to deposit and withdraw standard ERC20 tokens. The standard procedure to deposit ERC20 tokens is to call `L1GatewayRouter.depositERC20` on the L1. The token will be locked in `L1StandardERC20Gateway` contract.
The first time an ERC20 token is deposited via `L1StandardERC20Gateway`, the `L1StandardERC20Gateway` contract will compute the deterministic ERC20 contract address on the L2 and encode additional information for the `L2StandardERC20Gateway` to deploy a new ERC20 contract using a factory contract on the L2.

For other non-standard ERC20 tokens, we provide a custom ERC20 gateway. Anyone can implement such gateway as long as it implements all required [interfaces](../src/L1/gateways/IL1ERC20Gateway.sol). We implement the Wrapped ETH gateway as an example. To deposit or withdraw Wrapped ETH, one should first unwrap it to ETH, then transfer the ETH to `L1ScrollMessenger` just like Ether bridging.

## Deposit ERC-721/ERC-1155 Tokens

The deposit of ERC-721 or ERC-1155 tokens works very similar to ERC20 tokens. One can use the `L1ERC721Gateway.depositERC721` or `L1ERC1155Gateway.depositERC1155.depositERC1155` functions to deposit ERC-721/ERC-1155 tokens on the L1.

```solidity
function depositERC721(address _token, uint256 _tokenId, uint256 _gasLimit) external payable;

function depositERC721(address _token, address _to, uint256 _tokenId, uint256 _gasLimit) external payable;

function depositERC1155(address _token, uint256 _tokenId, uint256 _amount, uint256 _gasLimit) external payable;

function depositERC1155(address _token, address _to, uint256 _tokenId, uint256 _amount, uint256 _gasLimit) external payable;
```

To facilitate a large amount of ERC-721 or ERC-1155 token deposits, we also provide batch deposit functions in the `L1ERC721Gateway` and `L1ERC1155Gateway` contract via the following functions:

```solidity
function batchDepositERC721(address _token, uint256[] calldata _tokenIds, uint256 _gasLimit) external payable;

function batchDepositERC721(address _token, address _to, uint256[] calldata _tokenIds, uint256 _gasLimit) external payable;

function depositERC1155(address _token, uint256[] calldata _tokenIds, uint256[] calldata _amounts, uint256 _gasLimit) external payable;

function depositERC1155(address _token, address _to, uint256[] calldata _tokenIds, uint256[] calldata _amounts, uint256 _gasLimit) external payable;
```
