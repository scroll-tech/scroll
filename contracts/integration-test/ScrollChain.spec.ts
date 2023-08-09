/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { constants } from "ethers";
import { ethers } from "hardhat";
import { ScrollChain, L1MessageQueue } from "../typechain";

describe("ScrollChain", async () => {
  let queue: L1MessageQueue;
  let chain: ScrollChain;

  beforeEach(async () => {
    const [deployer] = await ethers.getSigners();

    const L1MessageQueue = await ethers.getContractFactory("L1MessageQueue", deployer);
    queue = await L1MessageQueue.deploy();
    await queue.deployed();

    const RollupVerifier = await ethers.getContractFactory("RollupVerifier", deployer);
    const verifier = await RollupVerifier.deploy();
    await verifier.deployed();

    const ScrollChain = await ethers.getContractFactory("ScrollChain", {
      signer: deployer,
      libraries: { RollupVerifier: verifier.address },
    });
    chain = await ScrollChain.deploy(0);
    await chain.deployed();

    await chain.initialize(queue.address, constants.AddressZero, 44);
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
  /*
  it("should succeed", async () => {
    await chain.importGenesisBatch({
      blocks: [
        {
          blockHash: "0x92826bd3aad2ef70d8061dc4e25150b305d1233d9cd7579433a77d6eb01dae1c",
          parentHash: constants.HashZero,
          blockNumber: 0,
          timestamp: 1639724192,
          baseFee: 1000000000,
          gasLimit: 940000000,
          numTransactions: 0,
          numL1Messages: 0,
        },
      ],
      prevStateRoot: constants.HashZero,
      newStateRoot: "0x1b186a7a90ec3b41a2417062fe44dce8ce82ae76bfbb09eae786a4f1be1895f5",
      withdrawTrieRoot: constants.HashZero,
      batchIndex: 0,
      parentBatchHash: constants.HashZero,
      l2Transactions: [],
    });
    const parentBatchHash = await chain.lastFinalizedBatchHash();
    console.log("genesis batch hash:", parentBatchHash);

    for (let numTx = 1; numTx <= 25; ++numTx) {
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
        const batch = {
          blocks: [
            {
              blockHash: "0xb5baa665b2664c3bfed7eb46e00ebc110ecf2ebd257854a9bf2b9dbc9b2c08f6",
              parentHash: "0x92826bd3aad2ef70d8061dc4e25150b305d1233d9cd7579433a77d6eb01dae1c",
              blockNumber: 1,
              timestamp: numTx * 100000 + txLength,
              baseFee: 0,
              gasLimit: 0,
              numTransactions: 0,
              numL1Messages: 0,
            },
          ],
          prevStateRoot: "0x1b186a7a90ec3b41a2417062fe44dce8ce82ae76bfbb09eae786a4f1be1895f5",
          newStateRoot: "0xb5baa665b2664c3bfed7eb46e00ebc110ecf2ebd257854a9bf2b9dbc9b2c08f6",
          withdrawTrieRoot: "0xb5baa665b2664c3bfed7eb46e00ebc110ecf2ebd257854a9bf2b9dbc9b2c08f6",
          batchIndex: 1,
          parentBatchHash: parentBatchHash,
          l2Transactions: concat(txs),
        };
        const estimateGas = await chain.estimateGas.commitBatch(batch);
        const tx = await chain.commitBatch(batch, { gasLimit: estimateGas.mul(12).div(10) });
        const receipt = await tx.wait();
        console.log(
          "Commit batch with l2TransactionsBytes:",
          numTx * (txLength + 4),
          "gasLimit:",
          tx.gasLimit.toString(),
          "estimateGas:",
          estimateGas.toString(),
          "gasUsed:",
          receipt.gasUsed.toString()
        );
      }
    }
  });
  */
});
