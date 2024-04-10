// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

abstract contract LidoBridgeableTokens {
    /*************
     * Constants *
     *************/

    /// @notice The address of bridged token in L1 chain.
    address public immutable l1Token;

    /// @notice The address of the token minted on the L2 chain when token bridged.
    address public immutable l2Token;

    /**********
     * Errors *
     **********/

    /// @dev Thrown the given `l1Token` is not supported.
    error ErrorUnsupportedL1Token();

    /// @dev Thrown the given `l2Token` is not supported.
    error ErrorUnsupportedL2Token();

    /// @dev Thrown the given account is zero address.
    error ErrorAccountIsZeroAddress();

    /// @dev Thrown the `msg.value` is not zero.
    error ErrorNonZeroMsgValue();

    /**********************
     * Function Modifiers *
     **********************/

    /// @dev Validates that passed `_l1Token` is supported by the bridge
    modifier onlySupportedL1Token(address _l1Token) {
        if (_l1Token != l1Token) {
            revert ErrorUnsupportedL1Token();
        }
        _;
    }

    /// @dev Validates that passed `_l2Token` is supported by the bridge
    modifier onlySupportedL2Token(address _l2Token) {
        if (_l2Token != l2Token) {
            revert ErrorUnsupportedL2Token();
        }
        _;
    }

    /// @dev validates that `_account` is not zero address
    modifier onlyNonZeroAccount(address _account) {
        if (_account == address(0)) {
            revert ErrorAccountIsZeroAddress();
        }
        _;
    }

    /***************
     * Constructor *
     ***************/

    /// @param _l1Token The address of the bridged token in the L1 chain
    /// @param _l2Token The address of the token minted on the L2 chain when token bridged
    constructor(address _l1Token, address _l2Token) {
        l1Token = _l1Token;
        l2Token = _l2Token;
    }
}
