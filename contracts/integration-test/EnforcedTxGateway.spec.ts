/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { expect } from "chai";
import { BigNumberish, BytesLike, constants, utils } from "ethers";
import { ethers } from "hardhat";
import { EnforcedTxGateway, L1MessageQueue, L2GasPriceOracle, MockCaller } from "../typechain";
import { SignerWithAddress } from "@nomiclabs/hardhat-ethers/signers";

describe("EnforcedTxGateway.spec", async () => {
  let deployer: SignerWithAddress;
  let feeVault: SignerWithAddress;
  let signer: SignerWithAddress;

  let caller: MockCaller;
  let gateway: EnforcedTxGateway;
  let oracle: L2GasPriceOracle;
  let queue: L1MessageQueue;

  beforeEach(async () => {
    [deployer, feeVault, signer] = await ethers.getSigners();

    const L1MessageQueue = await ethers.getContractFactory("L1MessageQueue", deployer);
    queue = await L1MessageQueue.deploy();
    await queue.deployed();

    const L2GasPriceOracle = await ethers.getContractFactory("L2GasPriceOracle", deployer);
    oracle = await L2GasPriceOracle.deploy();

    const EnforcedTxGateway = await ethers.getContractFactory("EnforcedTxGateway", deployer);
    gateway = await EnforcedTxGateway.deploy();
    await gateway.deployed();

    const MockCaller = await ethers.getContractFactory("MockCaller", deployer);
    caller = await MockCaller.deploy();
    await caller.deployed();

    await queue.initialize(constants.AddressZero, constants.AddressZero, gateway.address, oracle.address, 10000000);
    await gateway.initialize(queue.address, feeVault.address);
    await oracle.initialize(21000, 0, 8, 16);

    const Whitelist = await ethers.getContractFactory("Whitelist", deployer);
    const whitelist = await Whitelist.deploy(deployer.address);
    await whitelist.deployed();

    await whitelist.updateWhitelistStatus([deployer.address], true);
    await oracle.updateWhitelist(whitelist.address);
    await oracle.setL2BaseFee(1);
  });

  context("auth", async () => {
    it("should initialize correctly", async () => {
      expect(await gateway.owner()).to.eq(deployer.address);
      expect(await gateway.messageQueue()).to.eq(queue.address);
      expect(await gateway.feeVault()).to.eq(feeVault.address);
      expect(await gateway.paused()).to.eq(false);
    });

    it("should revert, when initlaize again", async () => {
      await expect(gateway.initialize(constants.AddressZero, constants.AddressZero)).to.revertedWith(
        "Initializable: contract is already initialized"
      );
    });

    context("#updateFeeVault", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(gateway.connect(signer).updateFeeVault(constants.AddressZero)).to.revertedWith(
          "Ownable: caller is not the owner"
        );
      });

      it("should succeed", async () => {
        expect(await gateway.feeVault()).to.eq(feeVault.address);
        await expect(gateway.updateFeeVault(deployer.address))
          .to.emit(gateway, "UpdateFeeVault")
          .withArgs(feeVault.address, deployer.address);
        expect(await gateway.feeVault()).to.eq(deployer.address);
      });
    });

    context("#setPaused", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(gateway.connect(signer).setPaused(false)).to.revertedWith("Ownable: caller is not the owner");
      });

      it("should succeed", async () => {
        expect(await gateway.paused()).to.eq(false);
        await expect(gateway.setPaused(true)).to.emit(gateway, "Paused").withArgs(deployer.address);
        expect(await gateway.paused()).to.eq(true);
        await expect(gateway.setPaused(false)).to.emit(gateway, "Unpaused").withArgs(deployer.address);
        expect(await gateway.paused()).to.eq(false);
      });
    });
  });

  context("#sendTransaction, by EOA", async () => {
    it("should revert, when contract is paused", async () => {
      await gateway.setPaused(true);
      await expect(
        gateway.connect(signer)["sendTransaction(address,uint256,uint256,bytes)"](signer.address, 0, 0, "0x")
      ).to.revertedWith("Pausable: paused");
    });

    it("should revert, when call is not EOA", async () => {
      const tx = await gateway.populateTransaction["sendTransaction(address,uint256,uint256,bytes)"](
        signer.address,
        0,
        0,
        "0x"
      );
      await expect(caller.callTarget(gateway.address, tx.data!)).to.revertedWith(
        "Only EOA senders are allowed to send enforced transaction"
      );
    });

    it("should revert, when insufficient value for fee", async () => {
      const fee = await queue.estimateCrossDomainMessageFee(1000000);
      await expect(
        gateway
          .connect(signer)
          ["sendTransaction(address,uint256,uint256,bytes)"](signer.address, 0, 1000000, "0x", { value: fee.sub(1) })
      ).to.revertedWith("Insufficient value for fee");
    });

    it("should revert, when failed to deduct the fee", async () => {
      await gateway.updateFeeVault(gateway.address);
      const fee = await queue.estimateCrossDomainMessageFee(1000000);
      await expect(
        gateway
          .connect(signer)
          ["sendTransaction(address,uint256,uint256,bytes)"](signer.address, 0, 1000000, "0x", { value: fee })
      ).to.revertedWith("Failed to deduct the fee");
    });

    it("should succeed, no refund", async () => {
      const fee = await queue.estimateCrossDomainMessageFee(1000000);
      const feeVaultBalanceBefore = await ethers.provider.getBalance(feeVault.address);
      await expect(
        gateway
          .connect(signer)
          ["sendTransaction(address,uint256,uint256,bytes)"](deployer.address, 0, 1000000, "0x", { value: fee })
      )
        .to.emit(queue, "QueueTransaction")
        .withArgs(signer.address, deployer.address, 0, 0, 1000000, "0x");
      const feeVaultBalanceAfter = await ethers.provider.getBalance(feeVault.address);
      expect(feeVaultBalanceAfter.sub(feeVaultBalanceBefore)).to.eq(fee);
    });

    it("should succeed, with refund", async () => {
      const fee = await queue.estimateCrossDomainMessageFee(1000000);
      const feeVaultBalanceBefore = await ethers.provider.getBalance(feeVault.address);
      const signerBalanceBefore = await ethers.provider.getBalance(signer.address);
      const tx = await gateway
        .connect(signer)
        ["sendTransaction(address,uint256,uint256,bytes)"](deployer.address, 0, 1000000, "0x", { value: fee.add(100) });
      await expect(tx)
        .to.emit(queue, "QueueTransaction")
        .withArgs(signer.address, deployer.address, 0, 0, 1000000, "0x");
      const receipt = await tx.wait();
      const feeVaultBalanceAfter = await ethers.provider.getBalance(feeVault.address);
      const signerBalanceAfter = await ethers.provider.getBalance(signer.address);
      expect(feeVaultBalanceAfter.sub(feeVaultBalanceBefore)).to.eq(fee);
      expect(signerBalanceBefore.sub(signerBalanceAfter)).to.eq(
        receipt.gasUsed.mul(receipt.effectiveGasPrice).add(fee)
      );
    });
  });

  context("#sendTransaction, with signatures", async () => {
    const getSignature = async (
      signer: SignerWithAddress,
      target: string,
      value: BigNumberish,
      gasLimit: BigNumberish,
      data: BytesLike
    ): Promise<string> => {
      const queueIndex = await queue.nextCrossDomainMessageIndex();
      const txHash = await queue.computeTransactionHash(signer.address, queueIndex, value, target, gasLimit, data);
      return await signer.signMessage(utils.arrayify(txHash));
    };

    it("should revert, when contract is paused", async () => {
      await gateway.setPaused(true);
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,bytes,address)"](
            signer.address,
            signer.address,
            0,
            0,
            "0x",
            "0x",
            constants.AddressZero
          )
      ).to.revertedWith("Pausable: paused");
    });

    it("should revert, when signatue is wrong", async () => {
      const signature = await signer.signMessage("0x00");
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,bytes,address)"](
            signer.address,
            signer.address,
            0,
            0,
            "0x",
            signature,
            constants.AddressZero
          )
      ).to.revertedWith("Incorrect signature");
    });

    it("should revert, when insufficient value for fee", async () => {
      const signature = await getSignature(signer, signer.address, 0, 1000000, "0x");
      const fee = await queue.estimateCrossDomainMessageFee(1000000);
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,bytes,address)"](
            signer.address,
            signer.address,
            0,
            1000000,
            "0x",
            signature,
            signer.address,
            { value: fee.sub(1) }
          )
      ).to.revertedWith("Insufficient value for fee");
    });

    it("should revert, when failed to deduct the fee", async () => {
      await gateway.updateFeeVault(gateway.address);
      const signature = await getSignature(signer, signer.address, 0, 1000000, "0x");
      const fee = await queue.estimateCrossDomainMessageFee(1000000);
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,bytes,address)"](
            signer.address,
            signer.address,
            0,
            1000000,
            "0x",
            signature,
            signer.address,
            { value: fee }
          )
      ).to.revertedWith("Failed to deduct the fee");
    });

    it("should succeed, no refund", async () => {
      const signature = await getSignature(signer, deployer.address, 0, 1000000, "0x");
      const fee = await queue.estimateCrossDomainMessageFee(1000000);
      const feeVaultBalanceBefore = await ethers.provider.getBalance(feeVault.address);
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,bytes,address)"](
            signer.address,
            deployer.address,
            0,
            1000000,
            "0x",
            signature,
            signer.address,
            { value: fee }
          )
      )
        .to.emit(queue, "QueueTransaction")
        .withArgs(signer.address, deployer.address, 0, 0, 1000000, "0x");
      const feeVaultBalanceAfter = await ethers.provider.getBalance(feeVault.address);
      expect(feeVaultBalanceAfter.sub(feeVaultBalanceBefore)).to.eq(fee);
    });

    it("should succeed, with refund", async () => {
      const signature = await getSignature(signer, deployer.address, 0, 1000000, "0x");
      const fee = await queue.estimateCrossDomainMessageFee(1000000);
      const feeVaultBalanceBefore = await ethers.provider.getBalance(feeVault.address);
      const signerBalanceBefore = await ethers.provider.getBalance(signer.address);
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,bytes,address)"](
            signer.address,
            deployer.address,
            0,
            1000000,
            "0x",
            signature,
            signer.address,
            { value: fee.add(100) }
          )
      )
        .to.emit(queue, "QueueTransaction")
        .withArgs(signer.address, deployer.address, 0, 0, 1000000, "0x");
      const feeVaultBalanceAfter = await ethers.provider.getBalance(feeVault.address);
      const signerBalanceAfter = await ethers.provider.getBalance(signer.address);
      expect(feeVaultBalanceAfter.sub(feeVaultBalanceBefore)).to.eq(fee);
      expect(signerBalanceAfter.sub(signerBalanceBefore)).to.eq(100);
    });

    it("should revert, when refund failed", async () => {
      const signature = await getSignature(signer, signer.address, 0, 1000000, "0x");
      const fee = await queue.estimateCrossDomainMessageFee(1000000);
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,bytes,address)"](
            signer.address,
            signer.address,
            0,
            1000000,
            "0x",
            signature,
            gateway.address,
            { value: fee.add(100) }
          )
      ).to.revertedWith("Failed to refund the fee");
    });
  });
});
