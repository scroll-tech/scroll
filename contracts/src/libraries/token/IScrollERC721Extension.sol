// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

// Functions needed on top of the ERC721 standard to be compliant with the Scroll bridge
interface IScrollERC721Extension {
    /// @notice Return the address of Gateway the token belongs to.
    function gateway() external view returns (address);

    /// @notice Return the address of counterpart token.
    function counterpart() external view returns (address);

    /// @notice Mint some token to recipient's account.
    /// @dev Gateway Utilities, only gateway contract can call
    /// @param _to The address of recipient.
    /// @param _tokenId The token id to mint.
    function mint(address _to, uint256 _tokenId) external;

    /// @notice Burn some token from account.
    /// @dev Gateway Utilities, only gateway contract can call
    /// @param _tokenId The token id to burn.
    function burn(uint256 _tokenId) external;
}
