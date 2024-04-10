// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {Clones} from "@openzeppelin/contracts/proxy/Clones.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";

import {IScrollStandardERC20Factory} from "./IScrollStandardERC20Factory.sol";

/// @title ScrollStandardERC20Factory
/// @notice The `ScrollStandardERC20Factory` is used to deploy `ScrollStandardERC20` for `L2StandardERC20Gateway`.
/// It uses the `Clones` contract to deploy contract with minimum gas usage.
/// @dev The implementation of deployed token is non-upgradable. This design may be changed in the future.
contract ScrollStandardERC20Factory is Ownable, IScrollStandardERC20Factory {
    /// @notice The address of `ScrollStandardERC20` implementation.
    address public implementation;

    constructor(address _implementation) {
        require(_implementation != address(0), "zero implementation address");

        implementation = _implementation;
    }

    /// @inheritdoc IScrollStandardERC20Factory
    function computeL2TokenAddress(address _gateway, address _l1Token) external view returns (address) {
        // In StandardERC20Gateway, all corresponding l2 tokens are depoyed by Create2 with salt,
        // we can calculate the l2 address directly.
        bytes32 _salt = _getSalt(_gateway, _l1Token);

        return Clones.predictDeterministicAddress(implementation, _salt);
    }

    /// @inheritdoc IScrollStandardERC20Factory
    /// @dev This function should only be called by owner to avoid DDoS attack on StandardTokenBridge.
    function deployL2Token(address _gateway, address _l1Token) external onlyOwner returns (address) {
        bytes32 _salt = _getSalt(_gateway, _l1Token);

        address _l2Token = Clones.cloneDeterministic(implementation, _salt);

        emit DeployToken(_l1Token, _l2Token);

        return _l2Token;
    }

    function _getSalt(address _gateway, address _l1Token) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(_gateway, keccak256(abi.encodePacked(_l1Token))));
    }
}
