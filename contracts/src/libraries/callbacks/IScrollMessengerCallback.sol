// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IScrollMessengerCallback {
    function onScrollMessengerCallback(bytes memory data) external;
}
