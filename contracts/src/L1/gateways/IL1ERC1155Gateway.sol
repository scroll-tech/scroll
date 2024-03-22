// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

/// @title The interface for the ERC1155 cross chain gateway on layer 1.
interface IL1ERC1155Gateway {
    /**********
     * Events *
     **********/

    /// @notice Emitted when the ERC1155 NFT is transferred to recipient on layer 1.
    /// @param _l1Token The address of ERC1155 NFT on layer 1.
    /// @param _l2Token The address of ERC1155 NFT on layer 2.
    /// @param _from The address of sender on layer 2.
    /// @param _to The address of recipient on layer 1.
    /// @param _tokenId The token id of the ERC1155 NFT to withdraw from layer 2.
    /// @param _amount The number of token to withdraw from layer 2.
    event FinalizeWithdrawERC1155(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _tokenId,
        uint256 _amount
    );

    /// @notice Emitted when the ERC1155 NFT is batch transferred to recipient on layer 1.
    /// @param _l1Token The address of ERC1155 NFT on layer 1.
    /// @param _l2Token The address of ERC1155 NFT on layer 2.
    /// @param _from The address of sender on layer 2.
    /// @param _to The address of recipient on layer 1.
    /// @param _tokenIds The list of token ids of the ERC1155 NFT to withdraw from layer 2.
    /// @param _amounts The list of corresponding number of token to withdraw from layer 2.
    event FinalizeBatchWithdrawERC1155(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256[] _tokenIds,
        uint256[] _amounts
    );

    /// @notice Emitted when the ERC1155 NFT is deposited to gateway on layer 1.
    /// @param _l1Token The address of ERC1155 NFT on layer 1.
    /// @param _l2Token The address of ERC1155 NFT on layer 2.
    /// @param _from The address of sender on layer 1.
    /// @param _to The address of recipient on layer 2.
    /// @param _tokenId The token id of the ERC1155 NFT to deposit on layer 1.
    /// @param _amount The number of token to deposit on layer 1.
    event DepositERC1155(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _tokenId,
        uint256 _amount
    );

    /// @notice Emitted when the ERC1155 NFT is batch deposited to gateway on layer 1.
    /// @param _l1Token The address of ERC1155 NFT on layer 1.
    /// @param _l2Token The address of ERC1155 NFT on layer 2.
    /// @param _from The address of sender on layer 1.
    /// @param _to The address of recipient on layer 2.
    /// @param _tokenIds The list of token ids of the ERC1155 NFT to deposit on layer 1.
    /// @param _amounts The list of corresponding number of token to deposit on layer 1.
    event BatchDepositERC1155(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256[] _tokenIds,
        uint256[] _amounts
    );

    /// @notice Emitted when some ERC1155 token is refunded.
    /// @param token The address of the token in L1.
    /// @param recipient The address of receiver in L1.
    /// @param tokenId The id of token refunded.
    /// @param amount The amount of token refunded.
    event RefundERC1155(address indexed token, address indexed recipient, uint256 tokenId, uint256 amount);

    /// @notice Emitted when some ERC1155 token is refunded.
    /// @param token The address of the token in L1.
    /// @param recipient The address of receiver in L1.
    /// @param tokenIds The list of ids of token refunded.
    /// @param amounts The list of amount of token refunded.
    event BatchRefundERC1155(address indexed token, address indexed recipient, uint256[] tokenIds, uint256[] amounts);

    /*************************
     * Public View Functions *
     *************************/

    /// @notice Deposit some ERC1155 NFT to caller's account on layer 2.
    /// @param _token The address of ERC1155 NFT on layer 1.
    /// @param _tokenId The token id to deposit.
    /// @param _amount The amount of token to deposit.
    /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
    function depositERC1155(
        address _token,
        uint256 _tokenId,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable;

    /// @notice Deposit some ERC1155 NFT to a recipient's account on layer 2.
    /// @param _token The address of ERC1155 NFT on layer 1.
    /// @param _to The address of recipient on layer 2.
    /// @param _tokenId The token id to deposit.
    /// @param _amount The amount of token to deposit.
    /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
    function depositERC1155(
        address _token,
        address _to,
        uint256 _tokenId,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable;

    /// @notice Deposit a list of some ERC1155 NFT to caller's account on layer 2.
    /// @param _token The address of ERC1155 NFT on layer 1.
    /// @param _tokenIds The list of token ids to deposit.
    /// @param _amounts The list of corresponding number of token to deposit.
    /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
    function batchDepositERC1155(
        address _token,
        uint256[] calldata _tokenIds,
        uint256[] calldata _amounts,
        uint256 _gasLimit
    ) external payable;

    /// @notice Deposit a list of some ERC1155 NFT to a recipient's account on layer 2.
    /// @param _token The address of ERC1155 NFT on layer 1.
    /// @param _to The address of recipient on layer 2.
    /// @param _tokenIds The list of token ids to deposit.
    /// @param _amounts The list of corresponding number of token to deposit.
    /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
    function batchDepositERC1155(
        address _token,
        address _to,
        uint256[] calldata _tokenIds,
        uint256[] calldata _amounts,
        uint256 _gasLimit
    ) external payable;

    /// @notice Complete ERC1155 withdraw from layer 2 to layer 1 and send fund to recipient's account on layer 1.
    ///      The function should only be called by L1ScrollMessenger.
    ///      The function should also only be called by L2ERC1155Gateway on layer 2.
    /// @param _l1Token The address of corresponding layer 1 token.
    /// @param _l2Token The address of corresponding layer 2 token.
    /// @param _from The address of account who withdraw the token on layer 2.
    /// @param _to The address of recipient on layer 1 to receive the token.
    /// @param _tokenId The token id to withdraw.
    /// @param _amount The amount of token to withdraw.
    function finalizeWithdrawERC1155(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _tokenId,
        uint256 _amount
    ) external;

    /// @notice Complete ERC1155 batch withdraw from layer 2 to layer 1 and send fund to recipient's account on layer 1.
    ///      The function should only be called by L1ScrollMessenger.
    ///      The function should also only be called by L2ERC1155Gateway on layer 2.
    /// @param _l1Token The address of corresponding layer 1 token.
    /// @param _l2Token The address of corresponding layer 2 token.
    /// @param _from The address of account who withdraw the token on layer 2.
    /// @param _to The address of recipient on layer 1 to receive the token.
    /// @param _tokenIds The list of token ids to withdraw.
    /// @param _amounts The list of corresponding number of token to withdraw.
    function finalizeBatchWithdrawERC1155(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256[] calldata _tokenIds,
        uint256[] calldata _amounts
    ) external;
}
