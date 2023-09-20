// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {ScrollGatewayBase} from "../../src/libraries/gateway/ScrollGatewayBase.sol";
import {ScrollMessengerBase} from "../../src/libraries/ScrollMessengerBase.sol";

import {ETHRateLimiter} from "../../src/rate-limiter/ETHRateLimiter.sol";
import {TokenRateLimiter} from "../../src/rate-limiter/TokenRateLimiter.sol";

contract DeployL2RateLimiter is Script {
    uint256 L2_DEPLOYER_PRIVATE_KEY = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");

    address L2_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L2_SCROLL_MESSENGER_PROXY_ADDR");

    uint256 RATE_LIMITER_PERIOD_LENGTH = vm.envUint("RATE_LIMITER_PERIOD_LENGTH");
    uint104 ETH_TOTAL_LIMIT = uint104(vm.envUint("ETH_TOTAL_LIMIT"));

    function run() external {
        vm.startBroadcast(L2_DEPLOYER_PRIVATE_KEY);

        deployETHRateLimiter();
        deployTokenRateLimiter();

        vm.stopBroadcast();
    }

    function deployETHRateLimiter() internal {
        ETHRateLimiter limiter = new ETHRateLimiter(
            RATE_LIMITER_PERIOD_LENGTH,
            L2_SCROLL_MESSENGER_PROXY_ADDR,
            ETH_TOTAL_LIMIT
        );

        logAddress("L2_ETH_RATE_LIMITER_ADDR", address(limiter));
    }

    function deployTokenRateLimiter() internal {
        TokenRateLimiter limiter = new TokenRateLimiter(RATE_LIMITER_PERIOD_LENGTH);

        logAddress("L2_TOKEN_RATE_LIMITER_ADDR", address(limiter));
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
