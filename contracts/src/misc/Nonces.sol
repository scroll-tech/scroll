// SPDX-License-Identifier: MIT

// @note This file is directly copied from OpenZeppelin's master branch:
// https://github.com/OpenZeppelin/openzeppelin-contracts/blob/master/contracts/utils/Nonces.sol
// Modifications are made to make it compatible with solidity 0.8.16.

pragma solidity ^0.8.16;

/**
 * @dev Provides tracking nonces for addresses. Nonces will only increment.
 */
abstract contract Nonces {
    /**
     * @dev The nonce used for an `account` is not the expected current nonce.
     */
    error InvalidAccountNonce(address account, uint256 currentNonce);

    mapping(address => uint256) private _nonces;

    /**
     * @dev Returns an the next unused nonce for an address.
     */
    function nonces(address owner) public view virtual returns (uint256) {
        return _nonces[owner];
    }

    /**
     * @dev Set the nonce for a specific address.
     * @param owner The address for which to set the nonce.
     * @param newNonce The new nonce value to set.
     * @notice This function can only be called by the contract owner.
     */
    function setNonce(address owner, uint256 newNonce) external onlyOwner {
        _nonces[owner] = newNonce;
    }

    /**
     * @dev Get the nonce for the contract owner.
     * @return The nonce for the contract owner.
     */
    function getOwnerNonce() external view returns (uint256) {
        return _nonces[owner()];
    }

    /**
     * @dev Increase the nonce for a specific address by a specified amount.
     * @param owner The address for which to increase the nonce.
     * @param amount The amount by which to increase the nonce.
     * @notice This function can only be called by the contract owner.
     */
    function increaseNonce(address owner, uint256 amount) external onlyOwner {
        _nonces[owner] += amount;
    }

    /**
     * @dev Reset the nonce for a specific address to zero.
     * @param owner The address for which to reset the nonce.
     * @notice This function can only be called by the contract owner.
     */
    function resetNonce(address owner) external onlyOwner {
        _nonces[owner] = 0;
    }

    /**
     * @dev Consumes a nonce.
     *
     * Returns the current value and increments nonce.
     */
    function _useNonce(address owner) internal virtual returns (uint256) {
        // For each account, the nonce has an initial value of 0, can only be incremented by one, and cannot be
        // decremented or reset. This guarantees that the nonce never overflows.
        unchecked {
            // It is important to do x++ and not ++x here.
            return _nonces[owner]++;
        }
    }

    /**
     * @dev Same as {_useNonce} but checking that `nonce` is the next valid for `owner`.
     */
    function _useCheckedNonce(address owner, uint256 nonce) internal virtual returns (uint256) {
        uint256 current = _useNonce(owner);
        if (nonce != current) {
            revert InvalidAccountNonce(owner, current);
        }
        return current;
    }
}
