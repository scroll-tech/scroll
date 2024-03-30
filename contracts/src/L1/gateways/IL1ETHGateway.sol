// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

interface IL1ETHGateway {
    /**********
     * Events *
     **********/

    /// @notice Emitted when ETH is withdrawn from L2 to L1 and transfer to recipient.
    /// @param from The address of sender in L2.
    /// @param to The address of recipient in L1.
    /// @param amount The amount of ETH withdrawn from L2 to L1.
    /// @param data The optional calldata passed to recipient in L1.
    event FinalizeWithdrawETH(address indexed from, address indexed to, uint256 amount, bytes data);

    /// @notice Emitted when someone deposit ETH from L1 to L2.
    /// @param from The address of sender in L1.
    /// @param to The address of recipient in L2.
    /// @param amount The amount of ETH will be deposited from L1 to L2.
    /// @param data The optional calldata passed to recipient in L2.
    event DepositETH(address indexed from, address indexed to, uint256 amount, bytes data);

    /// @notice Emitted when some ETH is refunded.
    /// @param recipient The address of receiver in L1.
    /// @param amount The amount of ETH refunded to receiver.
    event RefundETH(address indexed recipient, uint256 amount);

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Deposit ETH to caller's account in L2.
    /// @param amount The amount of ETH to be deposited.
    /// @param gasLimit Gas limit required to complete the deposit on L2.
    function depositETH(uint256 amount, uint256 gasLimit) external payable;

    /// @notice Deposit ETH to some recipient's account in L2.
    /// @param to The address of recipient's account on L2.
    /// @param amount The amount of ETH to be deposited.
    /// @param gasLimit Gas limit required to complete the deposit on L2.
    function depositETH(
        address to,
        uint256 amount,
        uint256 gasLimit
    ) external payable;

    /// @notice Deposit ETH to some recipient's account in L2 and call the target contract.
    /// @param to The address of recipient's account on L2.
    /// @param amount The amount of ETH to be deposited.
    /// @param data Optional data to forward to recipient's account.
    /// @param gasLimit Gas limit required to complete the deposit on L2.
    function depositETHAndCall(
        address to,
        uint256 amount,
        bytes calldata data,
        uint256 gasLimit
    ) external payable;

    /// @notice Complete ETH withdraw from L2 to L1 and send fund to recipient's account in L1.
    /// @dev This function should only be called by L1ScrollMessenger.
    ///      This function should also only be called by L1ETHGateway in L2.
    /// @param from The address of account who withdraw ETH in L2.
    /// @param to The address of recipient in L1 to receive ETH.
    /// @param amount The amount of ETH to withdraw.
    /// @param data Optional data to forward to recipient's account.
    function finalizeWithdrawETH(
        address from,
        address to,
        uint256 amount,
        bytes calldata data
    ) external payable;
}
