// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

/// @title The interface for the ERC1155 cross chain gateway in layer 2.
interface IL2ERC1155Gateway {
  /**************************************** Events ****************************************/

  event FinalizeDepositERC1155(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _tokenId,
    uint256 _amount
  );

  event FinalizeBatchDepositERC1155(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256[] _tokenIds,
    uint256[] _amounts
  );

  event WithdrawERC1155(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _tokenId,
    uint256 _amount
  );

  event BatchWithdrawERC1155(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256[] _tokenIds,
    uint256[] _amounts
  );

  /**************************************** Mutated Funtions ****************************************/

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

  function batchWithdrawERC1155(
    address _token,
    uint256[] memory _tokenIds,
    uint256[] memory _amounts,
    uint256 _gasLimit
  ) external;

  function batchWithdrawERC1155(
    address _token,
    address _to,
    uint256[] memory _tokenIds,
    uint256[] memory _amounts,
    uint256 _gasLimit
  ) external;

  function finalizeDepositERC1155(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256 _tokenId,
    uint256 _amount
  ) external;

  function finalizeBatchDepositERC1155(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256[] calldata _tokenIds,
    uint256[] calldata _amounts
  ) external;
}
