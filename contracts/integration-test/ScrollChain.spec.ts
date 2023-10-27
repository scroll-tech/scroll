/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { concat } from "ethers/lib/utils";
import { constants } from "ethers";
const { ethers, upgrades } = require('hardhat');
import { ScrollChain, L1MessageQueue } from "../typechain";

describe("ScrollChain", async () => {
  // Scroll contracts
  let queue: L1MessageQueue;
  let chain: ScrollChain;
  // Opside contracts
  let deployer;
  let trustedSequencer;
  let trustedAggregator;
  let admin;
  let aggregator1;
  let aggregator2;
  let aggregator3;
  let aggregator4;
  let verifierContract;
  let polygonZkEVMBridgeContract;
  let ZkEVMContract;
  let maticTokenContract;
  let polygonZkEVMGlobalExitRoot;
  let slotAdapterContract;
  let depositContract;
  let opsideSlotsContract;
  let globalRewardDistributionContract;
  let openRegistrarContract;
  let globalRewardPoolContract;

  beforeEach(async () => {
    const [deployer] = await ethers.getSigners();
    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);

    // Opside contracts deployment
    const globalRewardDistributionFactory = await ethers.getContractFactory("GlobalRewardDistribution", deployer);
    globalRewardDistributionContract = await globalRewardDistributionFactory.deploy();
    await globalRewardDistributionContract.deployed();

    // deploy opsideSlotsFactory
    const opsideSlotsFactory = await ethers.getContractFactory("OpsideSlots", deployer);
    const opsideSlotsFactoryImpl = await opsideSlotsFactory.deploy();
    await opsideSlotsFactoryImpl.deployed();
    const opsideSlots = await TransparentUpgradeableProxy.deploy(opsideSlotsFactoryImpl.address, deployer.address, "0x");
    await opsideSlots.deployed();

    // deploy opsideSlotsFactory
    const OpenRegistrarFactory = await ethers.getContractFactory("OpenRegistrar", deployer);
    const OpenRegistrarFactoryImpl = await OpenRegistrarFactory.deploy();
    await OpenRegistrarFactoryImpl.deployed();
    const OpenRegistrarFactoryProxy = await TransparentUpgradeableProxy.deploy(OpenRegistrarFactoryImpl.address, deployer.address, "0x");
    await OpenRegistrarFactoryProxy.deployed();


    const globalRewardPoolFactory = await ethers.getContractFactory("GlobalRewardPool", deployer);
    globalRewardPoolContract = await upgrades.deployProxy(
        globalRewardPoolFactory,
        [opsideSlotsContract.address, globalRewardDistributionContract.address],
        {}
      );

    await opsideSlotsContract.initialize(openRegistrarContract.address, globalRewardPoolContract.address);

    const SlotAdapterFactory = await ethers.getContractFactory('SlotAdapter');

    const depositFactory = await ethers.getContractFactory('MinerDeposit');




    // Scroll contracts deployment
    const ProxyAdmin = await ethers.getContractFactory("ProxyAdmin", deployer);
    const admin = await ProxyAdmin.deploy();
    await admin.deployed();

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

    await chain.initialize(queue.address, constants.AddressZero, 100);
    await chain.addSequencer(deployer.address);
    await queue.initialize(
      constants.AddressZero,
      chain.address,
      constants.AddressZero,
      constants.AddressZero,
      10000000
    );
  });

  // @note skip this benchmark tests
  it("should succeed", async () => {
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
                txsInChunk.push(concat(txs));
              }
              chunks.push(concat([chunk, concat(txsInChunk)]));
            }
            
            const estimateGas = await chain.estimateGas.commitBatch(0, batchHeader0, chunks, "0x");

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
