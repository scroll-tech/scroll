/* eslint-disable node/no-missing-import */
import { expect } from "chai";
import exp from "constants";
import * as dotenv from "dotenv";
import { Signature, Transaction, ZeroAddress, ZeroHash, randomBytes } from "ethers";

import { ethers } from "hardhat";

dotenv.config();

async function main() {
  const [deployer, signer] = await ethers.getSigners();

  const EmptyContract = await ethers.getContractFactory("EmptyContract", deployer);
  const empty = await EmptyContract.deploy();
  console.log("EmptyContract address:", await empty.getAddress());

  const ProxyAdmin = await ethers.getContractFactory("ProxyAdmin", deployer);
  const admin = await ProxyAdmin.deploy();
  console.log("ProxyAdmin address:", await admin.getAddress());

  const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);
  const queueProxy = await TransparentUpgradeableProxy.deploy(empty.getAddress(), admin.getAddress(), "0x");
  console.log("queueProxy address:", await queueProxy.getAddress());
  const chainProxy = await TransparentUpgradeableProxy.deploy(empty.getAddress(), admin.getAddress(), "0x");
  console.log("chainProxy address:", await chainProxy.getAddress());

  const L1MessageQueue = await ethers.getContractFactory("L1MessageQueue", deployer);
  const queueImpl = await L1MessageQueue.deploy(deployer.address, chainProxy.getAddress(), deployer.address);
  console.log("queueImpl address:", await queueImpl.getAddress());
  let tx = await admin.upgrade(queueProxy.getAddress(), queueImpl.getAddress());
  await tx.wait();
  console.log("admin.upgrade, hash:", tx.hash);

  const ScrollChain = await ethers.getContractFactory("ScrollChain", deployer);
  const chainImpl = await ScrollChain.deploy(0, queueProxy.getAddress(), deployer.address);
  console.log("chainImpl address:", await chainImpl.getAddress());
  tx = await admin.upgrade(chainProxy.getAddress(), chainImpl.getAddress());
  await tx.wait();
  console.log("admin.upgrade, hash:", tx.hash);

  const queue = await ethers.getContractAt("L1MessageQueue", await queueProxy.getAddress(), deployer);
  const chain = await ethers.getContractAt("ScrollChain", await chainProxy.getAddress(), deployer);

  tx = await chain.initialize(queue.getAddress(), ZeroAddress, 100);
  await tx.wait();
  console.log("chain.initialize, hash:", tx.hash);
  tx = await chain.addSequencer(deployer.address);
  await tx.wait();
  console.log("chain.addSequencer, hash:", tx.hash);
  tx = await chain.addProver(deployer.address);
  await tx.wait();
  console.log("chain.addProver, hash:", tx.hash);
  tx = await queue.initialize(deployer.address, chain.getAddress(), deployer.address, deployer.address, 10000000);
  await tx.wait();
  console.log("queue.initialize, hash:", tx.hash);
  tx = await queue.updateGasOracle(ZeroAddress);
  await tx.wait();
  console.log("queue.updateGasOracle, hash:", tx.hash);

  // import 10 L1 messages
  for (let i = 0; i < 10; i++) {
    tx = await queue.appendCrossDomainMessage(deployer.address, 1000000, "0x");
    await tx.wait();
    console.log("queue.appendCrossDomainMessage, hash:", tx.hash);
  }

  // import genesis batch first
  const batchHeader0 = new Uint8Array(89);
  batchHeader0[25] = 1;
  tx = await chain.importGenesisBatch(batchHeader0, randomBytes(32));
  await tx.wait();
  console.log("chain.importGenesisBatch, hash:", tx.hash);

  // should revert when caller is not sequencer
  {
    await expect(chain.connect(signer).commitBatch(1, batchHeader0, [], "0x")).to.revertedWithCustomError(
      chain,
      "ErrorCallerIsNotSequencer"
    );
    console.log("pass: should revert when caller is not sequencer");
  }
  // should revert when batch is empty
  {
    await expect(chain.commitBatch(1, batchHeader0, [], "0x")).to.revertedWithCustomError(chain, "ErrorBatchIsEmpty");
    console.log("pass: should revert when batch is empty");
  }
  // should revert when batch header length too small
  {
    const header = new Uint8Array(120);
    header[0] = 1;
    await expect(chain.commitBatch(1, header, ["0x"], "0x")).to.revertedWithCustomError(
      chain,
      "ErrorBatchHeaderLengthTooSmall"
    );
    console.log("pass: should revert when batch header length too small");
  }
  // should revert when wrong bitmap length
  {
    const header = new Uint8Array(122);
    header[0] = 1;
    await expect(chain.commitBatch(1, header, ["0x"], "0x")).to.revertedWithCustomError(
      chain,
      "ErrorIncorrectBitmapLength"
    );
    console.log("pass: should revert when wrong bitmap length");
  }
  // should revert when incorrect parent batch hash
  {
    batchHeader0[25] = 2;
    await expect(chain.commitBatch(1, batchHeader0, ["0x"], "0x")).to.revertedWithCustomError(
      chain,
      "ErrorIncorrectBatchHash"
    );
    batchHeader0[25] = 1;
    console.log("pass: should revert when incorrect parent batch hash");
  }
  // should revert when ErrorInvalidBatchHeaderVersion
  {
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
    console.log("pass: should revert when ErrorInvalidBatchHeaderVersion");
  }

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
  /*
  // should revert when ErrorNoBlobFound
  {
    await expect(chain.commitBatch(1, batchHeader0, ["0x"], "0x")).to.revertedWithCustomError(
      chain,
      "ErrorNoBlobFound"
    );
    console.log("pass: should revert when ErrorNoBlobFound");
  }
  // should revert when ErrorFoundMultipleBlob
  {
    const data = chain.interface.encodeFunctionData("commitBatch", [1, batchHeader0, ["0x"], "0x"]);
    const tx = await makeTransaction(data, 0n, [ZeroHash, ZeroHash]);
    const signature = await deployer.signMessage(tx.unsignedHash);
    tx.signature = Signature.from(signature);
    const r = await ethers.provider.broadcastTransaction(tx.serialized);
    await expect(r).to.revertedWithCustomError(chain, "ErrorFoundMultipleBlob");
    console.log("pass: should revert when ErrorFoundMultipleBlob");
  }
  */
  // should revert should revert when ErrorNoBlockInChunk
  {
    const chunks = new Array(1);
    const chunk0 = new Uint8Array(1);
    chunks[0] = chunk0;
    await expect(chain.commitBatch(1, batchHeader0, chunks, "0x")).to.revertedWithCustomError(
      chain,
      "ErrorNoBlockInChunk"
    );
    console.log("pass: should revert should revert when ErrorNoBlockInChunk");
  }
  // should revert when ErrorIncorrectChunkLength
  {
    const chunks = new Array(1);
    const chunk0 = new Uint8Array(1);
    chunk0[0] = 1;
    chunks[0] = chunk0;
    await expect(chain.commitBatch(1, batchHeader0, chunks, "0x")).to.revertedWithCustomError(
      chain,
      "ErrorIncorrectChunkLength"
    );
    console.log("pass: should revert when ErrorIncorrectChunkLength");
  }
  // should revert when ErrorLastL1MessageSkipped
  {
    const chunks = new Array(1);
    const chunk0 = new Uint8Array(1 + 60);
    const bitmap = new Uint8Array(32);
    chunk0[0] = 1; // one block in this chunk
    chunk0[58] = 1; // numTransactions = 1
    chunk0[60] = 1; // numL1Messages = 1
    bitmap[31] = 1;
    chunks[0] = chunk0;
    await expect(chain.commitBatch(1, batchHeader0, chunks, bitmap)).to.revertedWithCustomError(
      chain,
      "ErrorLastL1MessageSkipped"
    );
    console.log("pass: should revert when ErrorLastL1MessageSkipped");
  }
  // should revert when ErrorNumTxsLessThanNumL1Msgs
  {
    const chunks = new Array(1);
    const chunk0 = new Uint8Array(1 + 60);
    const bitmap = new Uint8Array(32);
    chunk0[0] = 1; // one block in this chunk
    chunk0[58] = 1; // numTransactions = 1
    chunk0[60] = 3; // numL1Messages = 3
    bitmap[31] = 3;
    chunks[0] = chunk0;
    await expect(chain.commitBatch(1, batchHeader0, chunks, bitmap)).to.revertedWithCustomError(
      chain,
      "ErrorNumTxsLessThanNumL1Msgs"
    );
    console.log("pass: should revert when ErrorNumTxsLessThanNumL1Msgs");
  }
  // should succeed
  {
    const batchHash0 = await chain.committedBatches(0);
    console.log("batchHash0", batchHash0);
    console.log("messageHash0", await queue.messageQueue(0));
    // commit batch1, one chunk with one block, 1 tx, 1 L1 message, no skip
    // => payload for data hash of chunk0
    //   0000000000000000
    //   0000000000000000
    //   0000000000000000000000000000000000000000000000000000000000000000
    //   0000000000000000
    //   0001
    //   24411d21c906219f9e13068d4d6f61a9d71396c6b17a08a84cd1852d5edaa9db
    // => data hash for chunk0
    //   e71abc791ac90b3076b2ac3114c786854d5c48c40d088617a3346c166e85c0a2
    // => data hash for all chunks
    //   19cb48b9e81b1cc64417751760e24c7034cc3c3563f4102409704ed890860d27
    // => payload for batch header
    //   01
    //   0000000000000001
    //   0000000000000001
    //   0000000000000001
    //   19cb48b9e81b1cc64417751760e24c7034cc3c3563f4102409704ed890860d27
    //   0000000000000000000000000000000000000000000000000000000000000000
    //   99a2f6e2bca66d5eb3c178157b8be43b719f5c34150300bd6aea2e3c95954542
    //   0000000000000000000000000000000000000000000000000000000000000000
    // => hash for batch header
    //   c317384d9595643daac2fc8be2aa35030f539ad9269b33cb511b7b1001f3da9d
    const chunk0 = new Uint8Array(1 + 60);
    chunk0[0] = 1; // numBlocks = 1
    chunk0[58] = 1; // numTransactions = 1
    chunk0[60] = 1; // numL1Messages = 1
    const chunks = new Array(1);
    chunks[0] = chunk0;
    const bitmap = new Uint8Array(32);
    const tx = await chain.commitBatch(1, batchHeader0, chunks, bitmap);
    const receipt = await tx.wait();
    await expect(tx)
      .to.emit(chain, "CommitBatch")
      .withArgs(1, "0xc317384d9595643daac2fc8be2aa35030f539ad9269b33cb511b7b1001f3da9d");
    expect(await chain.isBatchFinalized(1)).to.eq(false);
    expect(await chain.committedBatches(1)).to.eq("0xc317384d9595643daac2fc8be2aa35030f539ad9269b33cb511b7b1001f3da9d");
    console.log("pass: should succeed");
  }
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});