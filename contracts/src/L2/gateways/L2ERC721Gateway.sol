// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {IERC721Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC721/IERC721Upgradeable.sol";
import {ERC721HolderUpgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC721/utils/ERC721HolderUpgradeable.sol";

import {IL2ERC721Gateway} from "./IL2ERC721Gateway.sol";
import {IL2ScrollMessenger} from "../IL2ScrollMessenger.sol";
import {IL1ERC721Gateway} from "../../L1/gateways/IL1ERC721Gateway.sol";
import {ScrollGatewayBase, IScrollGateway} from "../../libraries/gateway/ScrollGatewayBase.sol";
import {IScrollERC721} from "../../libraries/token/IScrollERC721.sol";

/// @title L2ERC721Gateway
/// @notice The `L2ERC721Gateway` is used to withdraw ERC721 compatible NFTs on layer 2 and
/// finalize deposit the NFTs from layer 1.
/// @dev The withdrawn NFTs tokens will be burned directly. On finalizing deposit, the corresponding
/// NFT will be minted and transfered to the recipient.
///
/// This will be changed if we have more specific scenarios.
contract L2ERC721Gateway is OwnableUpgradeable, ERC721HolderUpgradeable, ScrollGatewayBase, IL2ERC721Gateway {
    /**********
     * Events *
     **********/

    /// @notice Emitted when token mapping for ERC721 token is updated.
    /// @param _l2Token The address of corresponding ERC721 token on layer 2.
    /// @param _l1Token The address of ERC721 token on layer 1.
    event UpdateTokenMapping(address _l2Token, address _l1Token);

    /*************
     * Variables *
     *************/

    /// @notice Mapping from layer 2 token address to layer 1 token address for ERC721 NFT.
    // solhint-disable-next-line var-name-mixedcase
    mapping(address => address) public tokenMapping;

    /***************
     * Constructor *
     ***************/
    constructor() {
        _disableInitializers();
    }

    function initialize(address _counterpart, address _messenger) external initializer {
        OwnableUpgradeable.__Ownable_init();
        ERC721HolderUpgradeable.__ERC721Holder_init();

        ScrollGatewayBase._initialize(_counterpart, address(0), _messenger);
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL2ERC721Gateway
    function withdrawERC721(
        address _token,
        uint256 _tokenId,
        uint256 _gasLimit
    ) external payable override {
        _withdrawERC721(_token, msg.sender, _tokenId, _gasLimit);
    }

    /// @inheritdoc IL2ERC721Gateway
    function withdrawERC721(
        address _token,
        address _to,
        uint256 _tokenId,
        uint256 _gasLimit
    ) external payable override {
        _withdrawERC721(_token, _to, _tokenId, _gasLimit);
    }

    /// @inheritdoc IL2ERC721Gateway
    function batchWithdrawERC721(
        address _token,
        uint256[] calldata _tokenIds,
        uint256 _gasLimit
    ) external payable override {
        _batchWithdrawERC721(_token, msg.sender, _tokenIds, _gasLimit);
    }

    /// @inheritdoc IL2ERC721Gateway
    function batchWithdrawERC721(
        address _token,
        address _to,
        uint256[] calldata _tokenIds,
        uint256 _gasLimit
    ) external payable override {
        _batchWithdrawERC721(_token, _to, _tokenIds, _gasLimit);
    }

    /// @inheritdoc IL2ERC721Gateway
    function finalizeDepositERC721(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _tokenId
    ) external virtual onlyCallByCounterpart nonReentrant {
        require(_l1Token != address(0), "token address cannot be 0");
        require(_l1Token == tokenMapping[_l2Token], "l2 token mismatch");

        IScrollERC721(_l2Token).mint(_to, _tokenId);

        emit FinalizeDepositERC721(_l1Token, _l2Token, _from, _to, _tokenId);
    }

    /// @inheritdoc IL2ERC721Gateway
    function finalizeBatchDepositERC721(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256[] calldata _tokenIds
    ) external virtual onlyCallByCounterpart nonReentrant {
        require(_l1Token != address(0), "token address cannot be 0");
        require(_l1Token == tokenMapping[_l2Token], "l2 token mismatch");

        for (uint256 i = 0; i < _tokenIds.length; i++) {
            IScrollERC721(_l2Token).mint(_to, _tokenIds[i]);
        }

        emit FinalizeBatchDepositERC721(_l1Token, _l2Token, _from, _to, _tokenIds);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update layer 2 to layer 1 token mapping.
    /// @param _l2Token The address of corresponding ERC721 token on layer 2.
    /// @param _l1Token The address of ERC721 token on layer 1.
    function updateTokenMapping(address _l2Token, address _l1Token) external onlyOwner {
        require(_l1Token != address(0), "token address cannot be 0");

        tokenMapping[_l2Token] = _l1Token;

        emit UpdateTokenMapping(_l2Token, _l1Token);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to withdraw ERC721 NFT to layer 1.
    /// @param _token The address of ERC721 NFT on layer 2.
    /// @param _to The address of recipient on layer 1.
    /// @param _tokenId The token id to withdraw.
    /// @param _gasLimit Estimated gas limit required to complete the withdraw on layer 1.
    function _withdrawERC721(
        address _token,
        address _to,
        uint256 _tokenId,
        uint256 _gasLimit
    ) internal virtual nonReentrant {
        address _l1Token = tokenMapping[_token];
        require(_l1Token != address(0), "no corresponding l1 token");

        // 1. burn token
        // @note in case the token has given too much power to the gateway, we check owner here.
        require(IScrollERC721(_token).ownerOf(_tokenId) == msg.sender, "token not owned");
        IScrollERC721(_token).burn(_tokenId);

        // 2. Generate message passed to L1ERC721Gateway.
        bytes memory _message = abi.encodeCall(
            IL1ERC721Gateway.finalizeWithdrawERC721,
            (_l1Token, _token, msg.sender, _to, _tokenId)
        );

        // 3. Send message to L2ScrollMessenger.
        IL2ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit);

        emit WithdrawERC721(_l1Token, _token, msg.sender, _to, _tokenId);
    }

    /// @dev Internal function to batch withdraw ERC721 NFT to layer 1.
    /// @param _token The address of ERC721 NFT on layer 2.
    /// @param _to The address of recipient on layer 1.
    /// @param _tokenIds The list of token ids to withdraw.
    /// @param _gasLimit Estimated gas limit required to complete the withdraw on layer 1.
    function _batchWithdrawERC721(
        address _token,
        address _to,
        uint256[] calldata _tokenIds,
        uint256 _gasLimit
    ) internal virtual nonReentrant {
        require(_tokenIds.length > 0, "no token to withdraw");

        address _l1Token = tokenMapping[_token];
        require(_l1Token != address(0), "no corresponding l1 token");

        // 1. transfer token to this contract
        for (uint256 i = 0; i < _tokenIds.length; i++) {
            // @note in case the token has given too much power to the gateway, we check owner here.
            require(IScrollERC721(_token).ownerOf(_tokenIds[i]) == msg.sender, "token not owned");
            IScrollERC721(_token).burn(_tokenIds[i]);
        }

        // 2. Generate message passed to L1ERC721Gateway.
        bytes memory _message = abi.encodeCall(
            IL1ERC721Gateway.finalizeBatchWithdrawERC721,
            (_l1Token, _token, msg.sender, _to, _tokenIds)
        );

        // 3. Send message to L2ScrollMessenger.
        IL2ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit);

        emit BatchWithdrawERC721(_l1Token, _token, msg.sender, _to, _tokenIds);
    }
}
