// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

/// @title The interface for the ERC721 cross chain gateway on layer 2.
interface IL2ERC721Gateway {
    /**********
     * Events *
     **********/

    /// @notice Emitted when the ERC721 NFT is transfered to recipient on layer 2.
    /// @param l1Token The address of ERC721 NFT on layer 1.
    /// @param l2Token The address of ERC721 NFT on layer 2.
    /// @param from The address of sender on layer 1.
    /// @param to The address of recipient on layer 2.
    /// @param tokenId The token id of the ERC721 NFT deposited on layer 1.
    event FinalizeDepositERC721(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256 tokenId
    );

    /// @notice Emitted when the ERC721 NFT is batch transfered to recipient on layer 2.
    /// @param l1Token The address of ERC721 NFT on layer 1.
    /// @param l2Token The address of ERC721 NFT on layer 2.
    /// @param from The address of sender on layer 1.
    /// @param to The address of recipient on layer 2.
    /// @param tokenIds The list of token ids of the ERC721 NFT deposited on layer 1.
    event FinalizeBatchDepositERC721(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256[] tokenIds
    );

    /// @notice Emitted when the ERC721 NFT is transfered to gateway on layer 2.
    /// @param l1Token The address of ERC721 NFT on layer 1.
    /// @param l2Token The address of ERC721 NFT on layer 2.
    /// @param from The address of sender on layer 2.
    /// @param to The address of recipient on layer 1.
    /// @param tokenId The token id of the ERC721 NFT to withdraw on layer 2.
    event WithdrawERC721(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256 tokenId
    );

    /// @notice Emitted when the ERC721 NFT is batch transfered to gateway on layer 2.
    /// @param l1Token The address of ERC721 NFT on layer 1.
    /// @param l2Token The address of ERC721 NFT on layer 2.
    /// @param from The address of sender on layer 2.
    /// @param to The address of recipient on layer 1.
    /// @param tokenIds The list of token ids of the ERC721 NFT to withdraw on layer 2.
    event BatchWithdrawERC721(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256[] tokenIds
    );

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Withdraw some ERC721 NFT to caller's account on layer 1.
    /// @param token The address of ERC721 NFT on layer 2.
    /// @param tokenId The token id to withdraw.
    /// @param gasLimit Unused, but included for potential forward compatibility considerations.
    function withdrawERC721(
        address token,
        uint256 tokenId,
        uint256 gasLimit
    ) external payable;

    /// @notice Withdraw some ERC721 NFT to caller's account on layer 1.
    /// @param token The address of ERC721 NFT on layer 2.
    /// @param to The address of recipient on layer 1.
    /// @param tokenId The token id to withdraw.
    /// @param gasLimit Unused, but included for potential forward compatibility considerations.
    function withdrawERC721(
        address token,
        address to,
        uint256 tokenId,
        uint256 gasLimit
    ) external payable;

    /// @notice Batch withdraw a list of ERC721 NFT to caller's account on layer 1.
    /// @param token The address of ERC721 NFT on layer 2.
    /// @param tokenIds The list of token ids to withdraw.
    /// @param gasLimit Unused, but included for potential forward compatibility considerations.
    function batchWithdrawERC721(
        address token,
        uint256[] memory tokenIds,
        uint256 gasLimit
    ) external payable;

    /// @notice Batch withdraw a list of ERC721 NFT to caller's account on layer 1.
    /// @param token The address of ERC721 NFT on layer 2.
    /// @param to The address of recipient on layer 1.
    /// @param tokenIds The list of token ids to withdraw.
    /// @param gasLimit Unused, but included for potential forward compatibility considerations.
    function batchWithdrawERC721(
        address token,
        address to,
        uint256[] memory tokenIds,
        uint256 gasLimit
    ) external payable;

    /// @notice Complete ERC721 deposit from layer 1 to layer 2 and send NFT to recipient's account on layer 2.
    /// @dev Requirements:
    ///  - The function should only be called by L2ScrollMessenger.
    ///  - The function should also only be called by L1ERC721Gateway on layer 1.
    /// @param l1Token The address of corresponding layer 1 token.
    /// @param l2Token The address of corresponding layer 2 token.
    /// @param from The address of account who withdraw the token on layer 1.
    /// @param to The address of recipient on layer 2 to receive the token.
    /// @param tokenId The token id to withdraw.
    function finalizeDepositERC721(
        address l1Token,
        address l2Token,
        address from,
        address to,
        uint256 tokenId
    ) external;

    /// @notice Complete ERC721 deposit from layer 1 to layer 2 and send NFT to recipient's account on layer 2.
    /// @dev Requirements:
    ///  - The function should only be called by L2ScrollMessenger.
    ///  - The function should also only be called by L1ERC721Gateway on layer 1.
    /// @param l1Token The address of corresponding layer 1 token.
    /// @param l2Token The address of corresponding layer 2 token.
    /// @param from The address of account who withdraw the token on layer 1.
    /// @param to The address of recipient on layer 2 to receive the token.
    /// @param tokenIds The list of token ids to withdraw.
    function finalizeBatchDepositERC721(
        address l1Token,
        address l2Token,
        address from,
        address to,
        uint256[] calldata tokenIds
    ) external;
}
