// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {ERC1155HolderUpgradeable, ERC1155ReceiverUpgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC1155/utils/ERC1155HolderUpgradeable.sol";

import {IL2ERC1155Gateway} from "./IL2ERC1155Gateway.sol";
import {IL2ScrollMessenger} from "../IL2ScrollMessenger.sol";
import {IL1ERC1155Gateway} from "../../L1/gateways/IL1ERC1155Gateway.sol";
import {ScrollGatewayBase} from "../../libraries/gateway/ScrollGatewayBase.sol";
import {IScrollERC1155} from "../../libraries/token/IScrollERC1155.sol";

/// @title L2ERC1155Gateway
/// @notice The `L2ERC1155Gateway` is used to withdraw ERC1155 compatible NFTs on layer 2 and
/// finalize deposit the NFTs from layer 1.
/// @dev The withdrawn NFTs tokens will be burned directly. On finalizing deposit, the corresponding
/// NFT will be minted and transferred to the recipient.
///
/// This will be changed if we have more specific scenarios.
contract L2ERC1155Gateway is ERC1155HolderUpgradeable, ScrollGatewayBase, IL2ERC1155Gateway {
    /**********
     * Events *
     **********/

    /// @notice Emitted when token mapping for ERC1155 token is updated.
    /// @param l2Token The address of corresponding ERC1155 token in layer 2.
    /// @param oldL1Token The address of the old corresponding ERC1155 token in layer 1.
    /// @param newL1Token The address of the new corresponding ERC1155 token in layer 1.
    event UpdateTokenMapping(address indexed l2Token, address indexed oldL1Token, address indexed newL1Token);

    /*************
     * Variables *
     *************/

    /// @notice Mapping from layer 2 token address to layer 1 token address for ERC1155 NFT.
    // solhint-disable-next-line var-name-mixedcase
    mapping(address => address) public tokenMapping;

    /***************
     * Constructor *
     ***************/

    /// @notice Constructor for `L2ERC1155Gateway` implementation contract.
    ///
    /// @param _counterpart The address of `L1ERC1155Gateway` contract in L1.
    /// @param _messenger The address of `L2ScrollMessenger` contract in L2.
    constructor(address _counterpart, address _messenger) ScrollGatewayBase(_counterpart, address(0), _messenger) {
        _disableInitializers();
    }

    /// @notice Initialize the storage of `L2ERC1155Gateway`.
    ///
    /// @dev The parameters `_counterpart` and `_messenger` are no longer used.
    ///
    /// @param _counterpart The address of `L1ERC1155Gateway` contract in L1.
    /// @param _messenger The address of `L2ScrollMessenger` contract in L2.
    function initialize(address _counterpart, address _messenger) external initializer {
        ERC1155HolderUpgradeable.__ERC1155Holder_init();
        ERC1155ReceiverUpgradeable.__ERC1155Receiver_init();

        ScrollGatewayBase._initialize(_counterpart, address(0), _messenger);
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL2ERC1155Gateway
    function withdrawERC1155(
        address _token,
        uint256 _tokenId,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        _withdrawERC1155(_token, _msgSender(), _tokenId, _amount, _gasLimit);
    }

    /// @inheritdoc IL2ERC1155Gateway
    function withdrawERC1155(
        address _token,
        address _to,
        uint256 _tokenId,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        _withdrawERC1155(_token, _to, _tokenId, _amount, _gasLimit);
    }

    /// @inheritdoc IL2ERC1155Gateway
    function batchWithdrawERC1155(
        address _token,
        uint256[] calldata _tokenIds,
        uint256[] calldata _amounts,
        uint256 _gasLimit
    ) external payable override {
        _batchWithdrawERC1155(_token, _msgSender(), _tokenIds, _amounts, _gasLimit);
    }

    /// @inheritdoc IL2ERC1155Gateway
    function batchWithdrawERC1155(
        address _token,
        address _to,
        uint256[] calldata _tokenIds,
        uint256[] calldata _amounts,
        uint256 _gasLimit
    ) external payable override {
        _batchWithdrawERC1155(_token, _to, _tokenIds, _amounts, _gasLimit);
    }

    /// @inheritdoc IL2ERC1155Gateway
    function finalizeDepositERC1155(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _tokenId,
        uint256 _amount
    ) external virtual onlyCallByCounterpart nonReentrant {
        require(_l1Token != address(0), "token address cannot be 0");
        require(_l1Token == tokenMapping[_l2Token], "l2 token mismatch");

        IScrollERC1155(_l2Token).mint(_to, _tokenId, _amount, "");

        emit FinalizeDepositERC1155(_l1Token, _l2Token, _from, _to, _tokenId, _amount);
    }

    /// @inheritdoc IL2ERC1155Gateway
    function finalizeBatchDepositERC1155(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256[] calldata _tokenIds,
        uint256[] calldata _amounts
    ) external virtual onlyCallByCounterpart nonReentrant {
        require(_l1Token != address(0), "token address cannot be 0");
        require(_l1Token == tokenMapping[_l2Token], "l2 token mismatch");

        IScrollERC1155(_l2Token).batchMint(_to, _tokenIds, _amounts, "");

        emit FinalizeBatchDepositERC1155(_l1Token, _l2Token, _from, _to, _tokenIds, _amounts);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update layer 2 to layer 1 token mapping.
    /// @param _l2Token The address of corresponding ERC1155 token on layer 2.
    /// @param _l1Token The address of ERC1155 token on layer 1.
    function updateTokenMapping(address _l2Token, address _l1Token) external onlyOwner {
        require(_l1Token != address(0), "token address cannot be 0");

        address _oldL1Token = tokenMapping[_l2Token];
        tokenMapping[_l2Token] = _l1Token;

        emit UpdateTokenMapping(_l2Token, _oldL1Token, _l1Token);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to withdraw ERC1155 NFT to layer 2.
    /// @param _token The address of ERC1155 NFT on layer 1.
    /// @param _to The address of recipient on layer 2.
    /// @param _tokenId The token id to withdraw.
    /// @param _amount The amount of token to withdraw.
    /// @param _gasLimit Estimated gas limit required to complete the withdraw on layer 2.
    function _withdrawERC1155(
        address _token,
        address _to,
        uint256 _tokenId,
        uint256 _amount,
        uint256 _gasLimit
    ) internal virtual nonReentrant {
        require(_amount > 0, "withdraw zero amount");

        address _l1Token = tokenMapping[_token];
        require(_l1Token != address(0), "no corresponding l1 token");

        address _sender = _msgSender();

        // 1. burn token
        IScrollERC1155(_token).burn(_sender, _tokenId, _amount);

        // 2. Generate message passed to L1ERC1155Gateway.
        bytes memory _message = abi.encodeCall(
            IL1ERC1155Gateway.finalizeWithdrawERC1155,
            (_l1Token, _token, _sender, _to, _tokenId, _amount)
        );

        // 3. Send message to L2ScrollMessenger.
        IL2ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit);

        emit WithdrawERC1155(_l1Token, _token, _sender, _to, _tokenId, _amount);
    }

    /// @dev Internal function to batch withdraw ERC1155 NFT to layer 2.
    /// @param _token The address of ERC1155 NFT on layer 1.
    /// @param _to The address of recipient on layer 2.
    /// @param _tokenIds The list of token ids to withdraw.
    /// @param _amounts The list of corresponding number of token to withdraw.
    /// @param _gasLimit Estimated gas limit required to complete the withdraw on layer 1.
    function _batchWithdrawERC1155(
        address _token,
        address _to,
        uint256[] calldata _tokenIds,
        uint256[] calldata _amounts,
        uint256 _gasLimit
    ) internal virtual nonReentrant {
        require(_tokenIds.length > 0, "no token to withdraw");
        require(_tokenIds.length == _amounts.length, "length mismatch");

        for (uint256 i = 0; i < _amounts.length; i++) {
            require(_amounts[i] > 0, "withdraw zero amount");
        }

        address _l1Token = tokenMapping[_token];
        require(_l1Token != address(0), "no corresponding l1 token");

        address _sender = _msgSender();

        // 1. transfer token to this contract
        IScrollERC1155(_token).batchBurn(_sender, _tokenIds, _amounts);

        // 2. Generate message passed to L1ERC1155Gateway.
        bytes memory _message = abi.encodeCall(
            IL1ERC1155Gateway.finalizeBatchWithdrawERC1155,
            (_l1Token, _token, _sender, _to, _tokenIds, _amounts)
        );

        // 3. Send message to L2ScrollMessenger.
        IL2ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit);

        emit BatchWithdrawERC1155(_l1Token, _token, _sender, _to, _tokenIds, _amounts);
    }
}
