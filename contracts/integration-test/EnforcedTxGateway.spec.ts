/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { expect } from "chai";
import { BigNumberish, BytesLike, constants } from "ethers";
import { ethers } from "hardhat";
import { EnforcedTxGateway, L1MessageQueue, L2GasPriceOracle, MockCaller } from "../typechain";
import { SignerWithAddress } from "@nomiclabs/hardhat-ethers/signers";
import { arrayify } from "ethers/lib/utils";

describe("EnforcedTxGateway.spec", async () => {
  let deployer: SignerWithAddress;
  let feeVault: SignerWithAddress;
  let signer: SignerWithAddress;

  let caller: MockCaller;
  let gateway: EnforcedTxGateway;
  let oracle: L2GasPriceOracle;
  let queue: L1MessageQueue;

  const deployProxy = async (name: string, admin: string): Promise<string> => {
    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);
    const Factory = await ethers.getContractFactory(name, deployer);
    const impl = await Factory.deploy();
    await impl.deployed();
    const proxy = await TransparentUpgradeableProxy.deploy(impl.address, admin, "0x");
    await proxy.deployed();
    return proxy.address;
  };

  beforeEach(async () => {
    [deployer, feeVault, signer] = await ethers.getSigners();

    const ProxyAdmin = await ethers.getContractFactory("ProxyAdmin", deployer);
    const admin = await ProxyAdmin.deploy();
    await admin.deployed();

    queue = await ethers.getContractAt("L1MessageQueue", await deployProxy("L1MessageQueue", admin.address), deployer);

    oracle = await ethers.getContractAt(
      "L2GasPriceOracle",
      await deployProxy("L2GasPriceOracle", admin.address),
      deployer
    );

    gateway = await ethers.getContractAt(
      "EnforcedTxGateway",
      await deployProxy("EnforcedTxGateway", admin.address),
      deployer
    );

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

    it("should revert, when initialize again", async () => {
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

    context("#setPause", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(gateway.connect(signer).setPause(false)).to.revertedWith("Ownable: caller is not the owner");
      });

      it("should succeed", async () => {
        expect(await gateway.paused()).to.eq(false);
        await expect(gateway.setPause(true)).to.emit(gateway, "Paused").withArgs(deployer.address);
        expect(await gateway.paused()).to.eq(true);
        await expect(gateway.setPause(false)).to.emit(gateway, "Unpaused").withArgs(deployer.address);
        expect(await gateway.paused()).to.eq(false);
      });
    });
  });

  context("#sendTransaction, by EOA", async () => {
    it("should revert, when contract is paused", async () => {
      await gateway.setPause(true);
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
    ) => {
      const enforcedTx = {
        sender: signer.address,
        target: target,
        value: value,
        gasLimit: gasLimit,
        data: arrayify(data),
        nonce: await gateway.nonces(signer.address),
        deadline: constants.MaxUint256,
      };

      const domain = {
        name: "EnforcedTxGateway",
        version: "1",
        chainId: (await ethers.provider.getNetwork()).chainId,
        verifyingContract: gateway.address,
      };

      const types = {
        EnforcedTransaction: [
          {
            name: "sender",
            type: "address",
          },
          {
            name: "target",
            type: "address",
          },
          {
            name: "value",
            type: "uint256",
          },
          {
            name: "gasLimit",
            type: "uint256",
          },
          {
            name: "data",
            type: "bytes",
          },
          {
            name: "nonce",
            type: "uint256",
          },
          {
            name: "deadline",
            type: "uint256",
          },
        ],
      };

      const signature = await signer._signTypedData(domain, types, enforcedTx);
      return signature;
    };

    it("should revert, when contract is paused", async () => {
      await gateway.setPause(true);
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,uint256,bytes,address)"](
            signer.address,
            signer.address,
            0,
            0,
            "0x",
            constants.MaxUint256,
            "0x",
            constants.AddressZero
          )
      ).to.revertedWith("Pausable: paused");
    });

    it("should revert, when signature expired", async () => {
      const timestamp = (await ethers.provider.getBlock("latest")).timestamp;
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,uint256,bytes,address)"](
            signer.address,
            signer.address,
            0,
            0,
            "0x",
            timestamp - 1,
            "0x",
            constants.AddressZero
          )
      ).to.revertedWith("signature expired");
    });

    it("should revert, when signature is wrong", async () => {
      const signature = await signer.signMessage("0x00");
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,uint256,bytes,address)"](
            signer.address,
            signer.address,
            0,
            0,
            "0x",
            constants.MaxUint256,
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
          ["sendTransaction(address,address,uint256,uint256,bytes,uint256,bytes,address)"](
            signer.address,
            signer.address,
            0,
            1000000,
            "0x",
            constants.MaxUint256,
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
          ["sendTransaction(address,address,uint256,uint256,bytes,uint256,bytes,address)"](
            signer.address,
            signer.address,
            0,
            1000000,
            "0x",
            constants.MaxUint256,
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
      expect(await gateway.nonces(signer.address)).to.eq(0);
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,uint256,bytes,address)"](
            signer.address,
            deployer.address,
            0,
            1000000,
            "0x",
            constants.MaxUint256,
            signature,
            signer.address,
            { value: fee }
          )
      )
        .to.emit(queue, "QueueTransaction")
        .withArgs(signer.address, deployer.address, 0, 0, 1000000, "0x");
      expect(await gateway.nonces(signer.address)).to.eq(1);
      const feeVaultBalanceAfter = await ethers.provider.getBalance(feeVault.address);
      expect(feeVaultBalanceAfter.sub(feeVaultBalanceBefore)).to.eq(fee);

      // use the same nonce to sign should fail
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,uint256,bytes,address)"](
            signer.address,
            deployer.address,
            0,
            1000000,
            "0x",
            constants.MaxUint256,
            signature,
            signer.address,
            { value: fee }
          )
      ).to.revertedWith("Incorrect signature");
    });

    it("should succeed, with refund", async () => {
      const signature = await getSignature(signer, deployer.address, 0, 1000000, "0x");
      const fee = await queue.estimateCrossDomainMessageFee(1000000);
      const feeVaultBalanceBefore = await ethers.provider.getBalance(feeVault.address);
      const signerBalanceBefore = await ethers.provider.getBalance(signer.address);
      expect(await gateway.nonces(signer.address)).to.eq(0);
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,uint256,bytes,address)"](
            signer.address,
            deployer.address,
            0,
            1000000,
            "0x",
            constants.MaxUint256,
            signature,
            signer.address,
            { value: fee.add(100) }
          )
      )
        .to.emit(queue, "QueueTransaction")
        .withArgs(signer.address, deployer.address, 0, 0, 1000000, "0x");
      expect(await gateway.nonces(signer.address)).to.eq(1);
      const feeVaultBalanceAfter = await ethers.provider.getBalance(feeVault.address);
      const signerBalanceAfter = await ethers.provider.getBalance(signer.address);
      expect(feeVaultBalanceAfter.sub(feeVaultBalanceBefore)).to.eq(fee);
      expect(signerBalanceAfter.sub(signerBalanceBefore)).to.eq(100);

      // use the same nonce to sign should fail
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,uint256,bytes,address)"](
            signer.address,
            deployer.address,
            0,
            1000000,
            "0x",
            constants.MaxUint256,
            signature,
            signer.address,
            { value: fee.add(100) }
          )
      ).to.revertedWith("Incorrect signature");
    });

    it("should revert, when refund failed", async () => {
      const signature = await getSignature(signer, signer.address, 0, 1000000, "0x1234");
      const fee = await queue.estimateCrossDomainMessageFee(1000000);
      await expect(
        gateway
          .connect(deployer)
          ["sendTransaction(address,address,uint256,uint256,bytes,uint256,bytes,address)"](
            signer.address,
            signer.address,
            0,
            1000000,
            "0x1234",
            constants.MaxUint256,
            signature,
            gateway.address,
            { value: fee.add(100) }
          )
      ).to.revertedWith("Failed to refund the fee");
    });
  });
});
