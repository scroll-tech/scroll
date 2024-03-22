// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

interface IScrollGatewayCallback {
    function onScrollGatewayCallback(bytes memory data) external;
}
