// SPDX-License-Identifier: AGPL-3.0
pragma solidity 0.8.17;

import "@openzeppelin/contracts/governance/TimelockController.sol";
import "./PolygonZkEVM.sol";

/**
 * @dev Contract module which acts as a timelocked controller.
 * This gives time for users of the controlled contract to exit before a potentially dangerous maintenance operation is applied.
 * If emergency mode of the zkevm contract system is active, this timelock have no delay.
 */
contract PolygonZkEVMTimelock is TimelockController {
    // Polygon ZK-EVM address. Will be used to check if it's on emergency state.
    PolygonZkEVM public immutable polygonZkEVM;

    /**
     * @notice Constructor of timelock
     * @param minDelay initial minimum delay for operations
     * @param proposers accounts to be granted proposer and canceller roles
     * @param executors accounts to be granted executor role
     * @param admin optional account to be granted admin role; disable with zero address
     * @param _polygonZkEVM polygonZkEVM address
     **/
    constructor(
        uint256 minDelay,
        address[] memory proposers,
        address[] memory executors,
        address admin,
        PolygonZkEVM _polygonZkEVM
    ) TimelockController(minDelay, proposers, executors, admin) {
        polygonZkEVM = _polygonZkEVM;
    }

    /**
     * @dev Returns the minimum delay for an operation to become valid.
     *
     * This value can be changed by executing an operation that calls `updateDelay`.
     * If Polygon ZK-EVM is on emergency state the minDelay will be 0 instead.
     */
    function getMinDelay() public view override returns (uint256 duration) {
        if (address(polygonZkEVM) != address(0) && polygonZkEVM.isEmergencyState()) {
            return 0;
        } else {
            return super.getMinDelay();
        }
    }
}
