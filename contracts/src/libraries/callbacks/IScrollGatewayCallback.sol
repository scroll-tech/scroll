// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IScrollGatewayCallback {
    function onScrollGatewayCallback(bytes memory data) external;
}
