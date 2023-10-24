// SPDX-License-Identifier: AGPL-3.0
pragma solidity 0.8.17;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/Create2.sol";
import "@openzeppelin/contracts/utils/Address.sol";

/**
 * Contract responsible for deploying deterministic address contracts related with the PolygonZkEVM
 */
contract PolygonZkEVMDeployer is Ownable {
    /**
     * @param _owner Owner
     */
    constructor(address _owner) Ownable() {
        _transferOwnership(_owner);
    }

    /**
     * @dev Emitted when a contract is deployed
     */
    event NewDeterministicDeployment(address newContractAddress);

    /**
     * @dev Emitted when a contract is called
     */
    event FunctionCall();

    /**
     * @notice Allows to deploy a contract using create2
     * @param amount Amount used in create2
     * @param salt Salt used in create2
     * @param initBytecode Init bytecode that will be use in create2
     */
    function deployDeterministic(
        uint256 amount,
        bytes32 salt,
        bytes memory initBytecode
    ) public payable onlyOwner {
        address newContractAddress = Create2.deploy(amount, salt, initBytecode);
        emit NewDeterministicDeployment(newContractAddress);
    }

    /**
     * @notice Allows to deploy a contract using create2 and call it afterwards
     * @param amount Amount used in create2
     * @param salt Salt used in create2
     * @param initBytecode Init bytecode that will be use in create2
     * @param dataCall Data used in the call after deploying the smart contract
     */
    function deployDeterministicAndCall(
        uint256 amount,
        bytes32 salt,
        bytes memory initBytecode,
        bytes memory dataCall
    ) public payable onlyOwner {
        address newContractAddress = Create2.deploy(amount, salt, initBytecode);
        Address.functionCall(newContractAddress, dataCall);

        emit NewDeterministicDeployment(newContractAddress);
    }

    /**
     * @param targetAddress Amount of contract deploy
     * @param dataCall Data used to call the target smart contract
     * @param amount Data used to call the target smart contract
     */
    function functionCall(
        address targetAddress,
        bytes memory dataCall,
        uint256 amount
    ) public payable onlyOwner {
        Address.functionCallWithValue(targetAddress, dataCall, amount);

        emit FunctionCall();
    }

    /**
     * @param salt Salt used in create2
     * @param bytecodeHash Init bytecode hashed, it contains the constructor parameters
     */
    function predictDeterministicAddress(
        bytes32 salt,
        bytes32 bytecodeHash
    ) public view returns (address) {
        return Create2.computeAddress(salt, bytecodeHash);
    }
}
