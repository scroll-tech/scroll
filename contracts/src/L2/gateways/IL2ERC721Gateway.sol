// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

/// @title The interface for the ERC721 cross chain gateway in layer 2.
interface IL2ERC721Gateway {
  /**************************************** Events ****************************************/

  /// @notice Emitted when the ERC721 NFT is transfered to recipient in layer 2.
  /// @param _l1Token The address of ERC721 NFT in layer 1.
  /// @param _l2Token The address of ERC721 NFT in layer 2.
  /// @param _from The address of sender in layer 1.
  /// @param _to The address of recipient in layer 2.
  /// @param _tokenId The token id of the ERC721 NFT deposited in layer 1.
  event FinalizeDepositERC721(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _tokenId
  );

  /// @notice Emitted when the ERC721 NFT is batch transfered to recipient in layer 2.
  /// @param _l1Token The address of ERC721 NFT in layer 1.
  /// @param _l2Token The address of ERC721 NFT in layer 2.
  /// @param _from The address of sender in layer 1.
  /// @param _to The address of recipient in layer 2.
  /// @param _tokenIds The list of token ids of the ERC721 NFT deposited in layer 1.
  event FinalizeBatchDepositERC721(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256[] _tokenIds
  );

  /// @notice Emitted when the ERC721 NFT is transfered to gateway in layer 2.
  /// @param _l1Token The address of ERC721 NFT in layer 1.
  /// @param _l2Token The address of ERC721 NFT in layer 2.
  /// @param _from The address of sender in layer 2.
  /// @param _to The address of recipient in layer 1.
  /// @param _tokenId The token id of the ERC721 NFT to withdraw in layer 2.
  event WithdrawERC721(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _tokenId
  );

  /// @notice Emitted when the ERC721 NFT is batch transfered to gateway in layer 2.
  /// @param _l1Token The address of ERC721 NFT in layer 1.
  /// @param _l2Token The address of ERC721 NFT in layer 2.
  /// @param _from The address of sender in layer 2.
  /// @param _to The address of recipient in layer 1.
  /// @param _tokenIds The list of token ids of the ERC721 NFT to withdraw in layer 2.
  event BatchWithdrawERC721(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256[] _tokenIds
  );

  /**************************************** Mutated Funtions ****************************************/

  /// @notice Withdraw some ERC721 NFT to caller's account on layer 1.
  /// @param _token The address of ERC721 NFT in layer 2.
  /// @param _tokenId The token id to withdraw.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function withdrawERC721(
    address _token,
    uint256 _tokenId,
    uint256 _gasLimit
  ) external;

  /// @notice Withdraw some ERC721 NFT to caller's account on layer 1.
  /// @param _token The address of ERC721 NFT in layer 2.
  /// @param _to The address of recipient in layer 1.
  /// @param _tokenId The token id to withdraw.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function withdrawERC721(
    address _token,
    address _to,
    uint256 _tokenId,
    uint256 _gasLimit
  ) external;

  /// @notice Batch withdraw a list of ERC721 NFT to caller's account on layer 1.
  /// @param _token The address of ERC721 NFT in layer 2.
  /// @param _tokenIds The list of token ids to withdraw.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function batchWithdrawERC721(
    address _token,
    uint256[] memory _tokenIds,
    uint256 _gasLimit
  ) external;

  /// @notice Batch withdraw a list of ERC721 NFT to caller's account on layer 1.
  /// @param _token The address of ERC721 NFT in layer 2.
  /// @param _to The address of recipient in layer 1.
  /// @param _tokenIds The list of token ids to withdraw.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function batchWithdrawERC721(
    address _token,
    address _to,
    uint256[] memory _tokenIds,
    uint256 _gasLimit
  ) external;

  /// @notice Complete ERC721 deposit from layer 1 to layer 2 and send NFT to recipient's account in layer 2.
  /// @dev Requirements:
  ///  - The function should only be called by L2ScrollMessenger.
  ///  - The function should also only be called by L1ERC721Gateway in layer 1.
  /// @param _l1Token The address of corresponding layer 1 token.
  /// @param _l2Token The address of corresponding layer 2 token.
  /// @param _from The address of account who withdraw the token in layer 1.
  /// @param _to The address of recipient in layer 2 to receive the token.
  /// @param _tokenId The token id to withdraw.
  function finalizeDepositERC721(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256 _tokenId
  ) external;

  /// @notice Complete ERC721 deposit from layer 1 to layer 2 and send NFT to recipient's account in layer 2.
  /// @dev Requirements:
  ///  - The function should only be called by L2ScrollMessenger.
  ///  - The function should also only be called by L1ERC721Gateway in layer 1.
  /// @param _l1Token The address of corresponding layer 1 token.
  /// @param _l2Token The address of corresponding layer 2 token.
  /// @param _from The address of account who withdraw the token in layer 1.
  /// @param _to The address of recipient in layer 2 to receive the token.
  /// @param _tokenIds The list of token ids to withdraw.
  function finalizeBatchDepositERC721(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256[] calldata _tokenIds
  ) external;
}
