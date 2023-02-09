// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";
import { MockERC20 } from "solmate/test/utils/mocks/MockERC20.sol";

import { L1GatewayTestBase } from "./L1GatewayTestBase.t.sol";

import { L1GatewayRouter } from "../L1/gateways/L1GatewayRouter.sol";
import { IL1ETHGateway, L1ETHGateway } from "../L1/gateways/L1ETHGateway.sol";
import { IL1ScrollMessenger, L1ScrollMessenger } from "../L1/L1ScrollMessenger.sol";
import { IL2ETHGateway, L2ETHGateway } from "../L2/gateways/L2ETHGateway.sol";
import { AddressAliasHelper } from "../libraries/common/AddressAliasHelper.sol";

contract L1ETHGatewayTest is L1GatewayTestBase {
  // from L1ETHGateway
  event DepositETH(address indexed from, address indexed to, uint256 amount, bytes data);
  event FinalizeWithdrawETH(address indexed from, address indexed to, uint256 amount, bytes data);

  L1ETHGateway private gateway;
  L1GatewayRouter private router;

  L2ETHGateway private conterpartGateway;

  function setUp() public {
    setUpBase();

    // Deploy L1 contracts
    gateway = new L1ETHGateway();
    router = new L1GatewayRouter();

    // Deploy L2 contracts
    conterpartGateway = new L2ETHGateway();

    // Initialize L1 contracts
    gateway.initialize(address(conterpartGateway), address(router), address(l1Messenger));
    router.initialize(address(gateway), address(0), address(router), address(l1Messenger));
  }

  function testInitialization() public {
    assertEq(address(conterpartGateway), gateway.counterpart());
    assertEq(address(router), gateway.router());
    assertEq(address(l1Messenger), gateway.messenger());
  }

  function testReinitilize() public {
    hevm.expectRevert("Initializable: contract is already initialized");
    gateway.initialize(address(conterpartGateway), address(router), address(l1Messenger));
  }

  function testDepositETH(
    uint256 amount,
    uint256 gasLimit,
    uint256 feePerGas
  ) public {
    amount = bound(amount, 0, address(this).balance / 2);
    gasLimit = bound(gasLimit, 0, 1000000);
    feePerGas = bound(feePerGas, 0, 1000);

    gasOracle.setL2BaseFee(feePerGas);

    uint256 feeToPay = feePerGas * gasLimit;
    bytes memory message = abi.encodeWithSelector(
      IL2ETHGateway.finalizeDepositETH.selector,
      address(this),
      address(this),
      amount,
      new bytes(0)
    );
    bytes memory xDomainCalldata = abi.encodeWithSignature(
      "relayMessage(address,address,uint256,uint256,bytes)",
      address(gateway),
      address(conterpartGateway),
      amount,
      0,
      message
    );

    if (amount == 0) {
      hevm.expectRevert("deposit zero eth");
      gateway.depositETH{ value: amount }(amount, gasLimit);
    } else {
      // emit QueueTransaction from L1MessageQueue
      {
        hevm.expectEmit(true, true, false, true);
        address sender = AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger));
        emit QueueTransaction(sender, address(l2Messenger), 0, gasLimit, xDomainCalldata, 0);
      }

      // emit SentMessage from L1ScrollMessenger
      {
        hevm.expectEmit(true, true, false, true);
        emit SentMessage(address(gateway), address(conterpartGateway), amount, message, 0);
      }

      // emit DepositETH from L1ETHGateway
      hevm.expectEmit(true, true, false, true);
      emit DepositETH(address(this), address(this), amount, new bytes(0));

      uint256 messengerBalance = address(l1Messenger).balance;
      uint256 feeVaultBalance = address(feeVault).balance;
      gateway.depositETH{ value: amount + feeToPay }(amount, gasLimit);
      assertEq(amount + messengerBalance, address(l1Messenger).balance);
      assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
    }
  }

  function testDepositETHWithRecipient(
    uint256 amount,
    address recipient,
    uint256 gasLimit,
    uint256 feePerGas
  ) public {
    amount = bound(amount, 0, address(this).balance / 2);
    gasLimit = bound(gasLimit, 0, 1000000);
    feePerGas = bound(feePerGas, 0, 1000);

    gasOracle.setL2BaseFee(feePerGas);

    uint256 feeToPay = feePerGas * gasLimit;
    bytes memory message = abi.encodeWithSelector(
      IL2ETHGateway.finalizeDepositETH.selector,
      address(this),
      recipient,
      amount,
      new bytes(0)
    );
    bytes memory xDomainCalldata = abi.encodeWithSignature(
      "relayMessage(address,address,uint256,uint256,bytes)",
      address(gateway),
      address(conterpartGateway),
      amount,
      0,
      message
    );

    if (amount == 0) {
      hevm.expectRevert("deposit zero eth");
      gateway.depositETH{ value: amount }(recipient, amount, gasLimit);
    } else {
      // emit QueueTransaction from L1MessageQueue
      {
        hevm.expectEmit(true, true, false, true);
        address sender = AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger));
        emit QueueTransaction(sender, address(l2Messenger), 0, gasLimit, xDomainCalldata, 0);
      }

      // emit SentMessage from L1ScrollMessenger
      {
        hevm.expectEmit(true, true, false, true);
        emit SentMessage(address(gateway), address(conterpartGateway), amount, message, 0);
      }

      // emit DepositETH from L1ETHGateway
      hevm.expectEmit(true, true, false, true);
      emit DepositETH(address(this), recipient, amount, new bytes(0));

      uint256 messengerBalance = address(l1Messenger).balance;
      uint256 feeVaultBalance = address(feeVault).balance;
      gateway.depositETH{ value: amount + feeToPay }(recipient, amount, gasLimit);
      assertEq(amount + messengerBalance, address(l1Messenger).balance);
      assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
    }
  }

  function testDepositETHWithRecipientAndCalldata(
    uint256 amount,
    address recipient,
    bytes memory dataToCall,
    uint256 gasLimit,
    uint256 feePerGas
  ) public {
    amount = bound(amount, 0, address(this).balance / 2);
    gasLimit = bound(gasLimit, 0, 1000000);
    feePerGas = bound(feePerGas, 0, 1000);

    gasOracle.setL2BaseFee(feePerGas);

    uint256 feeToPay = feePerGas * gasLimit;
    bytes memory message = abi.encodeWithSelector(
      IL2ETHGateway.finalizeDepositETH.selector,
      address(this),
      recipient,
      amount,
      dataToCall
    );
    bytes memory xDomainCalldata = abi.encodeWithSignature(
      "relayMessage(address,address,uint256,uint256,bytes)",
      address(gateway),
      address(conterpartGateway),
      amount,
      0,
      message
    );

    if (amount == 0) {
      hevm.expectRevert("deposit zero eth");
      gateway.depositETHAndCall{ value: amount }(recipient, amount, dataToCall, gasLimit);
    } else {
      // emit QueueTransaction from L1MessageQueue
      {
        hevm.expectEmit(true, true, false, true);
        address sender = AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger));
        emit QueueTransaction(sender, address(l2Messenger), 0, gasLimit, xDomainCalldata, 0);
      }

      // emit SentMessage from L1ScrollMessenger
      {
        hevm.expectEmit(true, true, false, true);
        emit SentMessage(address(gateway), address(conterpartGateway), amount, message, 0);
      }

      // emit DepositETH from L1ETHGateway
      hevm.expectEmit(true, true, false, true);
      emit DepositETH(address(this), recipient, amount, dataToCall);

      uint256 messengerBalance = address(l1Messenger).balance;
      uint256 feeVaultBalance = address(feeVault).balance;
      gateway.depositETHAndCall{ value: amount + feeToPay }(recipient, amount, dataToCall, gasLimit);
      assertEq(amount + messengerBalance, address(l1Messenger).balance);
      assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
    }
  }

  function testFinalizeWithdrawETH(
    address sender,
    address recipient,
    uint256 amount,
    bytes memory dataToCall
  ) public {
    // blacklist some addresses
    hevm.assume(recipient != address(l1Messenger));
    hevm.assume(recipient != address(messageQueue));
    hevm.assume(recipient != address(gasOracle));
    hevm.assume(recipient != address(rollup));
    hevm.assume(recipient != address(feeVault));
    hevm.assume(recipient != address(l2Messenger));
    hevm.assume(recipient != address(gateway));
    hevm.assume(recipient != address(router));
    hevm.assume(recipient != address(conterpartGateway));

    amount = bound(amount, 1, address(this).balance / 2);

    // deposit some ETH to L1ScrollMessenger
    gateway.depositETH{ value: amount }(amount, 0);

    bytes memory message = abi.encodeWithSelector(
      IL1ETHGateway.finalizeWithdrawETH.selector,
      sender,
      recipient,
      amount,
      dataToCall
    );
    bytes memory xDomainCalldata = abi.encodeWithSignature(
      "relayMessage(address,address,uint256,uint256,bytes)",
      address(conterpartGateway),
      address(gateway),
      amount,
      0,
      message
    );

    prepareL2MessageRoot(keccak256(xDomainCalldata));

    IL1ScrollMessenger.L2MessageProof memory proof;
    proof.batchHash = rollup.lastFinalizedBatchHash();

    // emit FinalizeWithdrawETH from L1ETHGateway
    {
      hevm.expectEmit(true, true, false, true);
      emit FinalizeWithdrawETH(sender, recipient, amount, dataToCall);
    }

    // emit RelayedMessage from L1ScrollMessenger
    {
      hevm.expectEmit(true, false, false, true);
      emit RelayedMessage(keccak256(xDomainCalldata));
    }

    uint256 messengerBalance = address(l1Messenger).balance;
    uint256 recipientBalance = recipient.balance;
    l1Messenger.relayMessageWithProof(address(conterpartGateway), address(gateway), amount, 0, message, proof);
    assertEq(messengerBalance - amount, address(l1Messenger).balance);
    assertEq(recipientBalance + amount, recipient.balance);
  }
}
