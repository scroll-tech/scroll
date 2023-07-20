// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

interface IScrollGatewayCallback {
    function onScrollGatewayCallback(bytes memory data) external;
}
