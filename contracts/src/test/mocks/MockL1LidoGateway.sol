// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {L1LidoGateway} from "../../lido/L1LidoGateway.sol";

contract MockL1LidoGateway is L1LidoGateway {
    constructor(
        address _l1Token,
        address _l2Token,
        address _counterpart,
        address _router,
        address _messenger
    ) L1LidoGateway(_l1Token, _l2Token, _counterpart, _router, _messenger) {}

    function reentrantCall(address target, bytes calldata data) external payable nonReentrant {
        (bool success, ) = target.call{value: msg.value}(data);
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
}
