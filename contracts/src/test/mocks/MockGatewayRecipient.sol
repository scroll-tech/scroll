// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {IScrollGatewayCallback} from "../../libraries/callbacks/IScrollGatewayCallback.sol";

contract MockGatewayRecipient is IScrollGatewayCallback {
    event ReceiveCall(bytes data);

    function onScrollGatewayCallback(bytes memory data) external {
        emit ReceiveCall(data);
    }

    receive() external payable {}
}
