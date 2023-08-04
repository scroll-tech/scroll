/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { constants } from "ethers";
import { concat } from "ethers/lib/utils";
import { ethers } from "hardhat";
import { ScrollChain, L1MessageQueue } from "../typechain";

describe("ScrollChain", async () => {
  let queue: L1MessageQueue;
  let chain: ScrollChain;

  beforeEach(async () => {
    const [deployer] = await ethers.getSigners();

    const ProxyAdmin = await ethers.getContractFactory("ProxyAdmin", deployer);
    const admin = await ProxyAdmin.deploy();
    await admin.deployed();

    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);

    const L1MessageQueue = await ethers.getContractFactory("L1MessageQueue", deployer);
    const queueImpl = await L1MessageQueue.deploy();
    await queueImpl.deployed();
    const queueProxy = await TransparentUpgradeableProxy.deploy(queueImpl.address, admin.address, "0x");
    await queueProxy.deployed();
    queue = await ethers.getContractAt("L1MessageQueue", queueProxy.address, deployer);

    const ScrollChain = await ethers.getContractFactory("ScrollChain", deployer);
    const chainImpl = await ScrollChain.deploy(0);
    await chainImpl.deployed();
    const chainProxy = await TransparentUpgradeableProxy.deploy(chainImpl.address, admin.address, "0x");
    await chainProxy.deployed();
    chain = await ethers.getContractAt("ScrollChain", chainProxy.address, deployer);

    await chain.initialize(queue.address, constants.AddressZero, 44);
    await chain.updateSequencer(deployer.address, true);
    await queue.initialize(
      constants.AddressZero,
      chain.address,
      constants.AddressZero,
      constants.AddressZero,
      10000000
    );
  });

  // @note skip this benchmark tests
  it.skip("should succeed", async () => {
    const batchHeader0 = new Uint8Array(89);
    batchHeader0[25] = 1;
    await chain.importGenesisBatch(batchHeader0, "0x0000000000000000000000000000000000000000000000000000000000000001");
    const parentBatchHash = await chain.committedBatches(0);
    console.log("genesis batch hash:", parentBatchHash);

    for (let numChunks = 1; numChunks <= 3; ++numChunks) {
      for (let numBlocks = 1; numBlocks <= 3; ++numBlocks) {
        for (let numTx = 1; numTx <= 10; ++numTx) {
          for (let txLength = 100; txLength <= 1000; txLength += 100) {
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
                txsInChunk.push(concat(txs));
              }
              chunks.push(concat([chunk, concat(txsInChunk)]));
            }

            const estimateGas = await chain.estimateGas.commitBatch(0, batchHeader0, chunks, "0x");
            console.log(
              `ChunkPerBatch[${numChunks}]`,
              `BlockPerChunk[${numBlocks}]`,
              `TxPerBlock[${numTx}]`,
              `BytesPerTx[${txLength}]`,
              `TotalBytes[${numChunks * numBlocks * numTx * (txLength + 1)}]`,
              `EstimateGas[${estimateGas.toString()}]`
            );
          }
        }
      }
    }
  });
});
