/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { SignerWithAddress } from "@nomiclabs/hardhat-ethers/signers";
import { expect } from "chai";
import { constants } from "ethers";
import { keccak256 } from "ethers/lib/utils";
import { ethers } from "hardhat";
import { ZKRollup, L1ScrollMessenger, L2ScrollMessenger, L1GatewayRouter, L2GatewayRouter } from "../typechain";

describe("GatewayRouter", async () => {
  const layer1GasLimit = 12345;
  const layer2GasLimit = 54321;
  const DROP_DELAY_DURATION = 86400 * 7;

  let deployer: SignerWithAddress;
  let alice: SignerWithAddress;
  let bob: SignerWithAddress;

  let rollup: ZKRollup;
  let l1Messenger: L1ScrollMessenger;
  let l2Messenger: L2ScrollMessenger;

  beforeEach(async () => {
    [deployer, alice, bob] = await ethers.getSigners();

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

  context("WETHGateway", async () => {
    let l1Gateway: L1GatewayRouter;
    let l2Gateway: L2GatewayRouter;

    beforeEach(async () => {
      // deploy gateway in layer 1
      const L1GatewayRouter = await ethers.getContractFactory("L1GatewayRouter", deployer);
      l1Gateway = await L1GatewayRouter.deploy();

      // deploy gateway in layer 2
      const L2GatewayRouter = await ethers.getContractFactory("L2GatewayRouter", deployer);
      l2Gateway = await L2GatewayRouter.deploy();

      // initialize gateway in layer 1
      await l1Gateway.initialize(constants.AddressZero, l2Gateway.address, l1Messenger.address);

      // initialize gateway in layer 2
      await l2Gateway.initialize(constants.AddressZero, l1Gateway.address, l2Messenger.address);
    });

    const run1to2 = async (sendToSelf: boolean) => {
      context(`layer 1 to layer 2: sendToSelf[${sendToSelf}]`, async () => {
        const amount = ethers.utils.parseEther("100");
        let recipient: SignerWithAddress;

        beforeEach(async () => {
          recipient = sendToSelf ? alice : bob;

          if ((await ethers.provider.getBalance(l2Messenger.address)).eq(constants.Zero)) {
            await deployer.sendTransaction({ to: l2Messenger.address, value: amount });
          }

          expect(await ethers.provider.getBalance(l2Messenger.address)).to.eq(amount);
        });

        it("should transfer to layer 2 without data", async () => {
          // 2. do deposit
          const nonce = await rollup.getQeueuLength();
          const beforeBalanceLayer1 = await ethers.provider.getBalance(l1Messenger.address);
          const depositTx = sendToSelf
            ? await l1Gateway.connect(alice)["depositETH(uint256)"](layer1GasLimit, { value: amount })
            : await l1Gateway
                .connect(alice)
                ["depositETH(address,uint256)"](recipient.address, layer1GasLimit, { value: amount });
          await depositTx.wait();
          const afterBalanceLayer1 = await ethers.provider.getBalance(l1Messenger.address);
          // should emit DepositETH
          await expect(depositTx)
            .to.emit(l1Gateway, "DepositETH")
            .withArgs(alice.address, recipient.address, amount, "0x");
          // should emit SentMessage
          const deadline = (await ethers.provider.getBlock("latest")).timestamp + DROP_DELAY_DURATION;
          const messageData = l2Gateway.interface.encodeFunctionData("finalizeDepositETH", [
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
          const beforeBalanceLayer2 = await ethers.provider.getBalance(recipient.address);
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
          const afterBalanceLayer2 = await ethers.provider.getBalance(recipient.address);
          // should emit RelayedMessage
          await expect(relayTx).to.emit(l2Messenger, "RelayedMessage");
          // should emit FinalizeDepositETH
          await expect(relayTx)
            .to.emit(l2Gateway, "FinalizeDepositETH")
            .withArgs(alice.address, recipient.address, amount, "0x");
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
          await l1Gateway["depositETH(uint256)"](layer1GasLimit, { value: amount });
        });

        it("should transfer to layer 1 without data", async () => {
          // 2. do withdraw in layer 2
          const nonce = await l2Messenger.messageNonce();
          const beforeBalanceLayer2 = await ethers.provider.getBalance(l2Messenger.address);
          const withdrawTx = sendToSelf
            ? await l2Gateway.connect(alice)["withdrawETH(uint256)"](layer2GasLimit, { value: amount })
            : await l2Gateway
                .connect(alice)
                ["withdrawETH(address,uint256)"](recipient.address, layer2GasLimit, { value: amount });
          await withdrawTx.wait();
          const afterBalanceLayer2 = await ethers.provider.getBalance(l2Messenger.address);
          // should emit WithdrawETH
          await expect(withdrawTx)
            .to.emit(l2Gateway, "WithdrawETH")
            .withArgs(alice.address, recipient.address, amount, "0x");
          // should emit SentMessage
          const deadline = (await ethers.provider.getBlock("latest")).timestamp + DROP_DELAY_DURATION;
          const messageData = l1Gateway.interface.encodeFunctionData("finalizeWithdrawETH", [
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
          const beforeBalanceLayer1 = await ethers.provider.getBalance(recipient.address);
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
          const afterBalanceLayer1 = await ethers.provider.getBalance(recipient.address);
          // should emit RelayedMessage
          await expect(relayTx).to.emit(l1Messenger, "RelayedMessage");
          // should emit FinalizeWithdrawETH
          await expect(relayTx)
            .to.emit(l1Gateway, "FinalizeWithdrawETH")
            .withArgs(alice.address, recipient.address, amount, "0x");
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
