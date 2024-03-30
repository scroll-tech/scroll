// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {MockERC20} from "solmate/test/utils/mocks/MockERC20.sol";

// solhint-disable no-empty-blocks

contract TransferReentrantToken is MockERC20 {
    address private target;
    uint256 private value;
    bytes private data;
    bool private isBeforeCall;

    constructor(
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    ) MockERC20(_name, _symbol, _decimals) {}

    function setReentrantCall(
        address _target,
        uint256 _value,
        bytes calldata _data,
        bool _isBeforeCall
    ) external payable {
        target = _target;
        value = _value;
        data = _data;
        isBeforeCall = _isBeforeCall;
    }

    function transferFrom(
        address from,
        address to,
        uint256 amount
    ) public virtual override returns (bool) {
        if (isBeforeCall && target != address(0)) {
            // solhint-disable-next-line avoid-low-level-calls
            (bool success, ) = target.call{value: value}(data);
            if (!success) {
                // solhint-disable-next-line no-inline-assembly
                assembly {
                    let ptr := mload(0x40)
                    let size := returndatasize()
                    returndatacopy(ptr, 0, size)
                    revert(ptr, size)
                }
            }
        }

        super.transferFrom(from, to, amount);

        if (!isBeforeCall && target != address(0)) {
            // solhint-disable-next-line avoid-low-level-calls
            (bool success, ) = target.call{value: value}(data);
            if (!success) {
                // solhint-disable-next-line no-inline-assembly
                assembly {
                    let ptr := mload(0x40)
                    let size := returndatasize()
                    returndatacopy(ptr, 0, size)
                    revert(ptr, size)
                }
            }
        }

        return true;
    }
}
