// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {IERC1155Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC1155/IERC1155Upgradeable.sol";
import {ERC1155HolderUpgradeable, ERC1155ReceiverUpgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC1155/utils/ERC1155HolderUpgradeable.sol";

import {IL2ERC1155Gateway} from "../../L2/gateways/IL2ERC1155Gateway.sol";
import {IL1ScrollMessenger} from "../IL1ScrollMessenger.sol";
import {IL1ERC1155Gateway} from "./IL1ERC1155Gateway.sol";

import {IMessageDropCallback} from "../../libraries/callbacks/IMessageDropCallback.sol";
import {ScrollGatewayBase} from "../../libraries/gateway/ScrollGatewayBase.sol";

/// @title L1ERC1155Gateway
/// @notice The `L1ERC1155Gateway` is used to deposit ERC1155 compatible NFT on layer 1 and
/// finalize withdraw the NFTs from layer 2.
/// @dev The deposited NFTs are held in this gateway. On finalizing withdraw, the corresponding
/// NFT will be transfer to the recipient directly.
///
/// This will be changed if we have more specific scenarios.
contract L1ERC1155Gateway is
    OwnableUpgradeable,
    ERC1155HolderUpgradeable,
    ScrollGatewayBase,
    IL1ERC1155Gateway,
    IMessageDropCallback
{
    /**********
     * Events *
     **********/

    /// @notice Emitted when token mapping for ERC1155 token is updated.
    /// @param _l1Token The address of ERC1155 token on layer 1.
    /// @param _l2Token The address of corresponding ERC1155 token on layer 2.
    event UpdateTokenMapping(address _l1Token, address _l2Token);

    /*************
     * Variables *
     *************/

    /// @notice Mapping from l1 token address to l2 token address for ERC1155 NFT.
    mapping(address => address) public tokenMapping;

    /***************
     * Constructor *
     ***************/

    constructor() {
        _disableInitializers();
    }

    /// @notice Initialize the storage of L1ERC1155Gateway.
    /// @param _counterpart The address of L2ERC1155Gateway in L2.
    /// @param _messenger The address of L1ScrollMessenger.
    function initialize(address _counterpart, address _messenger) external initializer {
        OwnableUpgradeable.__Ownable_init();
        ERC1155HolderUpgradeable.__ERC1155Holder_init();
        ERC1155ReceiverUpgradeable.__ERC1155Receiver_init();

        ScrollGatewayBase._initialize(_counterpart, address(0), _messenger);
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL1ERC1155Gateway
    function depositERC1155(
        address _token,
        uint256 _tokenId,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        _depositERC1155(_token, msg.sender, _tokenId, _amount, _gasLimit);
    }

    /// @inheritdoc IL1ERC1155Gateway
    function depositERC1155(
        address _token,
        address _to,
        uint256 _tokenId,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        _depositERC1155(_token, _to, _tokenId, _amount, _gasLimit);
    }

    /// @inheritdoc IL1ERC1155Gateway
    function batchDepositERC1155(
        address _token,
        uint256[] calldata _tokenIds,
        uint256[] calldata _amounts,
        uint256 _gasLimit
    ) external payable override {
        _batchDepositERC1155(_token, msg.sender, _tokenIds, _amounts, _gasLimit);
    }

    /// @inheritdoc IL1ERC1155Gateway
    function batchDepositERC1155(
        address _token,
        address _to,
        uint256[] calldata _tokenIds,
        uint256[] calldata _amounts,
        uint256 _gasLimit
    ) external payable override {
        _batchDepositERC1155(_token, _to, _tokenIds, _amounts, _gasLimit);
    }

    /// @inheritdoc IL1ERC1155Gateway
    function finalizeWithdrawERC1155(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _tokenId,
        uint256 _amount
    ) external virtual onlyCallByCounterpart nonReentrant {
        require(_l2Token != address(0), "token address cannot be 0");
        require(_l2Token == tokenMapping[_l1Token], "l2 token mismatch");

        IERC1155Upgradeable(_l1Token).safeTransferFrom(address(this), _to, _tokenId, _amount, "");

        emit FinalizeWithdrawERC1155(_l1Token, _l2Token, _from, _to, _tokenId, _amount);
    }

    /// @inheritdoc IL1ERC1155Gateway
    function finalizeBatchWithdrawERC1155(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256[] calldata _tokenIds,
        uint256[] calldata _amounts
    ) external virtual onlyCallByCounterpart nonReentrant {
        require(_l2Token != address(0), "token address cannot be 0");
        require(_l2Token == tokenMapping[_l1Token], "l2 token mismatch");

        IERC1155Upgradeable(_l1Token).safeBatchTransferFrom(address(this), _to, _tokenIds, _amounts, "");

        emit FinalizeBatchWithdrawERC1155(_l1Token, _l2Token, _from, _to, _tokenIds, _amounts);
    }

    /// @inheritdoc IMessageDropCallback
    function onDropMessage(bytes calldata _message) external payable virtual onlyInDropContext nonReentrant {
        require(msg.value == 0, "nonzero msg.value");

        if (bytes4(_message[0:4]) == IL2ERC1155Gateway.finalizeDepositERC1155.selector) {
            (address _token, , address _sender, , uint256 _tokenId, uint256 _amount) = abi.decode(
                _message[4:],
                (address, address, address, address, uint256, uint256)
            );
            IERC1155Upgradeable(_token).safeTransferFrom(address(this), _sender, _tokenId, _amount, "");

            emit RefundERC1155(_token, _sender, _tokenId, _amount);
        } else if (bytes4(_message[0:4]) == IL2ERC1155Gateway.finalizeBatchDepositERC1155.selector) {
            (address _token, , address _sender, , uint256[] memory _tokenIds, uint256[] memory _amounts) = abi.decode(
                _message[4:],
                (address, address, address, address, uint256[], uint256[])
            );
            IERC1155Upgradeable(_token).safeBatchTransferFrom(address(this), _sender, _tokenIds, _amounts, "");

            emit BatchRefundERC1155(_token, _sender, _tokenIds, _amounts);
        } else {
            revert("invalid selector");
        }
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update layer 2 to layer 2 token mapping.
    /// @param _l1Token The address of ERC1155 token on layer 1.
    /// @param _l2Token The address of corresponding ERC1155 token on layer 2.
    function updateTokenMapping(address _l1Token, address _l2Token) external onlyOwner {
        require(_l2Token != address(0), "token address cannot be 0");

        tokenMapping[_l1Token] = _l2Token;

        emit UpdateTokenMapping(_l1Token, _l2Token);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to deposit ERC1155 NFT to layer 2.
    /// @param _token The address of ERC1155 NFT on layer 1.
    /// @param _to The address of recipient on layer 2.
    /// @param _tokenId The token id to deposit.
    /// @param _amount The amount of token to deposit.
    /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
    function _depositERC1155(
        address _token,
        address _to,
        uint256 _tokenId,
        uint256 _amount,
        uint256 _gasLimit
    ) internal virtual nonReentrant {
        require(_amount > 0, "deposit zero amount");

        address _l2Token = tokenMapping[_token];
        require(_l2Token != address(0), "no corresponding l2 token");

        // 1. transfer token to this contract
        IERC1155Upgradeable(_token).safeTransferFrom(msg.sender, address(this), _tokenId, _amount, "");

        // 2. Generate message passed to L2ERC1155Gateway.
        bytes memory _message = abi.encodeCall(
            IL2ERC1155Gateway.finalizeDepositERC1155,
            (_token, _l2Token, msg.sender, _to, _tokenId, _amount)
        );

        // 3. Send message to L1ScrollMessenger.
        IL1ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit, msg.sender);

        emit DepositERC1155(_token, _l2Token, msg.sender, _to, _tokenId, _amount);
    }

    /// @dev Internal function to batch deposit ERC1155 NFT to layer 2.
    /// @param _token The address of ERC1155 NFT on layer 1.
    /// @param _to The address of recipient on layer 2.
    /// @param _tokenIds The list of token ids to deposit.
    /// @param _amounts The list of corresponding number of token to deposit.
    /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
    function _batchDepositERC1155(
        address _token,
        address _to,
        uint256[] calldata _tokenIds,
        uint256[] calldata _amounts,
        uint256 _gasLimit
    ) internal virtual nonReentrant {
        require(_tokenIds.length > 0, "no token to deposit");
        require(_tokenIds.length == _amounts.length, "length mismatch");

        for (uint256 i = 0; i < _amounts.length; i++) {
            require(_amounts[i] > 0, "deposit zero amount");
        }

        address _l2Token = tokenMapping[_token];
        require(_l2Token != address(0), "no corresponding l2 token");

        // 1. transfer token to this contract
        IERC1155Upgradeable(_token).safeBatchTransferFrom(msg.sender, address(this), _tokenIds, _amounts, "");

        // 2. Generate message passed to L2ERC1155Gateway.
        bytes memory _message = abi.encodeCall(
            IL2ERC1155Gateway.finalizeBatchDepositERC1155,
            (_token, _l2Token, msg.sender, _to, _tokenIds, _amounts)
        );

        // 3. Send message to L1ScrollMessenger.
        IL1ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit, msg.sender);

        emit BatchDepositERC1155(_token, _l2Token, msg.sender, _to, _tokenIds, _amounts);
    }
}
