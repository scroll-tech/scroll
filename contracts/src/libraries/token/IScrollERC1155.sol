// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {IERC1155} from "@openzeppelin/contracts/token/ERC1155/IERC1155.sol";

interface IScrollERC1155 is IERC1155 {
    /// @notice Return the address of Gateway the token belongs to.
    function gateway() external view returns (address);

    /// @notice Return the address of counterpart token.
    function counterpart() external view returns (address);

    /// @notice Mint some token to recipient's account.
    /// @dev Gateway Utilities, only gateway contract can call
    /// @param _to The address of recipient.
    /// @param _tokenId The token id to mint.
    /// @param _amount The amount of token to mint.
    /// @param _data The data passed to recipient
    function mint(
        address _to,
        uint256 _tokenId,
        uint256 _amount,
        bytes memory _data
    ) external;

    /// @notice Burn some token from account.
    /// @dev Gateway Utilities, only gateway contract can call
    /// @param _from The address of account to burn token.
    /// @param _tokenId The token id to burn.
    /// @param _amount The amount of token to burn.
    function burn(
        address _from,
        uint256 _tokenId,
        uint256 _amount
    ) external;

    /// @notice Batch mint some token to recipient's account.
    /// @dev Gateway Utilities, only gateway contract can call
    /// @param _to The address of recipient.
    /// @param _tokenIds The token id to mint.
    /// @param _amounts The list of corresponding amount of token to mint.
    /// @param _data The data passed to recipient
    function batchMint(
        address _to,
        uint256[] calldata _tokenIds,
        uint256[] calldata _amounts,
        bytes calldata _data
    ) external;

    /// @notice Batch burn some token from account.
    /// @dev Gateway Utilities, only gateway contract can call
    /// @param _from The address of account to burn token.
    /// @param _tokenIds The list of token ids to burn.
    /// @param _amounts The list of corresponding amount of token to burn.
    function batchBurn(
        address _from,
        uint256[] calldata _tokenIds,
        uint256[] calldata _amounts
    ) external;
}
