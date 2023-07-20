// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

// Functions needed on top of the ERC20 standard to be compliant with the Scroll bridge
interface IScrollERC20Extension {
    /// @notice Return the address of Gateway the token belongs to.
    function gateway() external view returns (address);

    /// @notice Return the address of counterpart token.
    function counterpart() external view returns (address);

    /// @dev ERC677 Standard, see https://github.com/ethereum/EIPs/issues/677
    /// Defi can use this method to transfer L1/L2 token to L2/L1,
    /// and deposit to L2/L1 contract in one transaction
    function transferAndCall(
        address receiver,
        uint256 amount,
        bytes calldata data
    ) external returns (bool success);

    /// @notice Mint some token to recipient's account.
    /// @dev Gateway Utilities, only gateway contract can call
    /// @param _to The address of recipient.
    /// @param _amount The amount of token to mint.
    function mint(address _to, uint256 _amount) external;

    /// @notice Mint some token from account.
    /// @dev Gateway Utilities, only gateway contract can call
    /// @param _from The address of account to burn token.
    /// @param _amount The amount of token to mint.
    function burn(address _from, uint256 _amount) external;
}
