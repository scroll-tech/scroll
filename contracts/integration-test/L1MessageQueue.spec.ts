/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { expect } from "chai";
import { BigNumber, constants } from "ethers";
import {
  concat,
  getAddress,
  hexlify,
  keccak256,
  randomBytes,
  RLP,
  stripZeros,
  TransactionTypes,
} from "ethers/lib/utils";
import { ethers } from "hardhat";
import { L1MessageQueue, L2GasPriceOracle } from "../typechain";
import { SignerWithAddress } from "@nomiclabs/hardhat-ethers/signers";

describe("L1MessageQueue", async () => {
  let deployer: SignerWithAddress;
  let scrollChain: SignerWithAddress;
  let messenger: SignerWithAddress;
  let gateway: SignerWithAddress;
  let signer: SignerWithAddress;

  let oracle: L2GasPriceOracle;
  let queue: L1MessageQueue;

  beforeEach(async () => {
    [deployer, scrollChain, messenger, gateway, signer] = await ethers.getSigners();

    const L1MessageQueue = await ethers.getContractFactory("L1MessageQueue", deployer);
    queue = await L1MessageQueue.deploy();
    await queue.deployed();

    const L2GasPriceOracle = await ethers.getContractFactory("L2GasPriceOracle", deployer);
    oracle = await L2GasPriceOracle.deploy();

    await oracle.initialize(21000, 0, 8, 16);
    await queue.initialize(messenger.address, scrollChain.address, gateway.address, oracle.address, 10000000);
  });

  context("auth", async () => {
    it("should initialize correctly", async () => {
      expect(await queue.owner()).to.eq(deployer.address);
      expect(await queue.messenger()).to.eq(messenger.address);
      expect(await queue.scrollChain()).to.eq(scrollChain.address);
      expect(await queue.enforcedTxGateway()).to.eq(gateway.address);
      expect(await queue.gasOracle()).to.eq(oracle.address);
      expect(await queue.maxGasLimit()).to.eq(10000000);
    });

    it("should revert, when initialize again", async () => {
      await expect(
        queue.initialize(constants.AddressZero, constants.AddressZero, constants.AddressZero, constants.AddressZero, 0)
      ).to.revertedWith("Initializable: contract is already initialized");
    });

    context("#updateGasOracle", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(queue.connect(signer).updateGasOracle(constants.AddressZero)).to.revertedWith(
          "Ownable: caller is not the owner"
        );
      });

      it("should succeed", async () => {
        expect(await queue.gasOracle()).to.eq(oracle.address);
        await expect(queue.updateGasOracle(deployer.address))
          .to.emit(queue, "UpdateGasOracle")
          .withArgs(oracle.address, deployer.address);
        expect(await queue.gasOracle()).to.eq(deployer.address);
      });
    });

    context("#updateEnforcedTxGateway", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(queue.connect(signer).updateEnforcedTxGateway(constants.AddressZero)).to.revertedWith(
          "Ownable: caller is not the owner"
        );
      });

      it("should succeed", async () => {
        expect(await queue.enforcedTxGateway()).to.eq(gateway.address);
        await expect(queue.updateEnforcedTxGateway(deployer.address))
          .to.emit(queue, "UpdateEnforcedTxGateway")
          .withArgs(gateway.address, deployer.address);
        expect(await queue.enforcedTxGateway()).to.eq(deployer.address);
      });
    });

    context("#updateMaxGasLimit", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(queue.connect(signer).updateMaxGasLimit(0)).to.revertedWith("Ownable: caller is not the owner");
      });

      it("should succeed", async () => {
        expect(await queue.maxGasLimit()).to.eq(10000000);
        await expect(queue.updateMaxGasLimit(0)).to.emit(queue, "UpdateMaxGasLimit").withArgs(10000000, 0);
        expect(await queue.maxGasLimit()).to.eq(0);
      });
    });
  });

  context("#computeTransactionHash", async () => {
    it("should succeed", async () => {
      const sender = "0xb2a70fab1a45b1b9be443b6567849a1702bc1232";
      const target = "0xcb18150e4efefb6786130e289a5f61a82a5b86d7";
      const transactionType = "0x7E";

      for (const nonce of [
        BigNumber.from(0),
        BigNumber.from(1),
        BigNumber.from(127),
        BigNumber.from(128),
        BigNumber.from(22334455),
        constants.MaxUint256,
      ]) {
        for (const value of [
          BigNumber.from(0),
          BigNumber.from(1),
          BigNumber.from(127),
          BigNumber.from(128),
          BigNumber.from(22334455),
          constants.MaxUint256,
        ]) {
          for (const gasLimit of [
            BigNumber.from(0),
            BigNumber.from(1),
            BigNumber.from(127),
            BigNumber.from(128),
            BigNumber.from(22334455),
            constants.MaxUint256,
          ]) {
            for (const dataLen of [0, 1, 2, 3, 4, 55, 56, 100]) {
              const tests = [randomBytes(dataLen)];
              if (dataLen === 1) {
                for (const byte of [0, 1, 127, 128]) {
                  tests.push(Uint8Array.from([byte]));
                }
              }
              for (const data of tests) {
                const transactionPayload = RLP.encode([
                  stripZeros(nonce.toHexString()),
                  stripZeros(gasLimit.toHexString()),
                  target,
                  stripZeros(value.toHexString()),
                  data,
                  sender,
                ]);
                const payload = concat([transactionType, transactionPayload]);
                const expectedHash = keccak256(payload);
                const computedHash = await queue.computeTransactionHash(sender, nonce, value, target, gasLimit, data);
                if (computedHash !== expectedHash) {
                  console.log(hexlify(transactionPayload));
                  console.log(nonce, gasLimit, target, value, data, sender);
                }
                expect(expectedHash).to.eq(computedHash);
              }
            }
          }
        }
      }
    });
  });

  context("#appendCrossDomainMessage", async () => {
    it("should revert, when non-messenger call", async () => {
      await expect(queue.connect(signer).appendCrossDomainMessage(constants.AddressZero, 0, "0x")).to.revertedWith(
        "Only callable by the L1ScrollMessenger"
      );
    });

    it("should revert, when exceed maxGasLimit", async () => {
      await expect(
        queue.connect(messenger).appendCrossDomainMessage(constants.AddressZero, 10000001, "0x")
      ).to.revertedWith("Gas limit must not exceed maxGasLimit");
    });

    it("should revert, when below intrinsic gas", async () => {
      await expect(queue.connect(messenger).appendCrossDomainMessage(constants.AddressZero, 0, "0x")).to.revertedWith(
        "Insufficient gas limit, must be above intrinsic gas"
      );
    });

    it("should succeed", async () => {
      expect(await queue.nextCrossDomainMessageIndex()).to.eq(constants.Zero);
      const sender = getAddress(
        BigNumber.from(messenger.address)
          .add("0x1111000000000000000000000000000000001111")
          .mod(BigNumber.from(2).pow(160))
          .toHexString()
          .slice(2)
          .padStart(40, "0")
      );
      const hash = await queue.computeTransactionHash(sender, 0, 0, signer.address, 100000, "0x01");
      await expect(queue.connect(messenger).appendCrossDomainMessage(signer.address, 100000, "0x01"))
        .to.emit(queue, "QueueTransaction")
        .withArgs(sender, signer.address, 0, 0, 100000, "0x01");
      expect(await queue.nextCrossDomainMessageIndex()).to.eq(constants.One);
      expect(await queue.getCrossDomainMessage(0)).to.eq(hash);
    });
  });

  context("#appendEnforcedTransaction", async () => {
    it("should revert, when non-gateway call", async () => {
      await expect(
        queue.connect(signer).appendEnforcedTransaction(signer.address, constants.AddressZero, 0, 0, "0x")
      ).to.revertedWith("Only callable by the EnforcedTxGateway");
    });

    it("should revert, when sender is not EOA", async () => {
      await expect(
        queue.connect(gateway).appendEnforcedTransaction(queue.address, constants.AddressZero, 0, 0, "0x")
      ).to.revertedWith("only EOA");
    });

    it("should revert, when exceed maxGasLimit", async () => {
      await expect(
        queue.connect(gateway).appendEnforcedTransaction(signer.address, constants.AddressZero, 0, 10000001, "0x")
      ).to.revertedWith("Gas limit must not exceed maxGasLimit");
    });

    it("should revert, when below intrinsic gas", async () => {
      await expect(
        queue.connect(gateway).appendEnforcedTransaction(signer.address, constants.AddressZero, 0, 0, "0x")
      ).to.revertedWith("Insufficient gas limit, must be above intrinsic gas");
    });

    it("should succeed", async () => {
      expect(await queue.nextCrossDomainMessageIndex()).to.eq(constants.Zero);
      const sender = signer.address;
      const hash = await queue.computeTransactionHash(sender, 0, 200, signer.address, 100000, "0x01");
      await expect(
        queue.connect(gateway).appendEnforcedTransaction(signer.address, signer.address, 200, 100000, "0x01")
      )
        .to.emit(queue, "QueueTransaction")
        .withArgs(sender, signer.address, 200, 0, 100000, "0x01");
      expect(await queue.nextCrossDomainMessageIndex()).to.eq(constants.One);
      expect(await queue.getCrossDomainMessage(0)).to.eq(hash);
    });
  });

  context("#popCrossDomainMessage", async () => {
    it("should revert, when non-scrollChain call", async () => {
      await expect(queue.connect(signer).popCrossDomainMessage(0, 0, 0)).to.revertedWith(
        "Only callable by the ScrollChain"
      );
    });

    it("should revert, when pop too many messages", async () => {
      await expect(queue.connect(scrollChain).popCrossDomainMessage(0, 257, 0)).to.revertedWith(
        "pop too many messages"
      );
    });

    it("should revert, when start index mismatch", async () => {
      await expect(queue.connect(scrollChain).popCrossDomainMessage(1, 256, 0)).to.revertedWith("start index mismatch");
    });

    it("should succeed", async () => {
      // append 100 messages
      for (let i = 0; i < 100; i++) {
        await queue.connect(messenger).appendCrossDomainMessage(constants.AddressZero, 1000000, "0x");
      }

      // pop 50 messages with no skip
      await expect(queue.connect(scrollChain).popCrossDomainMessage(0, 50, 0))
        .to.emit(queue, "DequeueTransaction")
        .withArgs(0, 50, 0);
      for (let i = 0; i < 50; i++) {
        expect(await queue.getCrossDomainMessage(i)).to.eq(constants.HashZero);
      }
      expect(await queue.pendingQueueIndex()).to.eq(50);

      // pop 10 messages all skip
      await expect(queue.connect(scrollChain).popCrossDomainMessage(50, 10, 1023))
        .to.emit(queue, "DequeueTransaction")
        .withArgs(50, 10, 1023);
      expect(await queue.pendingQueueIndex()).to.eq(60);
      for (let i = 50; i < 60; i++) {
        expect(BigNumber.from(await queue.getCrossDomainMessage(i))).to.gt(constants.Zero);
      }

      // pop 20 messages, skip first 5
      await expect(queue.connect(scrollChain).popCrossDomainMessage(60, 20, 31))
        .to.emit(queue, "DequeueTransaction")
        .withArgs(60, 20, 31);
      expect(await queue.pendingQueueIndex()).to.eq(80);
      for (let i = 60; i < 65; i++) {
        expect(BigNumber.from(await queue.getCrossDomainMessage(i))).to.gt(constants.Zero);
      }
      for (let i = 65; i < 80; i++) {
        expect(await queue.getCrossDomainMessage(i)).to.eq(constants.HashZero);
      }
    });
  });
});
