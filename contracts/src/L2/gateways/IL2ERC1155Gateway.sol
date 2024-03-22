// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

/// @title The interface for the ERC1155 cross chain gateway on layer 2.
interface IL2ERC1155Gateway {
    /**********
     * Events *
     **********/

    /// @notice Emitted when the ERC1155 NFT is transferred to recipient on layer 2.
    /// @param l1Token The address of ERC1155 NFT on layer 1.
    /// @param l2Token The address of ERC1155 NFT on layer 2.
    /// @param from The address of sender on layer 1.
    /// @param to The address of recipient on layer 2.
    /// @param tokenId The token id of the ERC1155 NFT deposited on layer 1.
    /// @param amount The amount of token deposited.
    event FinalizeDepositERC1155(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256 tokenId,
        uint256 amount
    );

    /// @notice Emitted when the ERC1155 NFT is batch transferred to recipient on layer 2.
    /// @param l1Token The address of ERC1155 NFT on layer 1.
    /// @param l2Token The address of ERC1155 NFT on layer 2.
    /// @param from The address of sender on layer 1.
    /// @param to The address of recipient on layer 2.
    /// @param tokenIds The list of token ids of the ERC1155 NFT deposited on layer 1.
    /// @param amounts The list of corresponding amounts deposited.
    event FinalizeBatchDepositERC1155(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256[] tokenIds,
        uint256[] amounts
    );

    /// @notice Emitted when the ERC1155 NFT is transferred to gateway on layer 2.
    /// @param l1Token The address of ERC1155 NFT on layer 1.
    /// @param l2Token The address of ERC1155 NFT on layer 2.
    /// @param from The address of sender on layer 2.
    /// @param to The address of recipient on layer 1.
    /// @param tokenId The token id of the ERC1155 NFT to withdraw on layer 2.
    /// @param amount The amount of token to withdraw.
    event WithdrawERC1155(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256 tokenId,
        uint256 amount
    );

    /// @notice Emitted when the ERC1155 NFT is batch transferred to gateway on layer 2.
    /// @param l1Token The address of ERC1155 NFT on layer 1.
    /// @param l2Token The address of ERC1155 NFT on layer 2.
    /// @param from The address of sender on layer 2.
    /// @param to The address of recipient on layer 1.
    /// @param tokenIds The list of token ids of the ERC1155 NFT to withdraw on layer 2.
    /// @param amounts The list of corresponding amounts to withdraw.
    event BatchWithdrawERC1155(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256[] tokenIds,
        uint256[] amounts
    );

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Withdraw some ERC1155 NFT to caller's account on layer 1.
    /// @param token The address of ERC1155 NFT on layer 2.
    /// @param tokenId The token id to withdraw.
    /// @param amount The amount of token to withdraw.
    /// @param gasLimit Unused, but included for potential forward compatibility considerations.
    function withdrawERC1155(
        address token,
        uint256 tokenId,
        uint256 amount,
        uint256 gasLimit
    ) external payable;

    /// @notice Withdraw some ERC1155 NFT to caller's account on layer 1.
    /// @param token The address of ERC1155 NFT on layer 2.
    /// @param to The address of recipient on layer 1.
    /// @param tokenId The token id to withdraw.
    /// @param amount The amount of token to withdraw.
    /// @param gasLimit Unused, but included for potential forward compatibility considerations.
    function withdrawERC1155(
        address token,
        address to,
        uint256 tokenId,
        uint256 amount,
        uint256 gasLimit
    ) external payable;

    /// @notice Batch withdraw a list of ERC1155 NFT to caller's account on layer 1.
    /// @param token The address of ERC1155 NFT on layer 2.
    /// @param tokenIds The list of token ids to withdraw.
    /// @param amounts The list of corresponding amounts to withdraw.
    /// @param gasLimit Unused, but included for potential forward compatibility considerations.
    function batchWithdrawERC1155(
        address token,
        uint256[] memory tokenIds,
        uint256[] memory amounts,
        uint256 gasLimit
    ) external payable;

    /// @notice Batch withdraw a list of ERC1155 NFT to caller's account on layer 1.
    /// @param token The address of ERC1155 NFT on layer 2.
    /// @param to The address of recipient on layer 1.
    /// @param tokenIds The list of token ids to withdraw.
    /// @param amounts The list of corresponding amounts to withdraw.
    /// @param gasLimit Unused, but included for potential forward compatibility considerations.
    function batchWithdrawERC1155(
        address token,
        address to,
        uint256[] memory tokenIds,
        uint256[] memory amounts,
        uint256 gasLimit
    ) external payable;

    /// @notice Complete ERC1155 deposit from layer 1 to layer 2 and send NFT to recipient's account on layer 2.
    /// @dev Requirements:
    ///  - The function should only be called by L2ScrollMessenger.
    ///  - The function should also only be called by L1ERC1155Gateway on layer 1.
    /// @param l1Token The address of corresponding layer 1 token.
    /// @param l2Token The address of corresponding layer 2 token.
    /// @param from The address of account who deposits the token on layer 1.
    /// @param to The address of recipient on layer 2 to receive the token.
    /// @param tokenId The token id to deposit.
    /// @param amount The amount of token to deposit.
    function finalizeDepositERC1155(
        address l1Token,
        address l2Token,
        address from,
        address to,
        uint256 tokenId,
        uint256 amount
    ) external;

    /// @notice Complete ERC1155 deposit from layer 1 to layer 2 and send NFT to recipient's account on layer 2.
    /// @dev Requirements:
    ///  - The function should only be called by L2ScrollMessenger.
    ///  - The function should also only be called by L1ERC1155Gateway on layer 1.
    /// @param l1Token The address of corresponding layer 1 token.
    /// @param l2Token The address of corresponding layer 2 token.
    /// @param from The address of account who deposits the token on layer 1.
    /// @param to The address of recipient on layer 2 to receive the token.
    /// @param tokenIds The list of token ids to deposit.
    /// @param amounts The list of corresponding amounts to deposit.
    function finalizeBatchDepositERC1155(
        address l1Token,
        address l2Token,
        address from,
        address to,
        uint256[] calldata tokenIds,
        uint256[] calldata amounts
    ) external;
}
