// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {ERC2771ContextUpgradeable} from "@openzeppelin/contracts-upgradeable/metatx/ERC2771ContextUpgradeable.sol";
import {IERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";
import {SafeERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";
import {IERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";
import {IERC20PermitUpgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/extensions/draft-IERC20PermitUpgradeable.sol";

import {OwnableBase} from "../libraries/common/OwnableBase.sol";

// solhint-disable no-empty-blocks

contract GasSwap is ERC2771ContextUpgradeable, OwnableBase {
    using SafeERC20Upgradeable for IERC20Upgradeable;

    /**********
     * Events *
     **********/

    /// @notice Emitted when the fee ratio is updated.
    /// @param fee The new fee ratio, multiplied by 1e18.
    event UpdateFee(uint256 fee);

    /*************
     * Constants *
     *************/

    /// @dev The fee precision.
    uint256 private constant PRECISION = 1e18;

    /***********
     * Structs *
     ***********/

    struct PermitData {
        // The address of token to spend.
        address token;
        // The amount of token to spend.
        uint256 value;
        // The deadline of the permit.
        uint256 deadline;
        // Below three are signatures.
        uint8 v;
        bytes32 r;
        bytes32 s;
    }

    struct SwapData {
        // The address of target contract to call.
        address target;
        // The calldata passed to target contract.
        bytes data;
        // The minimum amount of Ether should receive.
        uint256 minOutput;
    }

    /*************
     * Variables *
     *************/

    /// @notice Keep track whether an address is approved.
    mapping(address => bool) public approvedTargets;

    /// @notice The fee ratio charged for each swap, multiplied by 1e18.
    uint256 public fee;

    /***************
     * Constructor *
     ***************/

    constructor(address trustedForwarder) ERC2771ContextUpgradeable(trustedForwarder) {}

    function initialize() external initializer {
        owner = msg.sender;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    receive() external payable {}

    /// @notice Swap some token for ether.
    /// @param _permit The permit data, see comments from `PermitData`.
    /// @param _swap The swap data, see comments from `SwapData`.
    function swap(PermitData memory _permit, SwapData memory _swap) external {
        require(approvedTargets[_swap.target], "target not approved");

        // do permit
        IERC20PermitUpgradeable(_permit.token).permit(
            _msgSender(),
            address(this),
            _permit.value,
            _permit.deadline,
            _permit.v,
            _permit.r,
            _permit.s
        );

        // transfer token
        IERC20Upgradeable(_permit.token).safeTransferFrom(_msgSender(), address(this), _permit.value);

        // approve
        IERC20Upgradeable(_permit.token).safeApprove(_swap.target, 0);
        IERC20Upgradeable(_permit.token).safeApprove(_swap.target, _permit.value);

        // do swap
        uint256 _outputTokenAmount = address(this).balance;
        // solhint-disable-next-line avoid-low-level-calls
        (bool _success, bytes memory _res) = _swap.target.call(_swap.data);
        require(_success, string(concat(bytes("swap failed: "), bytes(getRevertMsg(_res)))));
        _outputTokenAmount = address(this).balance - _outputTokenAmount;

        require(_outputTokenAmount >= _swap.minOutput, "insufficient output amount");

        // take fee and tranfer ETH
        uint256 _fee = (_outputTokenAmount * fee) / PRECISION;
        (_success, ) = _msgSender().call{value: _outputTokenAmount - _fee}("");
        require(_success, "transfer ETH failed");
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Withdraw stucked tokens.
    /// @param _token The address of token to withdraw. Use `address(0)` if you want to withdraw Ether.
    /// @param _amount The amount of token to withdraw.
    function withdraw(address _token, uint256 _amount) external onlyOwner {
        if (_token == address(0)) {
            (bool success, ) = msg.sender.call{value: _amount}("");
            require(success, "ETH transfer failed");
        } else {
            IERC20Upgradeable(_token).safeTransfer(msg.sender, _amount);
        }
    }

    /// @notice Update the fee ratio.
    /// @param _fee The new fee ratio.
    function updateFee(uint256 _fee) external onlyOwner {
        fee = _fee;

        emit UpdateFee(_fee);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to concat two bytes array.
    function concat(bytes memory a, bytes memory b) internal pure returns (bytes memory) {
        return abi.encodePacked(a, b);
    }

    /// @dev Internal function decode revert message from return data.
    function getRevertMsg(bytes memory _returnData) internal pure returns (string memory) {
        if (_returnData.length < 68) return "Transaction reverted silently";

        // solhint-disable-next-line no-inline-assembly
        assembly {
            _returnData := add(_returnData, 0x04)
        }

        return abi.decode(_returnData, (string));
    }
}
