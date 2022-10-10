/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { SignerWithAddress } from "@nomiclabs/hardhat-ethers/signers";
import { expect } from "chai";
import { constants } from "ethers";
import { keccak256 } from "ethers/lib/utils";
import { ethers } from "hardhat";
import {
  ZKRollup,
  L1ScrollMessenger,
  L2ScrollMessenger,
  L1StandardERC20Gateway,
  L2StandardERC20Gateway,
  MockERC20,
  ScrollStandardERC20Factory,
  ScrollStandardERC20,
  L1WETHGateway,
  L2WETHGateway,
  WETH9,
} from "../typechain";

describe("ERC20Gateway", async () => {
  const layer1GasLimit = 12345;
  const layer2GasLimit = 54321;
  const DROP_DELAY_DURATION = 86400 * 7;

  let deployer: SignerWithAddress;
  let alice: SignerWithAddress;
  let bob: SignerWithAddress;
  let router: SignerWithAddress;

  let rollup: ZKRollup;
  let l1Messenger: L1ScrollMessenger;
  let l2Messenger: L2ScrollMessenger;

  beforeEach(async () => {
    [deployer, alice, bob, router] = await ethers.getSigners();

    const RollupVerifier = await ethers.getContractFactory("RollupVerifier", deployer);
    const verifier = await RollupVerifier.deploy();
    await verifier.deployed();

    // deploy ZKRollup in layer 1
    const ZKRollup = await ethers.getContractFactory("ZKRollup", {
      signer: deployer,
      libraries: { RollupVerifier: verifier.address },
    });
    rollup = (await ZKRollup.deploy()) as ZKRollup;
    await rollup.initialize(233);
    await rollup.importGenesisBlock({
      blockHash: keccak256(constants.HashZero),
      parentHash: constants.HashZero,
      baseFee: 0,
      stateRoot: constants.HashZero,
      blockHeight: 0,
      gasUsed: 0,
      timestamp: 0,
      extraData: "0x",
    });

    // deploy L1ScrollMessenger in layer 1
    const L1ScrollMessenger = await ethers.getContractFactory("L1ScrollMessenger", deployer);
    l1Messenger = await L1ScrollMessenger.deploy();
    await l1Messenger.initialize(rollup.address);
    await rollup.updateMessenger(l1Messenger.address);

    // deploy L2ScrollMessenger in layer 2
    const L2ScrollMessenger = await ethers.getContractFactory("L2ScrollMessenger", deployer);
    l2Messenger = await L2ScrollMessenger.deploy(deployer.address);
  });

  context("StandardERC20Gateway", async () => {
    let l1Gateway: L1StandardERC20Gateway;
    let l2Gateway: L2StandardERC20Gateway;

    let factory: ScrollStandardERC20Factory;

    beforeEach(async () => {
      // deploy token template in layer 2
      const ScrollStandardERC20 = await ethers.getContractFactory("ScrollStandardERC20", deployer);
      const tokenImpl = await ScrollStandardERC20.deploy();

      // deploy token factory in layer 2
      const ScrollStandardERC20Factory = await ethers.getContractFactory("ScrollStandardERC20Factory", deployer);
      factory = await ScrollStandardERC20Factory.deploy(tokenImpl.address);

      // deploy gateway in layer 1
      const L1StandardERC20Gateway = await ethers.getContractFactory("L1StandardERC20Gateway", deployer);
      l1Gateway = await L1StandardERC20Gateway.deploy();

      // deploy gateway in layer 2
      const L2StandardERC20Gateway = await ethers.getContractFactory("L2StandardERC20Gateway", deployer);
      l2Gateway = await L2StandardERC20Gateway.deploy();

      // initialize gateway in layer 1
      await l1Gateway.initialize(
        l2Gateway.address,
        router.address,
        l1Messenger.address,
        tokenImpl.address,
        factory.address
      );

      // initialize gateway in layer 2
      await l2Gateway.initialize(l1Gateway.address, router.address, l2Messenger.address, factory.address);

      await factory.transferOwnership(l2Gateway.address);
    });

    const run1to2 = async (decimals: number, sendToSelf: boolean) => {
      context(`layer 1 to layer 2: decimals[${decimals}], sendToSelf[${sendToSelf}]`, async () => {
        let l1Token: MockERC20;
        let l2Token: ScrollStandardERC20;
        let recipient: SignerWithAddress;

        const amount1 = ethers.utils.parseUnits("1000", decimals);
        const amount2 = ethers.utils.parseUnits("100", decimals);

        beforeEach(async () => {
          recipient = sendToSelf ? alice : bob;

          // deploy mock token in layer 1
          const MockERC20 = await ethers.getContractFactory("MockERC20", deployer);
          l1Token = await MockERC20.deploy("XYZ", "ZYX", decimals);
          await l1Token.mint(alice.address, amount1.add(amount2));

          // calculate l2 token address
          l2Token = await ethers.getContractAt(
            "ScrollStandardERC20",
            await l2Gateway.getL2ERC20Address(l1Token.address),
            deployer
          );
        });

        it("should succeed, when transfer on the first time", async () => {
          // 1. approve
          await l1Token.connect(alice).approve(l1Gateway.address, amount1);

          // 2. do deposit
          const nonce = await rollup.getQeueuLength();
          const beforeBalanceLayer1 = await l1Token.balanceOf(l1Gateway.address);
          const depositTx = sendToSelf
            ? await l1Gateway
                .connect(alice)
                ["depositERC20(address,uint256,uint256)"](l1Token.address, amount1, layer1GasLimit)
            : await l1Gateway
                .connect(alice)
                ["depositERC20(address,address,uint256,uint256)"](
                  l1Token.address,
                  recipient.address,
                  amount1,
                  layer1GasLimit
                );
          await depositTx.wait();
          const afterBalanceLayer1 = await l1Token.balanceOf(l1Gateway.address);
          // should emit DepositERC20
          await expect(depositTx)
            .to.emit(l1Gateway, "DepositERC20")
            .withArgs(l1Token.address, l2Token.address, alice.address, recipient.address, amount1, "0x");
          // should emit SentMessage
          const symbol = await l1Token.symbol();
          const name = await l1Token.name();
          const deployData = ethers.utils.defaultAbiCoder.encode(
            ["string", "string", "uint8"],
            [symbol, name, decimals]
          );
          const deadline = (await ethers.provider.getBlock("latest")).timestamp + DROP_DELAY_DURATION;
          const messageData = l2Gateway.interface.encodeFunctionData("finalizeDepositERC20", [
            l1Token.address,
            l2Token.address,
            alice.address,
            recipient.address,
            amount1,
            ethers.utils.defaultAbiCoder.encode(["bytes", "bytes"], ["0x", deployData]),
          ]);
          await expect(depositTx)
            .to.emit(l1Messenger, "SentMessage")
            .withArgs(l2Gateway.address, l1Gateway.address, 0, 0, deadline, messageData, nonce, layer1GasLimit);
          // should transfer token in gateway
          expect(afterBalanceLayer1.sub(beforeBalanceLayer1)).to.eq(amount1);

          // 3. do relay in layer 2
          const beforeBalanceLayer2 = constants.Zero;
          const relayTx = await l2Messenger.relayMessage(
            l1Gateway.address,
            l2Gateway.address,
            0,
            0,
            deadline,
            nonce,
            messageData
          );
          await relayTx.wait();
          const afterBalanceLayer2 = await l2Token.balanceOf(recipient.address);
          // should emit RelayedMessage
          await expect(relayTx).to.emit(l2Messenger, "RelayedMessage");
          // should emit FinalizeDepositERC20
          await expect(relayTx)
            .to.emit(l2Gateway, "FinalizeDepositERC20")
            .withArgs(l1Token.address, l2Token.address, alice.address, recipient.address, amount1, "0x");
          // should deploy token in layer 2
          expect(await l2Token.symbol()).to.eq(symbol);
          expect(await l2Token.name()).to.eq(name);
          expect(await l2Token.decimals()).to.eq(decimals);
          // should mint in layer 2
          expect(afterBalanceLayer2.sub(beforeBalanceLayer2)).to.eq(amount1);
        });

        it("should succeed, when transfer on the second time", async () => {
          // 1. approve first time
          await l1Token.connect(alice).approve(l1Gateway.address, amount1);

          // 2. do deposit first time
          const nonce1 = await rollup.getQeueuLength();
          let beforeBalanceLayer1 = await l1Token.balanceOf(l1Gateway.address);
          const depositTx1 = sendToSelf
            ? await l1Gateway
                .connect(alice)
                ["depositERC20(address,uint256,uint256)"](l1Token.address, amount1, layer1GasLimit)
            : await l1Gateway
                .connect(alice)
                ["depositERC20(address,address,uint256,uint256)"](
                  l1Token.address,
                  recipient.address,
                  amount1,
                  layer1GasLimit
                );
          await depositTx1.wait();
          let afterBalanceLayer1 = await l1Token.balanceOf(l1Gateway.address);
          const symbol = await l1Token.symbol();
          const name = await l1Token.name();
          const deployData = ethers.utils.defaultAbiCoder.encode(
            ["string", "string", "uint8"],
            [symbol, name, decimals]
          );
          const deadline1 = (await ethers.provider.getBlock("latest")).timestamp + DROP_DELAY_DURATION;
          const messageData1 = l2Gateway.interface.encodeFunctionData("finalizeDepositERC20", [
            l1Token.address,
            l2Token.address,
            alice.address,
            recipient.address,
            amount1,
            ethers.utils.defaultAbiCoder.encode(["bytes", "bytes"], ["0x", deployData]),
          ]);
          // should transfer token in gateway
          expect(afterBalanceLayer1.sub(beforeBalanceLayer1)).to.eq(amount1);

          // 3. do relay in layer 2 first time
          let beforeBalanceLayer2 = constants.Zero;
          const relayTx1 = await l2Messenger.relayMessage(
            l1Gateway.address,
            l2Gateway.address,
            0,
            0,
            deadline1,
            nonce1,
            messageData1
          );
          await relayTx1.wait();
          let afterBalanceLayer2 = await l2Token.balanceOf(recipient.address);
          expect(afterBalanceLayer2.sub(beforeBalanceLayer2)).to.eq(amount1);

          // 4. approve second time
          await l1Token.connect(alice).approve(l1Gateway.address, amount2);

          // 5. do deposit second time
          const calldata = "0x000033";
          const nonce2 = await rollup.getQeueuLength();
          beforeBalanceLayer1 = await l1Token.balanceOf(l1Gateway.address);
          const depositTx2 = await l1Gateway
            .connect(alice)
            .depositERC20AndCall(l1Token.address, recipient.address, amount2, calldata, layer1GasLimit);
          await depositTx2.wait();
          afterBalanceLayer1 = await l1Token.balanceOf(l1Gateway.address);
          // should emit DepositERC20
          await expect(depositTx2)
            .to.emit(l1Gateway, "DepositERC20")
            .withArgs(l1Token.address, l2Token.address, alice.address, recipient.address, amount2, calldata);
          // should emit SentMessage
          const deadline2 = (await ethers.provider.getBlock("latest")).timestamp + DROP_DELAY_DURATION;
          const messageData2 = l2Gateway.interface.encodeFunctionData("finalizeDepositERC20", [
            l1Token.address,
            l2Token.address,
            alice.address,
            recipient.address,
            amount2,
            calldata,
          ]);
          await expect(depositTx2)
            .to.emit(l1Messenger, "SentMessage")
            .withArgs(l2Gateway.address, l1Gateway.address, 0, 0, deadline2, messageData2, nonce2, layer1GasLimit);
          // should transfer token in gateway
          expect(afterBalanceLayer1.sub(beforeBalanceLayer1)).to.eq(amount2);

          // 3. do relay in layer 2
          beforeBalanceLayer2 = await l2Token.balanceOf(recipient.address);
          const relayTx2 = await l2Messenger.relayMessage(
            l1Gateway.address,
            l2Gateway.address,
            0,
            0,
            deadline2,
            nonce2,
            messageData2
          );
          await relayTx2.wait();
          afterBalanceLayer2 = await l2Token.balanceOf(recipient.address);
          // should emit RelayedMessage
          await expect(relayTx2).to.emit(l2Messenger, "RelayedMessage");
          // should emit FinalizeDepositERC20
          await expect(relayTx2)
            .to.emit(l2Gateway, "FinalizeDepositERC20")
            .withArgs(l1Token.address, l2Token.address, alice.address, recipient.address, amount2, calldata);
          // should mint in layer 2
          expect(afterBalanceLayer2.sub(beforeBalanceLayer2)).to.eq(amount2);
        });
      });
    };

    const run2to1 = async (decimals: number, sendToSelf: boolean) => {
      context(`layer 2 to layer 1: decimals[${decimals}], sendToSelf[${sendToSelf}]`, async () => {
        let l1Token: MockERC20;
        let l2Token: ScrollStandardERC20;
        let recipient: SignerWithAddress;

        const amount = ethers.utils.parseUnits("1000", decimals);

        beforeEach(async () => {
          recipient = sendToSelf ? alice : bob;

          // deploy mock token in layer 1
          const MockERC20 = await ethers.getContractFactory("MockERC20", deployer);
          l1Token = await MockERC20.deploy("XYZ", "ZYX", decimals);
          await l1Token.mint(alice.address, amount);

          // calculate l2 token address
          l2Token = await ethers.getContractAt(
            "ScrollStandardERC20",
            await l2Gateway.getL2ERC20Address(l1Token.address),
            deployer
          );

          await l1Token.connect(alice).approve(l1Gateway.address, constants.MaxUint256);
          const depositTx = await l1Gateway
            .connect(alice)
            ["depositERC20(address,uint256,uint256)"](l1Token.address, amount, layer1GasLimit);
          await depositTx.wait();
          const symbol = await l1Token.symbol();
          const name = await l1Token.name();
          const deployData = ethers.utils.defaultAbiCoder.encode(
            ["string", "string", "uint8"],
            [symbol, name, decimals]
          );
          const deadline = (await ethers.provider.getBlock("latest")).timestamp + DROP_DELAY_DURATION;
          const nonce = await rollup.getQeueuLength();
          const messageData = l2Gateway.interface.encodeFunctionData("finalizeDepositERC20", [
            l1Token.address,
            l2Token.address,
            alice.address,
            alice.address,
            amount,
            ethers.utils.defaultAbiCoder.encode(["bytes", "bytes"], ["0x", deployData]),
          ]);
          const relayTx = await l2Messenger.relayMessage(
            l1Gateway.address,
            l2Gateway.address,
            0,
            0,
            deadline,
            nonce,
            messageData
          );
          await relayTx.wait();

          expect(await l2Token.balanceOf(alice.address)).to.eq(amount);
        });

        it("should succeed, when transfer without data", async () => {
          // 1. approve
          await l2Token.connect(alice).approve(l2Gateway.address, amount);

          // 2. withdraw
          const nonce = await l2Messenger.messageNonce();
          const balanceBefore = await l2Token.balanceOf(alice.address);
          const withdrawTx = sendToSelf
            ? await l2Gateway
                .connect(alice)
                ["withdrawERC20(address,uint256,uint256)"](l2Token.address, amount, layer2GasLimit)
            : await l2Gateway
                .connect(alice)
                ["withdrawERC20(address,address,uint256,uint256)"](
                  l2Token.address,
                  recipient.address,
                  amount,
                  layer2GasLimit
                );
          await withdrawTx.wait();
          const deadline = (await ethers.provider.getBlock("latest")).timestamp + DROP_DELAY_DURATION;
          const balanceAfter = await l2Token.balanceOf(alice.address);
          // should emit WithdrawERC20
          await expect(withdrawTx)
            .to.emit(l2Gateway, "WithdrawERC20")
            .withArgs(l1Token.address, l2Token.address, alice.address, recipient.address, amount, "0x");
          // should emit SentMessage
          const messageData = l1Gateway.interface.encodeFunctionData("finalizeWithdrawERC20", [
            l1Token.address,
            l2Token.address,
            alice.address,
            recipient.address,
            amount,
            "0x",
          ]);
          await expect(withdrawTx)
            .to.emit(l2Messenger, "SentMessage")
            .withArgs(l1Gateway.address, l2Gateway.address, 0, 0, deadline, messageData, nonce, layer2GasLimit);
          // should transfer from alice
          expect(balanceBefore.sub(balanceAfter)).to.eq(amount);

          // 3. relay in layer 1
          const relayTx = await l1Messenger.relayMessageWithProof(
            l2Gateway.address,
            l1Gateway.address,
            0,
            0,
            deadline,
            nonce,
            messageData,
            { blockNumber: 0, merkleProof: "0x" }
          );
          await relayTx.wait();
          // should emit RelayedMessage
          await expect(relayTx).to.emit(l1Messenger, "RelayedMessage");
          // should emit FinalizeWithdrawERC20
          await expect(relayTx)
            .to.emit(l1Gateway, "FinalizeWithdrawERC20")
            .withArgs(l1Token.address, l2Token.address, alice.address, recipient.address, amount, "0x");
          // should transfer out
          expect(await l1Token.balanceOf(l1Gateway.address)).to.eq(0);
          expect(await l1Token.balanceOf(recipient.address)).to.eq(amount);
        });

        it("should succeed, when transfer with data", async () => {
          const calldata = "0x3d4233433232";
          // 1. approve
          await l2Token.connect(alice).approve(l2Gateway.address, amount);

          // 2. withdraw
          const nonce = await l2Messenger.messageNonce();
          const withdrawTx = await l2Gateway
            .connect(alice)
            .withdrawERC20AndCall(l2Token.address, recipient.address, amount, calldata, layer2GasLimit);
          await withdrawTx.wait();
          const deadline = (await ethers.provider.getBlock("latest")).timestamp + DROP_DELAY_DURATION;
          // should emit WithdrawERC20
          await expect(withdrawTx)
            .to.emit(l2Gateway, "WithdrawERC20")
            .withArgs(l1Token.address, l2Token.address, alice.address, recipient.address, amount, calldata);
          // should emit SentMessage
          const messageData = l1Gateway.interface.encodeFunctionData("finalizeWithdrawERC20", [
            l1Token.address,
            l2Token.address,
            alice.address,
            recipient.address,
            amount,
            calldata,
          ]);
          await expect(withdrawTx)
            .to.emit(l2Messenger, "SentMessage")
            .withArgs(l1Gateway.address, l2Gateway.address, 0, 0, deadline, messageData, nonce, layer2GasLimit);

          // 3. relay in layer 1
          const relayTx = await l1Messenger.relayMessageWithProof(
            l2Gateway.address,
            l1Gateway.address,
            0,
            0,
            deadline,
            nonce,
            messageData,
            { blockNumber: 0, merkleProof: "0x" }
          );
          await relayTx.wait();
          // should emit RelayedMessage
          await expect(relayTx).to.emit(l1Messenger, "RelayedMessage");
          // should emit FinalizeWithdrawERC20
          await expect(relayTx)
            .to.emit(l1Gateway, "FinalizeWithdrawERC20")
            .withArgs(l1Token.address, l2Token.address, alice.address, recipient.address, amount, calldata);
          // should transfer out
          expect(await l1Token.balanceOf(l1Gateway.address)).to.eq(0);
          expect(await l1Token.balanceOf(recipient.address)).to.eq(amount);
        });
      });
    };

    for (const decimals of [6, 18, 24]) {
      for (const sendToSelf of [true, false]) {
        run1to2(decimals, sendToSelf);
        run2to1(decimals, sendToSelf);
      }
    }
  });

  context("WETHGateway", async () => {
    let l1Gateway: L1WETHGateway;
    let l2Gateway: L2WETHGateway;
    let l1WETH: WETH9;
    let l2WETH: WETH9;

    beforeEach(async () => {
      // deploy weth in layer 1 and layer 2
      const WETH9 = await ethers.getContractFactory("WETH9", deployer);
      l1WETH = await WETH9.deploy();
      l2WETH = await WETH9.deploy();

      // deploy gateway in layer 1
      const L1WETHGateway = await ethers.getContractFactory("L1WETHGateway", deployer);
      l1Gateway = await L1WETHGateway.deploy();

      // deploy gateway in layer 2
      const L2WETHGateway = await ethers.getContractFactory("L2WETHGateway", deployer);
      l2Gateway = await L2WETHGateway.deploy();

      // initialize gateway in layer 1
      await l1Gateway.initialize(
        l2Gateway.address,
        router.address,
        l1Messenger.address,
        l1WETH.address,
        l2WETH.address
      );

      // initialize gateway in layer 2
      await l2Gateway.initialize(
        l1Gateway.address,
        router.address,
        l2Messenger.address,
        l2WETH.address,
        l1WETH.address
      );
    });

    const run1to2 = async (sendToSelf: boolean) => {
      context(`layer 1 to layer 2: sendToSelf[${sendToSelf}]`, async () => {
        const amount = ethers.utils.parseEther("100");
        let recipient: SignerWithAddress;

        beforeEach(async () => {
          recipient = sendToSelf ? alice : bob;

          if ((await ethers.provider.getBalance(l2Messenger.address)).eq(constants.Zero)) {
            await deployer.sendTransaction({ to: l2Messenger.address, value: amount });
            await l1WETH.connect(alice).deposit({ value: amount });
          }

          expect(await ethers.provider.getBalance(l2Messenger.address)).to.eq(amount);
        });

        it("should transfer to layer 2 without data", async () => {
          // 1. deposit and approve
          await l1WETH.connect(alice).approve(l1Gateway.address, amount);

          // 2. do deposit
          const nonce = await rollup.getQeueuLength();
          const beforeBalanceLayer1 = await ethers.provider.getBalance(l1Messenger.address);
          const depositTx = sendToSelf
            ? await l1Gateway
                .connect(alice)
                ["depositERC20(address,uint256,uint256)"](l1WETH.address, amount, layer1GasLimit)
            : await l1Gateway
                .connect(alice)
                ["depositERC20(address,address,uint256,uint256)"](
                  l1WETH.address,
                  recipient.address,
                  amount,
                  layer1GasLimit
                );
          await depositTx.wait();
          const afterBalanceLayer1 = await ethers.provider.getBalance(l1Messenger.address);
          // should emit DepositERC20
          await expect(depositTx)
            .to.emit(l1Gateway, "DepositERC20")
            .withArgs(l1WETH.address, l2WETH.address, alice.address, recipient.address, amount, "0x");
          // should emit SentMessage
          const deadline = (await ethers.provider.getBlock("latest")).timestamp + DROP_DELAY_DURATION;
          const messageData = l2Gateway.interface.encodeFunctionData("finalizeDepositERC20", [
            l1WETH.address,
            l2WETH.address,
            alice.address,
            recipient.address,
            amount,
            "0x",
          ]);
          await expect(depositTx)
            .to.emit(l1Messenger, "SentMessage")
            .withArgs(l2Gateway.address, l1Gateway.address, amount, 0, deadline, messageData, nonce, layer1GasLimit);
          // should unwrap transfer to messenger
          expect(afterBalanceLayer1.sub(beforeBalanceLayer1)).to.eq(amount);

          // 3. do relay in layer 2
          const beforeBalanceLayer2 = await l2WETH.balanceOf(recipient.address);
          const relayTx = await l2Messenger.relayMessage(
            l1Gateway.address,
            l2Gateway.address,
            amount,
            0,
            deadline,
            nonce,
            messageData
          );
          await relayTx.wait();
          const afterBalanceLayer2 = await l2WETH.balanceOf(recipient.address);
          // should emit RelayedMessage
          await expect(relayTx).to.emit(l2Messenger, "RelayedMessage");
          // should emit FinalizeDepositERC20
          await expect(relayTx)
            .to.emit(l2Gateway, "FinalizeDepositERC20")
            .withArgs(l1WETH.address, l2WETH.address, alice.address, recipient.address, amount, "0x");
          // should transfer and wrap weth in layer 2
          expect(afterBalanceLayer2.sub(beforeBalanceLayer2)).to.eq(amount);
          expect(await ethers.provider.getBalance(l2Messenger.address)).to.eq(constants.Zero);
        });

        it("should transfer to layer 2 data", async () => {
          const calldata = "0x3333444555fdad";
          // 1. deposit and approve
          await l1WETH.connect(alice).approve(l1Gateway.address, amount);

          // 2. do deposit
          const nonce = await rollup.getQeueuLength();
          const beforeBalanceLayer1 = await ethers.provider.getBalance(l1Messenger.address);
          const depositTx = await l1Gateway
            .connect(alice)
            .depositERC20AndCall(l1WETH.address, recipient.address, amount, calldata, layer1GasLimit);
          await depositTx.wait();
          const afterBalanceLayer1 = await ethers.provider.getBalance(l1Messenger.address);
          // should emit DepositERC20
          await expect(depositTx)
            .to.emit(l1Gateway, "DepositERC20")
            .withArgs(l1WETH.address, l2WETH.address, alice.address, recipient.address, amount, calldata);
          // should emit SentMessage
          const deadline = (await ethers.provider.getBlock("latest")).timestamp + DROP_DELAY_DURATION;
          const messageData = l2Gateway.interface.encodeFunctionData("finalizeDepositERC20", [
            l1WETH.address,
            l2WETH.address,
            alice.address,
            recipient.address,
            amount,
            calldata,
          ]);
          await expect(depositTx)
            .to.emit(l1Messenger, "SentMessage")
            .withArgs(l2Gateway.address, l1Gateway.address, amount, 0, deadline, messageData, nonce, layer1GasLimit);
          // should unwrap transfer to messenger
          expect(afterBalanceLayer1.sub(beforeBalanceLayer1)).to.eq(amount);

          // 3. do relay in layer 2
          const beforeBalanceLayer2 = await l2WETH.balanceOf(recipient.address);
          const relayTx = await l2Messenger.relayMessage(
            l1Gateway.address,
            l2Gateway.address,
            amount,
            0,
            deadline,
            nonce,
            messageData
          );
          await relayTx.wait();
          const afterBalanceLayer2 = await l2WETH.balanceOf(recipient.address);
          // should emit RelayedMessage
          await expect(relayTx).to.emit(l2Messenger, "RelayedMessage");
          // should emit FinalizeDepositERC20
          await expect(relayTx)
            .to.emit(l2Gateway, "FinalizeDepositERC20")
            .withArgs(l1WETH.address, l2WETH.address, alice.address, recipient.address, amount, calldata);
          // should transfer and wrap weth in layer 2
          expect(afterBalanceLayer2.sub(beforeBalanceLayer2)).to.eq(amount);
          expect(await ethers.provider.getBalance(l2Messenger.address)).to.eq(constants.Zero);
        });
      });
    };

    const run2to1 = async (sendToSelf: boolean) => {
      context(`layer 2 to layer 1: sendToSelf[${sendToSelf}]`, async () => {
        const amount = ethers.utils.parseEther("100");
        let recipient: SignerWithAddress;

        beforeEach(async () => {
          recipient = sendToSelf ? alice : bob;
          await l1WETH.connect(alice).deposit({ value: amount });
          await l1WETH.connect(alice).approve(l1Gateway.address, amount);
          await l1Gateway.connect(alice)["depositERC20(address,uint256,uint256)"](l1WETH.address, amount, 0);
          await l2WETH.connect(alice).deposit({ value: amount });
        });

        it("should transfer to layer 1 without data", async () => {
          // 1. approve
          await l2WETH.connect(alice).approve(l2Gateway.address, amount);

          // 2. do withdraw in layer 2
          const nonce = await l2Messenger.messageNonce();
          const beforeBalanceLayer2 = await ethers.provider.getBalance(l2Messenger.address);
          const withdrawTx = sendToSelf
            ? await l2Gateway
                .connect(alice)
                ["withdrawERC20(address,uint256,uint256)"](l2WETH.address, amount, layer2GasLimit)
            : await l2Gateway
                .connect(alice)
                ["withdrawERC20(address,address,uint256,uint256)"](
                  l2WETH.address,
                  recipient.address,
                  amount,
                  layer2GasLimit
                );
          await withdrawTx.wait();
          const afterBalanceLayer2 = await ethers.provider.getBalance(l2Messenger.address);
          // should emit WithdrawERC20
          await expect(withdrawTx)
            .to.emit(l2Gateway, "WithdrawERC20")
            .withArgs(l1WETH.address, l2WETH.address, alice.address, recipient.address, amount, "0x");
          // should emit SentMessage
          const deadline = (await ethers.provider.getBlock("latest")).timestamp + DROP_DELAY_DURATION;
          const messageData = l1Gateway.interface.encodeFunctionData("finalizeWithdrawERC20", [
            l1WETH.address,
            l2WETH.address,
            alice.address,
            recipient.address,
            amount,
            "0x",
          ]);
          await expect(withdrawTx)
            .to.emit(l2Messenger, "SentMessage")
            .withArgs(l1Gateway.address, l2Gateway.address, amount, 0, deadline, messageData, nonce, layer2GasLimit);
          // should unwrap transfer to messenger
          expect(afterBalanceLayer2.sub(beforeBalanceLayer2)).to.eq(amount);

          // 3. do relay in layer 1
          const beforeBalanceLayer1 = await l1WETH.balanceOf(recipient.address);
          const relayTx = await l1Messenger.relayMessageWithProof(
            l2Gateway.address,
            l1Gateway.address,
            amount,
            0,
            deadline,
            nonce,
            messageData,
            { blockNumber: 0, merkleProof: "0x" }
          );
          await relayTx.wait();
          const afterBalanceLayer1 = await l1WETH.balanceOf(recipient.address);
          // should emit RelayedMessage
          await expect(relayTx).to.emit(l1Messenger, "RelayedMessage");
          // should emit FinalizeWithdrawERC20
          await expect(relayTx)
            .to.emit(l1Gateway, "FinalizeWithdrawERC20")
            .withArgs(l1WETH.address, l2WETH.address, alice.address, recipient.address, amount, "0x");
          // should transfer and wrap weth in layer 1
          expect(afterBalanceLayer1.sub(beforeBalanceLayer1)).to.eq(amount);
        });

        it("should transfer to layer 1 with data", async () => {
          const calldata = "0x33445566778899";
          // 1. approve
          await l2WETH.connect(alice).approve(l2Gateway.address, amount);

          // 2. do withdraw in layer 2
          const nonce = await l2Messenger.messageNonce();
          const beforeBalanceLayer2 = await ethers.provider.getBalance(l2Messenger.address);
          const withdrawTx = await l2Gateway
            .connect(alice)
            .withdrawERC20AndCall(l2WETH.address, recipient.address, amount, calldata, layer2GasLimit);
          await withdrawTx.wait();
          const afterBalanceLayer2 = await ethers.provider.getBalance(l2Messenger.address);
          // should emit WithdrawERC20
          await expect(withdrawTx)
            .to.emit(l2Gateway, "WithdrawERC20")
            .withArgs(l1WETH.address, l2WETH.address, alice.address, recipient.address, amount, calldata);
          // should emit SentMessage
          const deadline = (await ethers.provider.getBlock("latest")).timestamp + DROP_DELAY_DURATION;
          const messageData = l1Gateway.interface.encodeFunctionData("finalizeWithdrawERC20", [
            l1WETH.address,
            l2WETH.address,
            alice.address,
            recipient.address,
            amount,
            calldata,
          ]);
          await expect(withdrawTx)
            .to.emit(l2Messenger, "SentMessage")
            .withArgs(l1Gateway.address, l2Gateway.address, amount, 0, deadline, messageData, nonce, layer2GasLimit);
          // should unwrap transfer to messenger
          expect(afterBalanceLayer2.sub(beforeBalanceLayer2)).to.eq(amount);

          // 3. do relay in layer 1
          const beforeBalanceLayer1 = await l1WETH.balanceOf(recipient.address);
          const relayTx = await l1Messenger.relayMessageWithProof(
            l2Gateway.address,
            l1Gateway.address,
            amount,
            0,
            deadline,
            nonce,
            messageData,
            { blockNumber: 0, merkleProof: "0x" }
          );
          await relayTx.wait();
          const afterBalanceLayer1 = await l1WETH.balanceOf(recipient.address);
          // should emit RelayedMessage
          await expect(relayTx).to.emit(l1Messenger, "RelayedMessage");
          // should emit FinalizeWithdrawERC20
          await expect(relayTx)
            .to.emit(l1Gateway, "FinalizeWithdrawERC20")
            .withArgs(l1WETH.address, l2WETH.address, alice.address, recipient.address, amount, calldata);
          // should transfer and wrap weth in layer 1
          expect(afterBalanceLayer1.sub(beforeBalanceLayer1)).to.eq(amount);
        });
      });
    };

    for (const sendToSelf of [true, false]) {
      run1to2(sendToSelf);
      run2to1(sendToSelf);
    }
  });
});
