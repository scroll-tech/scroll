// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import {ScrollConstants} from "./constants/ScrollConstants.sol";
import {IScrollMessenger} from "./IScrollMessenger.sol";

abstract contract ScrollMessengerBase is OwnableUpgradeable, IScrollMessenger {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates fee vault contract.
    /// @param _oldFeeVault The address of old fee vault contract.
    /// @param _newFeeVault The address of new fee vault contract.
    event UpdateFeeVault(address _oldFeeVault, address _newFeeVault);

    /*************
     * Constants *
     *************/

    // https://github.com/OpenZeppelin/openzeppelin-contracts/blob/v4.5.0/contracts/security/ReentrancyGuard.sol
    uint256 internal constant _NOT_ENTERED = 1;
    uint256 internal constant _ENTERED = 2;

    /*************
     * Variables *
     *************/

    /// @notice See {IScrollMessenger-xDomainMessageSender}
    address public override xDomainMessageSender;

    /// @notice The address of counterpart ScrollMessenger contract in L1/L2.
    address public counterpart;

    /// @notice The address of fee vault, collecting cross domain messaging fee.
    address public feeVault;

    // @note move to ScrollMessengerBase in next big refactor
    /// @dev The status of for non-reentrant check.
    uint256 private _lock_status;

    /**********************
     * Function Modifiers *
     **********************/

    modifier nonReentrant() {
        // On the first call to nonReentrant, _notEntered will be true
        require(_lock_status != _ENTERED, "ReentrancyGuard: reentrant call");

        // Any calls to nonReentrant after this point will fail
        _lock_status = _ENTERED;

        _;

        // By storing the original value once again, a refund is triggered (see
        // https://eips.ethereum.org/EIPS/eip-2200)
        _lock_status = _NOT_ENTERED;
    }

    modifier notInExecution() {
        require(
            xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER,
            "Message is already in execution"
        );
        _;
    }

    /***************
     * Constructor *
     ***************/

    function _initialize(address _counterpart, address _feeVault) internal {
        OwnableUpgradeable.__Ownable_init();

        // initialize to a nonzero value
        xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

        counterpart = _counterpart;
        feeVault = _feeVault;
    }

    // allow others to send ether to messenger
    receive() external payable {}

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update fee vault contract.
    /// @dev This function can only called by contract owner.
    /// @param _newFeeVault The address of new fee vault contract.
    function updateFeeVault(address _newFeeVault) external onlyOwner {
        address _oldFeeVault = feeVault;

        feeVault = _newFeeVault;
        emit UpdateFeeVault(_oldFeeVault, _newFeeVault);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to generate the correct cross domain calldata for a message.
    /// @param _sender Message sender address.
    /// @param _target Target contract address.
    /// @param _value The amount of ETH pass to the target.
    /// @param _messageNonce Nonce for the provided message.
    /// @param _message Message to send to the target.
    /// @return ABI encoded cross domain calldata.
    function _encodeXDomainCalldata(
        address _sender,
        address _target,
        uint256 _value,
        uint256 _messageNonce,
        bytes memory _message
    ) internal pure returns (bytes memory) {
        return
            abi.encodeWithSignature(
                "relayMessage(address,address,uint256,uint256,bytes)",
                _sender,
                _target,
                _value,
                _messageNonce,
                _message
            );
    }
}
