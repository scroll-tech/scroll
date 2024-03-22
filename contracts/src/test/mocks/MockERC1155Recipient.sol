// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {ERC1155TokenReceiver} from "solmate/tokens/ERC1155.sol";

contract MockERC1155Recipient is ERC1155TokenReceiver {
    address private target;
    uint256 private value;
    bytes private data;

    function setCall(
        address _target,
        uint256 _value,
        bytes calldata _data
    ) external payable {
        target = _target;
        value = _value;
        data = _data;
    }

    function onERC1155Received(
        address,
        address,
        uint256,
        uint256,
        bytes calldata
    ) external override returns (bytes4) {
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
        return ERC1155TokenReceiver.onERC1155Received.selector;
    }

    function onERC1155BatchReceived(
        address,
        address,
        uint256[] calldata,
        uint256[] calldata,
        bytes calldata
    ) external override returns (bytes4) {
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
        return ERC1155TokenReceiver.onERC1155BatchReceived.selector;
    }
}
