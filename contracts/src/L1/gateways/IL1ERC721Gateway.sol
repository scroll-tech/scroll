// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

/// @title The interface for the ERC721 cross chain gateway on layer 1.
interface IL1ERC721Gateway {
    /**********
     * Events *
     **********/

    /// @notice Emitted when the ERC721 NFT is transfered to recipient on layer 1.
    /// @param _l1Token The address of ERC721 NFT on layer 1.
    /// @param _l2Token The address of ERC721 NFT on layer 2.
    /// @param _from The address of sender on layer 2.
    /// @param _to The address of recipient on layer 1.
    /// @param _tokenId The token id of the ERC721 NFT to withdraw from layer 2.
    event FinalizeWithdrawERC721(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _tokenId
    );

    /// @notice Emitted when the ERC721 NFT is batch transfered to recipient on layer 1.
    /// @param _l1Token The address of ERC721 NFT on layer 1.
    /// @param _l2Token The address of ERC721 NFT on layer 2.
    /// @param _from The address of sender on layer 2.
    /// @param _to The address of recipient on layer 1.
    /// @param _tokenIds The list of token ids of the ERC721 NFT to withdraw from layer 2.
    event FinalizeBatchWithdrawERC721(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256[] _tokenIds
    );

    /// @notice Emitted when the ERC721 NFT is deposited to gateway on layer 1.
    /// @param _l1Token The address of ERC721 NFT on layer 1.
    /// @param _l2Token The address of ERC721 NFT on layer 2.
    /// @param _from The address of sender on layer 1.
    /// @param _to The address of recipient on layer 2.
    /// @param _tokenId The token id of the ERC721 NFT to deposit on layer 1.
    event DepositERC721(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _tokenId
    );

    /// @notice Emitted when the ERC721 NFT is batch deposited to gateway on layer 1.
    /// @param _l1Token The address of ERC721 NFT on layer 1.
    /// @param _l2Token The address of ERC721 NFT on layer 2.
    /// @param _from The address of sender on layer 1.
    /// @param _to The address of recipient on layer 2.
    /// @param _tokenIds The list of token ids of the ERC721 NFT to deposit on layer 1.
    event BatchDepositERC721(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256[] _tokenIds
    );

    /// @notice Emitted when some ERC721 token is refunded.
    /// @param token The address of the token in L1.
    /// @param recipient The address of receiver in L1.
    /// @param tokenId The id of token refunded.
    event RefundERC721(address indexed token, address indexed recipient, uint256 tokenId);

    /// @notice Emitted when a batch of ERC721 tokens are refunded.
    /// @param token The address of the token in L1.
    /// @param recipient The address of receiver in L1.
    /// @param tokenIds The list of token ids of the ERC721 NFT refunded.
    event BatchRefundERC721(address indexed token, address indexed recipient, uint256[] tokenIds);

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Deposit some ERC721 NFT to caller's account on layer 2.
    /// @param _token The address of ERC721 NFT on layer 1.
    /// @param _tokenId The token id to deposit.
    /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
    function depositERC721(
        address _token,
        uint256 _tokenId,
        uint256 _gasLimit
    ) external payable;

    /// @notice Deposit some ERC721 NFT to a recipient's account on layer 2.
    /// @param _token The address of ERC721 NFT on layer 1.
    /// @param _to The address of recipient on layer 2.
    /// @param _tokenId The token id to deposit.
    /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
    function depositERC721(
        address _token,
        address _to,
        uint256 _tokenId,
        uint256 _gasLimit
    ) external payable;

    /// @notice Deposit a list of some ERC721 NFT to caller's account on layer 2.
    /// @param _token The address of ERC721 NFT on layer 1.
    /// @param _tokenIds The list of token ids to deposit.
    /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
    function batchDepositERC721(
        address _token,
        uint256[] calldata _tokenIds,
        uint256 _gasLimit
    ) external payable;

    /// @notice Deposit a list of some ERC721 NFT to a recipient's account on layer 2.
    /// @param _token The address of ERC721 NFT on layer 1.
    /// @param _to The address of recipient on layer 2.
    /// @param _tokenIds The list of token ids to deposit.
    /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
    function batchDepositERC721(
        address _token,
        address _to,
        uint256[] calldata _tokenIds,
        uint256 _gasLimit
    ) external payable;

    /// @notice Complete ERC721 withdraw from layer 2 to layer 1 and send NFT to recipient's account on layer 1.
    /// @dev Requirements:
    ///  - The function should only be called by L1ScrollMessenger.
    ///  - The function should also only be called by L2ERC721Gateway on layer 2.
    /// @param _l1Token The address of corresponding layer 1 token.
    /// @param _l2Token The address of corresponding layer 2 token.
    /// @param _from The address of account who withdraw the token on layer 2.
    /// @param _to The address of recipient on layer 1 to receive the token.
    /// @param _tokenId The token id to withdraw.
    function finalizeWithdrawERC721(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _tokenId
    ) external;

    /// @notice Complete ERC721 batch withdraw from layer 2 to layer 1 and send NFT to recipient's account on layer 1.
    /// @dev Requirements:
    ///  - The function should only be called by L1ScrollMessenger.
    ///  - The function should also only be called by L2ERC721Gateway on layer 2.
    /// @param _l1Token The address of corresponding layer 1 token.
    /// @param _l2Token The address of corresponding layer 2 token.
    /// @param _from The address of account who withdraw the token on layer 2.
    /// @param _to The address of recipient on layer 1 to receive the token.
    /// @param _tokenIds The list of token ids to withdraw.
    function finalizeBatchWithdrawERC721(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256[] calldata _tokenIds
    ) external;
}
