// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {ERC2771Context} from "@openzeppelin/contracts/metatx/ERC2771Context.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import {IERC20Permit} from "@openzeppelin/contracts/token/ERC20/extensions/draft-IERC20Permit.sol";
import {Context} from "@openzeppelin/contracts/utils/Context.sol";

// solhint-disable no-empty-blocks

contract GasSwap is ERC2771Context, Ownable, ReentrancyGuard {
    using SafeERC20 for IERC20;
    using SafeERC20 for IERC20Permit;

    /**********
     * Events *
     **********/

    event UpdateFeeRatio(uint256 feeRatio);
    event UpdateApprovedTarget(address target, bool status);

    /*************
     * Constants *
     *************/

    uint256 private constant PRECISION = 1e18;

    /***********
     * Structs *
     ***********/

    struct PermitData {
        address token;
        uint256 value;
        uint256 deadline;
        uint8 v;
        bytes32 r;
        bytes32 s;
    }

    struct SwapData {
        address target;
        bytes data;
        uint256 minOutput;
    }

    /*************
     * Variables *
     *************/

    mapping(address => bool) public approvedTargets;
    uint256 public feeRatio;

    /***************
     * Constructor *
     ***************/

    constructor(address trustedForwarder) ERC2771Context(trustedForwarder) {}

    /*****************************
     * Public Mutating Functions *
     *****************************/

    receive() external payable {}

    function swap(PermitData memory _permit, SwapData memory _swap) external nonReentrant {
        require(approvedTargets[_swap.target], "target not approved");
        address _sender = _msgSender();

        IERC20Permit(_permit.token).safePermit(
            _sender,
            address(this),
            _permit.value,
            _permit.deadline,
            _permit.v,
            _permit.r,
            _permit.s
        );

        uint256 _balance = IERC20(_permit.token).balanceOf(address(this));

        IERC20(_permit.token).safeTransferFrom(_sender, address(this), _permit.value);

        IERC20(_permit.token).safeApprove(_swap.target, 0);
        IERC20(_permit.token).safeApprove(_swap.target, _permit.value);

        (bool success, bytes memory res) = _swap.target.functionCall(_swap.data, "swap failed");
        require(success, string(abi.encodePacked("swap failed: ", res)));

        uint256 _outputTokenAmount = address(this).balance;
        _outputTokenAmount -= _balance;

        uint256 _fee = (_outputTokenAmount * feeRatio) / PRECISION;
        _outputTokenAmount -= _fee;
        require(_outputTokenAmount >= _swap.minOutput, "insufficient output amount");

        (success, ) = _sender.call{value: _outputTokenAmount}("");
        require(success, "transfer ETH failed");

        uint256 _dust = IERC20(_permit.token).balanceOf(address(this)) - _balance;
        if (_dust > 0) {
            IERC20(_permit.token).safeTransfer(_sender, _dust);
        }
    }

    /************************
     * Restricted Functions *
     ************************/

    function withdraw(address _token, uint256 _amount) external onlyOwner {
        if (_token == address(0)) {
            (bool success, ) = _msgSender().call{value: _amount}("");
            require(success, "ETH transfer failed");
        } else {
            IERC20(_token).safeTransfer(_msgSender(), _amount);
        }
    }

    function updateFeeRatio(uint256 _feeRatio) external onlyOwner {
        feeRatio = _feeRatio;

        emit UpdateFeeRatio(_feeRatio);
    }

    function updateApprovedTarget(address _target, bool _status) external onlyOwner {
        approvedTargets[_target] = _status;

        emit UpdateApprovedTarget(_target, _status);
    }

    /**********************
     * Internal Functions *
     **********************/

    function _msgData() internal view virtual override(Context, ERC2771Context) returns (bytes calldata) {
        return ERC2771Context._msgData();
    }

    function _msgSender() internal view virtual override(Context, ERC2771Context) returns (address) {
        return ERC2771Context._msgSender();
    }
}
