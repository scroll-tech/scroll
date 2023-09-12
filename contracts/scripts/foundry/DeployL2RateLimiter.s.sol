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

    address L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L2_DAI_GATEWAY_PROXY_ADDR = vm.envAddress("L2_DAI_GATEWAY_PROXY_ADDR");
    // address L2_USDC_GATEWAY_PROXY_ADDR = vm.envAddress("L2_USDC_GATEWAY_PROXY_ADDR");

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

        ScrollMessengerBase(payable(L2_SCROLL_MESSENGER_PROXY_ADDR)).updateRateLimiter(address(limiter));
    }

    function deployTokenRateLimiter() internal {
        TokenRateLimiter limiter = new TokenRateLimiter(RATE_LIMITER_PERIOD_LENGTH);

        logAddress("L2_TOKEN_RATE_LIMITER_ADDR", address(limiter));

        bytes32 TOKEN_SPENDER_ROLE = limiter.TOKEN_SPENDER_ROLE();

        limiter.grantRole(TOKEN_SPENDER_ROLE, L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR);
        limiter.grantRole(TOKEN_SPENDER_ROLE, L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR);
        limiter.grantRole(TOKEN_SPENDER_ROLE, L2_DAI_GATEWAY_PROXY_ADDR);

        ScrollGatewayBase(payable(L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR)).updateRateLimiter(address(limiter));
        ScrollGatewayBase(payable(L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR)).updateRateLimiter(address(limiter));
        ScrollGatewayBase(payable(L2_DAI_GATEWAY_PROXY_ADDR)).updateRateLimiter(address(limiter));

        // @note comments out for now
        // limiter.grantRole(TOKEN_SPENDER_ROLE, L2_USDC_GATEWAY_PROXY_ADDR);
        // ScrollGatewayBase(payable(L2_USDC_GATEWAY_PROXY_ADDR)).updateRateLimiter(address(limiter));
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
