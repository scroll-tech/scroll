// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {ERC2771Context} from "@openzeppelin/contracts/metatx/ERC2771Context.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {IERC20Permit} from "@openzeppelin/contracts/token/ERC20/extensions/draft-IERC20Permit.sol";

import {OwnableBase} from "../libraries/common/OwnableBase.sol";

// solhint-disable no-empty-blocks

contract GasSwap is ERC2771Context, OwnableBase {
    using SafeERC20 for IERC20;

    /**********
     * Events *
     **********/

    /// @notice Emitted when the fee ratio is updated.
    /// @param feeRatio The new fee ratio, multiplied by 1e18.
    event UpdateFeeRatio(uint256 feeRatio);

    /// @notice Emitted when the status of target is updated.
    /// @param target The address of target contract.
    /// @param status The status updated.
    event UpdateApprovedTarget(address target, bool status);

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
    uint256 public feeRatio;

    /***************
     * Constructor *
     ***************/

    constructor(address trustedForwarder) ERC2771Context(trustedForwarder) {
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
        IERC20Permit(_permit.token).permit(
            _msgSender(),
            address(this),
            _permit.value,
            _permit.deadline,
            _permit.v,
            _permit.r,
            _permit.s
        );

        // transfer token
        IERC20(_permit.token).safeTransferFrom(_msgSender(), address(this), _permit.value);

        // approve
        IERC20(_permit.token).safeApprove(_swap.target, 0);
        IERC20(_permit.token).safeApprove(_swap.target, _permit.value);

        // do swap
        uint256 _outputTokenAmount = address(this).balance;
        // solhint-disable-next-line avoid-low-level-calls
        (bool _success, bytes memory _res) = _swap.target.call(_swap.data);
        require(_success, string(concat(bytes("swap failed: "), bytes(getRevertMsg(_res)))));
        _outputTokenAmount = address(this).balance - _outputTokenAmount;

        require(_outputTokenAmount >= _swap.minOutput, "insufficient output amount");

        // take fee and tranfer ETH
        uint256 _fee = (_outputTokenAmount * feeRatio) / PRECISION;
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
            IERC20(_token).safeTransfer(msg.sender, _amount);
        }
    }

    /// @notice Update the fee ratio.
    /// @param _feeRatio The new fee ratio.
    function updateFeeRatio(uint256 _feeRatio) external onlyOwner {
        feeRatio = _feeRatio;

        emit UpdateFeeRatio(_feeRatio);
    }

    /// @notice Update the status of a target address.
    /// @param _target The address of target to update.
    /// @param _status The new status.
    function updateApprovedTarget(address _target, bool _status) external onlyOwner {
        approvedTargets[_target] = _status;

        emit UpdateApprovedTarget(_target, _status);
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
