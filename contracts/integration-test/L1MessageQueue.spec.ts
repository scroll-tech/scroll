/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { expect } from "chai";
import { BigNumber, constants } from "ethers";
import { concat, getAddress, hexlify, keccak256, randomBytes, RLP } from "ethers/lib/utils";
import { ethers } from "hardhat";
import { L1MessageQueue, L2GasPriceOracle } from "../typechain";
import { SignerWithAddress } from "@nomiclabs/hardhat-ethers/signers";

describe("L1MessageQueue", async () => {
  let deployer: SignerWithAddress;
  let messenger: SignerWithAddress;
  let gateway: SignerWithAddress;
  let signer: SignerWithAddress;

  let oracle: L2GasPriceOracle;
  let queue: L1MessageQueue;

  beforeEach(async () => {
    [deployer, messenger, gateway, signer] = await ethers.getSigners();

    const L1MessageQueue = await ethers.getContractFactory("L1MessageQueue", deployer);
    queue = await L1MessageQueue.deploy();
    await queue.deployed();

    const L2GasPriceOracle = await ethers.getContractFactory("L2GasPriceOracle", deployer);
    oracle = await L2GasPriceOracle.deploy();

    await oracle.initialize(21000, 0, 8, 16);
    await queue.initialize(messenger.address, gateway.address, oracle.address, 10000000);
  });

  context("auth", async () => {
    it("should initialize correctly", async () => {
      expect(await queue.owner()).to.eq(deployer.address);
      expect(await queue.messenger()).to.eq(messenger.address);
      expect(await queue.enforcedTxGateway()).to.eq(gateway.address);
      expect(await queue.gasOracle()).to.eq(oracle.address);
      expect(await queue.maxGasLimit()).to.eq(10000000);
    });

    it("should revert, when initlaize again", async () => {
      await expect(
        queue.initialize(constants.AddressZero, constants.AddressZero, constants.AddressZero, 0)
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
      const sender = hexlify(randomBytes(20));
      const target = hexlify(randomBytes(20));
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
              const data = randomBytes(dataLen);
              const transactionPayload = RLP.encode([
                nonce.toHexString(),
                gasLimit.toHexString(),
                target,
                value.toHexString(),
                data,
                sender,
              ]);
              const payload = concat([transactionType, transactionPayload]);
              const expectedHash = keccak256(payload);
              const computedHash = await queue.computeTransactionHash(sender, nonce, value, target, gasLimit, data);
              expect(expectedHash).to.eq(computedHash);
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
});
