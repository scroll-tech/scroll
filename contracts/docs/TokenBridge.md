# Bridge Token Between Layer 1 and Layer 2

The Token Bridge of Scroll Protocol offers a way to move assets from layer 1 to layer 2 and back, including Ether, ERC20 token, ERC-721 token, ERC-1155 token, etc. The asset should be deposited and locked in layer 1 and then in exchange of the same amount of an equivalent token on layer 2. For example, if you deposit 1000 Ether in layer 1, you will get 1000 Ether in layer 2 for return. And if you withdraw 1000 Ether in layer 2, you will get 1000 Ether in layer 1 for return.

The Ether and ERC20 tokens can be deposited or withdrawn using one single contract `GatewayRouter` (`L1GatewayRouter` in layer 1 and `L2GatewayRouter` in layer 2). The ERC-721 tokens and ERC-1155 tokens can be deposited or withdrawn using the corresponding `ERC1155Gateway` and `ERC721Gateway` in layer 1 or layer 2 (They may be integrated into `GatewayRouter` in the future).

## Bridge Ether

To bridge Ether from layer 1 to layer 2, one can use `L1GatewayRouter.depositETH`. This will transfer ethers to the `L1ScrollMessenger` contract on the layer 1 and credits the same amount of ether to you in layer 2 at the specified address.

```solidity
function depositETH(uint256 _gasLimit) external payable;

function depositETH(address _to, uint256 _gasLimit) external payable;

```

In the layer 1, all deposited Ether will be locked in `L1ScrollMessenger` contract. It means your deposited Ether will firstly be transfered to `L1GatewayRouter` contract and then to `L1ScrollMessenger` contract.

To withdraw Ether from layer 2 to layer 1, one can use `L2GatewayRouter.withdrawETH`.

```solidity
function withdrawETH(uint256 _gasLimit) external payable;

function withdrawETH(address _to, uint256 _gasLimit) external payable;

```

In layer 2, the `L2ScrollMessenger` holds infinite amount of Ether at the beginning. All your withdrawn Ether will be transfered back to `L2ScrollMessenger`, just like the process in layer 1.

In addition, you can actually call `sendMessage` from the `L1ScrollMessenger` or `L2ScrollMessenger` contract to deposit or withdraw Ether. The `L1GatewayRouter.depositETH` and `L2GatewayRouter.withdrawETH` are just alias for `L1ScrollMessenger/L2ScrollMessenger.sendMessage`.

## Bridge ERC20 Tokens

We use the similar design as [Arbitrum protocol](https://developer.offchainlabs.com/docs/bridging_assets#bridging-erc20-tokens) do. Several gateway contracts are used to bridge different kinds of ERC20 tokens, such as Wrapped Ether, standard ERC20 tokens, etc.

We implement a `StandardERC20Gateway` to deposit and withdraw standard ERC20 tokens. The standard procedure to deposit ERC20 tokens is to call `L1GatewayRouter.depositERC20` in layer 1. The token will be locked in `L1StandardERC20Gateway` contract in layer 1. The the standard procedure to withdraw ERC20 tokens is to call `L2GatewayRouter.withdrawRC20` in layer 2 and the token will be burned in layer 2.

For many other non-standard ERC20 tokens, we provide a custom ERC20 gateway. Anyone can implement such gateway as long as it implements all required interfaces. We implement the Wrapped Ether gateway as an example. To deposit or withdraw Wrapped Ether, one should first unwrap it to Ether, then transfer the Ether to `ScrollMessenger` just like Ether bridging.

### Passing data when depositing ERC20 tokens

The Scroll protocol offer a way to call another contract after depositing the token in layer 2 by calling `L1GatewayRouter.depositERC20AndCall` in layer 1. The ERC20 token in layer 2 implements the [ERC 677 Standard](https://github.com/ethereum/EIPs/issues/677). By using `transferAndCall` function, we can transfer the token to corresponding recipient in layer 2 and then call the recipient with passed data.

```solidity
function depositERC20AndCall(
  address _token,
  address _to,
  uint256 _amount,
  bytes memory _data,
  uint256 _gasLimit
) external;

```

Like Bridging Ether, all above functionality can be achieved by calling corresponding function in ERC20Gateway contract.

## Bridge ERC-721/ERC-1155 Tokens

The depositing/withdrawing ERC-721 or ERC-1155 tokens works very similar to ERC20 tokens. One can use the following function to deposit ERC-721/ERC-1155 tokens in layer 1.

```solidity
function depositERC1155(
  address _token,
  uint256 _tokenId,
  uint256 _amount,
  uint256 _gasLimit
) external;

function depositERC1155(
  address _token,
  address _to,
  uint256 _tokenId,
  uint256 _amount,
  uint256 _gasLimit
) external;

function depositERC721(
  address _token,
  uint256 _tokenId,
  uint256 _gasLimit
) external;

function depositERC721(
  address _token,
  address _to,
  uint256 _tokenId,
  uint256 _gasLimit
) external;

```

One can use the following function to withdraw ERC-721/ERC-1155 tokens in layer 2.

```solidity
function withdrawERC1155(
  address _token,
  uint256 _tokenId,
  uint256 _amount,
  uint256 _gasLimit
) external;

function withdrawERC1155(
  address _token,
  address _to,
  uint256 _tokenId,
  uint256 _amount,
  uint256 _gasLimit
) external;

function withdrawERC721(
  address _token,
  uint256 _tokenId,
  uint256 _gasLimit
) external;

function withdrawERC721(
  address _token,
  address _to,
  uint256 _tokenId,
  uint256 _gasLimit
) external;

```

To save the gas usage, we also provide a batch deposit/withdraw function, such as `batchDepositERC1155` and `batchDepositERC721`, by passing a list of token ids to the function.

## Drop Depositing/Withdrawing

Coming soon...

## Force Exit

Coming soon...
