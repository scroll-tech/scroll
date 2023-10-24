// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;
import "./IBasePolygonZkEVMGlobalExitRoot.sol";

interface IPolygonZkEVMGlobalExitRoot is IBasePolygonZkEVMGlobalExitRoot {
    function getLastGlobalExitRoot() external view returns (bytes32);
}
