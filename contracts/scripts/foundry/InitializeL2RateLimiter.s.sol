// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import {Script} from "forge-std/Script.sol";

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {AccessControlEnumerable} from "@openzeppelin/contracts/access/AccessControlEnumerable.sol";
import {ProxyAdmin} from "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";

import {ScrollMessengerBase} from "../../src/libraries/ScrollMessengerBase.sol";
import {ScrollGatewayBase} from "../../src/libraries/gateway/ScrollGatewayBase.sol";
import {ETHRateLimiter} from "../../src/rate-limiter/ETHRateLimiter.sol";
import {TokenRateLimiter} from "../../src/rate-limiter/TokenRateLimiter.sol";

// solhint-disable max-states-count
// solhint-disable state-visibility
// solhint-disable var-name-mixedcase

contract InitializeL2RateLimiter is Script {
    uint256 L2_DEPLOYER_PRIVATE_KEY = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");

    address L2_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L2_SCROLL_MESSENGER_PROXY_ADDR");
    address L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L2_ETH_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ETH_GATEWAY_PROXY_ADDR");
    address L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L2_DAI_GATEWAY_PROXY_ADDR = vm.envAddress("L2_DAI_GATEWAY_PROXY_ADDR");
    // address L2_USDC_GATEWAY_PROXY_ADDR = vm.envAddress("L2_USDC_GATEWAY_PROXY_ADDR");

    address L2_ETH_RATE_LIMITER_ADDR = vm.envAddress("L2_ETH_RATE_LIMITER_ADDR");
    address L2_TOKEN_RATE_LIMITER_ADDR = vm.envAddress("L2_TOKEN_RATE_LIMITER_ADDR");

    function run() external {
        vm.startBroadcast(L2_DEPLOYER_PRIVATE_KEY);

        ScrollMessengerBase(payable(L2_SCROLL_MESSENGER_PROXY_ADDR)).updateRateLimiter(L2_ETH_RATE_LIMITER_ADDR);

        bytes32 TOKEN_SPENDER_ROLE = TokenRateLimiter(L2_TOKEN_RATE_LIMITER_ADDR).TOKEN_SPENDER_ROLE();
        TokenRateLimiter(L2_TOKEN_RATE_LIMITER_ADDR).grantRole(TOKEN_SPENDER_ROLE, L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR);
        TokenRateLimiter(L2_TOKEN_RATE_LIMITER_ADDR).grantRole(
            TOKEN_SPENDER_ROLE,
            L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR
        );
        TokenRateLimiter(L2_TOKEN_RATE_LIMITER_ADDR).grantRole(TOKEN_SPENDER_ROLE, L2_DAI_GATEWAY_PROXY_ADDR);

        ScrollGatewayBase(payable(L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR)).updateRateLimiter(L2_TOKEN_RATE_LIMITER_ADDR);
        ScrollGatewayBase(payable(L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR)).updateRateLimiter(L2_TOKEN_RATE_LIMITER_ADDR);
        ScrollGatewayBase(payable(L2_DAI_GATEWAY_PROXY_ADDR)).updateRateLimiter(L2_TOKEN_RATE_LIMITER_ADDR);

        // @note comments out for now
        // limiter.grantRole(TOKEN_SPENDER_ROLE, L2_USDC_GATEWAY_PROXY_ADDR);
        // ScrollGatewayBase(payable(L2_USDC_GATEWAY_PROXY_ADDR)).updateRateLimiter(address(limiter));

        vm.stopBroadcast();
    }
}
