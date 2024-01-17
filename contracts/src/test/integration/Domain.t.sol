// SPDX-License-Identifier: AGPL-3.0-or-later
pragma solidity >=0.8.0;

import {console2} from "forge-std/console2.sol";
import {StdChains} from "forge-std/StdChains.sol";
import {Vm} from "forge-std/Vm.sol";

// code from: https://github.com/marsfoundation/xchain-helpers/blob/master/src/testing/Domain.sol

contract Domain {
    // solhint-disable-next-line const-name-snakecase
    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    StdChains.Chain private _details;
    uint256 public forkId;

    constructor(StdChains.Chain memory _chain) {
        _details = _chain;
        forkId = vm.createFork(_chain.rpcUrl);
        vm.makePersistent(address(this));
    }

    function details() public view returns (StdChains.Chain memory) {
        return _details;
    }

    function selectFork() public {
        vm.selectFork(forkId);
        require(
            block.chainid == _details.chainId,
            string(
                abi.encodePacked(_details.chainAlias, " is pointing to the wrong RPC endpoint '", _details.rpcUrl, "'")
            )
        );
    }

    function rollFork(uint256 blocknum) public {
        vm.rollFork(forkId, blocknum);
    }
}
