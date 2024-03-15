/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { ZeroAddress } from "ethers";
import { ethers } from "hardhat";

import { ScrollChain, L1MessageQueue } from "../typechain";
import { HardhatEthersSigner } from "@nomicfoundation/hardhat-ethers/signers";
import { randomBytes } from "crypto";
import { expect } from "chai";

describe("ScrollChain.blob", async () => {
  let deployer: HardhatEthersSigner;
  let signer: HardhatEthersSigner;

  let queue: L1MessageQueue;
  let chain: ScrollChain;

  beforeEach(async () => {
    [deployer, signer] = await ethers.getSigners();

    const EmptyContract = await ethers.getContractFactory("EmptyContract", deployer);
    const empty = await EmptyContract.deploy();

    const ProxyAdmin = await ethers.getContractFactory("ProxyAdmin", deployer);
    const admin = await ProxyAdmin.deploy();

    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);
    const queueProxy = await TransparentUpgradeableProxy.deploy(empty.getAddress(), admin.getAddress(), "0x");
    const chainProxy = await TransparentUpgradeableProxy.deploy(empty.getAddress(), admin.getAddress(), "0x");

    const L1MessageQueue = await ethers.getContractFactory("L1MessageQueue", deployer);
    const queueImpl = await L1MessageQueue.deploy(deployer.address, chainProxy.getAddress(), deployer.address);
    await admin.upgrade(queueProxy.getAddress(), queueImpl.getAddress());

    const ScrollChain = await ethers.getContractFactory("ScrollChain", deployer);
    const chainImpl = await ScrollChain.deploy(0, queueProxy.getAddress(), deployer.address);
    await admin.upgrade(chainProxy.getAddress(), chainImpl.getAddress());

    queue = await ethers.getContractAt("L1MessageQueue", await queueProxy.getAddress(), deployer);
    chain = await ethers.getContractAt("ScrollChain", await chainProxy.getAddress(), deployer);

    await chain.initialize(queue.getAddress(), ZeroAddress, 100);
    await chain.addSequencer(deployer.address);
    await chain.addProver(deployer.address);
    await queue.initialize(deployer.address, chain.getAddress(), deployer.address, deployer.address, 10000000);
  });

  context("commit batch", async () => {
    let batchHeader0: Uint8Array;

    beforeEach(async () => {
      // import 10 L1 messages
      for (let i = 0; i < 10; i++) {
        queue.appendCrossDomainMessage(deployer.address, 1000000, "0x");
      }

      // import genesis batch first
      batchHeader0 = new Uint8Array(89);
      batchHeader0[25] = 1;
      await chain.importGenesisBatch(batchHeader0, randomBytes(32));
    });

    it("should revert when caller is not sequencer", async () => {
      await expect(chain.connect(signer).commitBatch(1, batchHeader0, [], "0x")).to.revertedWithCustomError(
        chain,
        "ErrorCallerIsNotSequencer"
      );
    });

    it("should revert when batch is empty", async () => {
      await expect(chain.commitBatch(1, batchHeader0, [], "0x")).to.revertedWithCustomError(chain, "ErrorBatchIsEmpty");
    });

    it("should revert when batch header length too small", async () => {
      const header = new Uint8Array(120);
      header[0] = 1;
      await expect(chain.commitBatch(1, header, ["0x"], "0x")).to.revertedWithCustomError(
        chain,
        "ErrorBatchHeaderLengthTooSmall"
      );
    });

    it("should revert when wrong bitmap length", async () => {
      const header = new Uint8Array(122);
      header[0] = 1;
      await expect(chain.commitBatch(1, header, ["0x"], "0x")).to.revertedWithCustomError(
        chain,
        "ErrorIncorrectBitmapLength"
      );
    });

    it("should revert when incorrect parent batch hash", async () => {
      batchHeader0[25] = 2;
      await expect(chain.commitBatch(1, batchHeader0, ["0x"], "0x")).to.revertedWithCustomError(
        chain,
        "ErrorIncorrectBatchHash"
      );
      batchHeader0[25] = 1;
    });

    it("should revert when ErrorInvalidBatchHeaderVersion", async () => {
      const header = new Uint8Array(121);
      header[0] = 2;
      await expect(chain.commitBatch(1, header, ["0x"], "0x")).to.revertedWithCustomError(
        chain,
        "ErrorInvalidBatchHeaderVersion"
      );
      await expect(chain.commitBatch(2, batchHeader0, ["0x"], "0x")).to.revertedWithCustomError(
        chain,
        "ErrorInvalidBatchHeaderVersion"
      );
    });

    it("should revert when ErrorNoBlobFound", async () => {
      await expect(chain.commitBatch(1, batchHeader0, ["0x"], "0x")).to.revertedWithCustomError(
        chain,
        "ErrorNoBlobFound"
      );
    });

    /* Hardhat doesn't have support for EIP4844 yet.
    const makeTransaction = async (data: string, value: bigint, blobVersionedHashes: Array<string>) => {
      const tx = new Transaction();
      tx.type = 3;
      tx.to = await chain.getAddress();
      tx.data = data;
      tx.nonce = await deployer.getNonce();
      tx.gasLimit = 1000000;
      tx.maxPriorityFeePerGas = (await ethers.provider.getFeeData()).maxPriorityFeePerGas;
      tx.maxFeePerGas = (await ethers.provider.getFeeData()).maxFeePerGas;
      tx.value = value;
      tx.chainId = (await ethers.provider.getNetwork()).chainId;
      tx.maxFeePerBlobGas = ethers.parseUnits("1", "gwei");
      tx.blobVersionedHashes = blobVersionedHashes;
      return tx;
    };

    it("should revert when ErrorFoundMultipleBlob", async () => {
      const data = chain.interface.encodeFunctionData("commitBatch", [1, batchHeader0, ["0x"], "0x"]);
      const tx = await makeTransaction(data, 0n, [ZeroHash, ZeroHash]);
      const signature = await deployer.signMessage(tx.unsignedHash);
      tx.signature = Signature.from(signature);
      const r = await ethers.provider.broadcastTransaction(tx.serialized);
      await expect(r).to.revertedWithCustomError(chain, "ErrorFoundMultipleBlob");
    });

    it("should revert when ErrorNoBlockInChunk", async () => {});

    it("should revert when ErrorIncorrectChunkLength", async () => {});

    it("should revert when ErrorLastL1MessageSkipped", async () => {});

    it("should revert when ErrorNumTxsLessThanNumL1Msgs", async () => {});

    it("should revert when ErrorTooManyTxsInOneChunk", async () => {});

    it("should revert when ErrorIncorrectBitmapLength", async () => {});

    it("should succeed", async () => {});
    */
  });
});
