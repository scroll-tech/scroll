/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { ZeroAddress, concat, getBytes } from "ethers";
import { ethers } from "hardhat";

import { ScrollChain, L1MessageQueue } from "../typechain";

describe("ScrollChain", async () => {
  let queue: L1MessageQueue;
  let chain: ScrollChain;

  beforeEach(async () => {
    const [deployer] = await ethers.getSigners();

    const EmptyContract = await ethers.getContractFactory("EmptyContract", deployer);
    const empty = await EmptyContract.deploy();

    const ProxyAdmin = await ethers.getContractFactory("ProxyAdmin", deployer);
    const admin = await ProxyAdmin.deploy();

    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);
    const queueProxy = await TransparentUpgradeableProxy.deploy(empty.getAddress(), admin.getAddress(), "0x");
    const chainProxy = await TransparentUpgradeableProxy.deploy(empty.getAddress(), admin.getAddress(), "0x");

    const L1MessageQueue = await ethers.getContractFactory("L1MessageQueue", deployer);
    const queueImpl = await L1MessageQueue.deploy(ZeroAddress, chainProxy.getAddress(), deployer.address);
    await admin.upgrade(queueProxy.getAddress(), queueImpl.getAddress());

    const ScrollChain = await ethers.getContractFactory("ScrollChain", deployer);
    const chainImpl = await ScrollChain.deploy(0, queueProxy.getAddress(), deployer.address);
    await admin.upgrade(chainProxy.getAddress(), chainImpl.getAddress());

    queue = await ethers.getContractAt("L1MessageQueue", await queueProxy.getAddress(), deployer);
    chain = await ethers.getContractAt("ScrollChain", await chainProxy.getAddress(), deployer);

    await chain.initialize(queue.getAddress(), ZeroAddress, 100);
    await chain.addSequencer(deployer.address);
    await queue.initialize(ZeroAddress, chain.getAddress(), ZeroAddress, ZeroAddress, 10000000);
  });

  // @note skip this benchmark tests
  it.skip("should succeed", async () => {
    const batchHeader0 = new Uint8Array(89);
    batchHeader0[25] = 1;
    await chain.importGenesisBatch(batchHeader0, "0x0000000000000000000000000000000000000000000000000000000000000001");
    const parentBatchHash = await chain.committedBatches(0);
    console.log("genesis batch hash:", parentBatchHash);
    console.log(`ChunkPerBatch`, `BlockPerChunk`, `TxPerBlock`, `BytesPerTx`, `TotalBytes`, `EstimateGas`);
    for (let numChunks = 3; numChunks <= 6; ++numChunks) {
      for (let numBlocks = 1; numBlocks <= 5; ++numBlocks) {
        for (let numTx = 20; numTx <= Math.min(30, 100 / numBlocks); ++numTx) {
          for (let txLength = 800; txLength <= 1000; txLength += 100) {
            const txs: Array<Uint8Array> = [];
            for (let i = 0; i < numTx; i++) {
              const tx = new Uint8Array(4 + txLength);
              let offset = 3;
              for (let x = txLength; x > 0; x = Math.floor(x / 256)) {
                tx[offset] = x % 256;
                offset -= 1;
              }
              tx.fill(1, 4);
              txs.push(tx);
            }
            const chunk = new Uint8Array(1 + 60 * numBlocks);
            chunk[0] = numBlocks;
            for (let i = 0; i < numBlocks; i++) {
              chunk[1 + i * 60 + 57] = numTx;
            }
            const chunks: Array<Uint8Array> = [];
            for (let i = 0; i < numChunks; i++) {
              const txsInChunk: Array<Uint8Array> = [];
              for (let j = 0; j < numBlocks; j++) {
                txsInChunk.push(getBytes(concat(txs)));
              }
              chunks.push(getBytes(concat([chunk, concat(txsInChunk)])));
            }

            const estimateGas = await chain.commitBatch.estimateGas(0, batchHeader0, chunks, "0x");
            console.log(
              `${numChunks}`,
              `${numBlocks}`,
              `${numTx}`,
              `${txLength}`,
              `${numChunks * numBlocks * numTx * (txLength + 1)}`,
              `${estimateGas.toString()}`
            );
          }
        }
      }
    }
  });
});
