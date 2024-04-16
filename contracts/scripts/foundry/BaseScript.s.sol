// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

import "forge-std/Script.sol";

/**
 * @title Base Script
 * @author Puffer Finance
 */
abstract contract BaseScript is Script {
    uint256 internal PK = 1234; // makeAddr("pufferDeployer")

    /**
     * @dev Deployer private key is in `PK` env variable
     */
    uint256 internal _deployerPrivateKey = vm.envOr("PK", PK);
    address internal _broadcaster = vm.addr(_deployerPrivateKey);

    constructor() {
        // For local chain (ANVIL) hardcode the deployer as first account from the blockchain
        if (isAnvil()) {
            // Fist account from ANVIL
            _deployerPrivateKey = uint256(1234);
            _broadcaster = vm.addr(_deployerPrivateKey);
        }
    }

    modifier broadcast() {
        vm.startBroadcast(_deployerPrivateKey);
        _;
        vm.stopBroadcast();
    }

    function isMainnet() internal view returns (bool) {
        return (block.chainid == 1);
    }

    function isAnvil() internal view returns (bool) {
        return (block.chainid == 31337);
    }
}

contract EmptyContract {}
