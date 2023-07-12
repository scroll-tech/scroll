// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {ERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/ERC20Upgradeable.sol";
import {ERC20PermitUpgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/extensions/draft-ERC20PermitUpgradeable.sol";
import {IScrollStandardERC20} from "./IScrollStandardERC20.sol";
import {IERC677Receiver} from "../callbacks/IERC677Receiver.sol";

contract ScrollStandardERC20 is ERC20PermitUpgradeable, IScrollStandardERC20 {
    /// @inheritdoc IScrollStandardERC20
    address public override gateway;

    /// @inheritdoc IScrollStandardERC20
    address public override counterpart;

    uint8 private decimals_;

    modifier onlyGateway() {
        require(gateway == msg.sender, "Only Gateway");
        _;
    }

    function initialize(
        string memory _name,
        string memory _symbol,
        uint8 _decimals,
        address _gateway,
        address _counterpart
    ) external initializer {
        __ERC20Permit_init(_name);
        __ERC20_init(_name, _symbol);

        decimals_ = _decimals;
        gateway = _gateway;
        counterpart = _counterpart;
    }

    function decimals() public view override returns (uint8) {
        return decimals_;
    }

    /// @dev ERC677 Standard, see https://github.com/ethereum/EIPs/issues/677
    /// Defi can use this method to transfer L1/L2 token to L2/L1,
    /// and deposit to L2/L1 contract in one transaction
    function transferAndCall(
        address receiver,
        uint256 amount,
        bytes calldata data
    ) external returns (bool success) {
        ERC20Upgradeable.transfer(receiver, amount);
        if (isContract(receiver)) {
            contractFallback(receiver, amount, data);
        }
        return true;
    }

    function contractFallback(
        address to,
        uint256 value,
        bytes memory data
    ) private {
        IERC677Receiver receiver = IERC677Receiver(to);
        receiver.onTokenTransfer(msg.sender, value, data);
    }

    function isContract(address _addr) private view returns (bool hasCode) {
        uint256 length;
        // solhint-disable-next-line no-inline-assembly
        assembly {
            length := extcodesize(_addr)
        }
        return length > 0;
    }

    /// @notice Mint some token to recipient's account.
    /// @dev Gateway Utilities, only gateway contract can call
    /// @param _to The address of recipient.
    /// @param _amount The amount of token to mint.
    function mint(address _to, uint256 _amount) external onlyGateway {
        _mint(_to, _amount);
    }

    /// @notice Mint some token from account.
    /// @dev Gateway Utilities, only gateway contract can call
    /// @param _from The address of account to burn token.
    /// @param _amount The amount of token to mint.
    function burn(address _from, uint256 _amount) external onlyGateway {
        _burn(_from, _amount);
    }
}
