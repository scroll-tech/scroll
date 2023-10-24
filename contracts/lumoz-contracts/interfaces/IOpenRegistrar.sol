// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import "../util/Structs.sol";

interface IOpenRegistrar {
    /**
     * @notice Request to register a new rollup slot
     */
    function request(string calldata _name, address _manager, uint16 _period, uint256 _amount) external payable;

    /**
     * @notice Accept a request
     */
    function accept(uint256 _regId) external;

    /**
     * @notice Reject a request
     */
    function reject(uint256 _regId) external;

    /**
     * @notice Get details of a request
     */
    function getRequest(
        uint256 _regId
    ) external view returns (Request memory req);


    /**
     * @notice Get regId by slotId
     */
    function getRegId(uint256 _slotId) external view returns (uint256);


    /**
     * @notice Get total number of requests
     */
    function totalRequests() external view returns (uint256);

    /**
     * @notice add _registrant to allow
    */
    function addRegistrant(address _registrant) external;
}
