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
    let SlotManager;
    let aggregator1;
    let aggregator2;
    let aggregator3;
    let slotAdapterContract;
    let depositContract;
    let opsideSlotsContract;
    let globalRewardDistributionContract;
    let openRegistrarContract;
    let globalRewardPoolContract;

    // Constants
    const NoProofPunishAmount = ethers.utils.parseEther('1');
    const IncorrectProofPunishAmount = ethers.utils.parseEther('10');

    beforeEach(async () => {
        const [deployer, trustedSequencer, trustedAggregator, SlotManager, aggregator1, aggregator2, aggregator3] = await ethers.getSigners();
        const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);

        // Scroll contracts deployment and initialize
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

        // Opside contracts deployment and initialize
        const globalRewardDistributionFactory = await ethers.getContractFactory("GlobalRewardDistribution", deployer);
        globalRewardDistributionContract = await globalRewardDistributionFactory.deploy();
        await globalRewardDistributionContract.deployed();

        const opsideSlotsFactory = await ethers.getContractFactory("OpsideSlots", deployer);
        opsideSlotsContract = await upgrades.deployProxy(
            opsideSlotsFactory,
            [
            ],
            { initializer: false });

        const openRegistrarFactory = await ethers.getContractFactory("OpenRegistrar", deployer);
        openRegistrarContract = await upgrades.deployProxy(
            openRegistrarFactory,
            [
                opsideSlotsContract.address
            ],
            {
            });


        const globalRewardPoolFactory = await ethers.getContractFactory("GlobalRewardPool", deployer);
        globalRewardPoolContract = await upgrades.deployProxy(
            globalRewardPoolFactory,
            [opsideSlotsContract.address, globalRewardDistributionContract.address],
            {}
        );

        await opsideSlotsContract.initialize(openRegistrarContract.address, globalRewardPoolContract.address);
        const SlotAdapterFactory = await ethers.getContractFactory('SlotAdapter');
        const depositFactory = await ethers.getContractFactory('MinerDeposit');

        depositContract = await upgrades.deployProxy(depositFactory, [], {});
        slotAdapterContract = await upgrades.deployProxy(SlotAdapterFactory, [], {
            initializer: false,
            constructorArgs: [],
            unsafeAllow: ['constructor', 'state-variable-immutable'],
        });

        await slotAdapterContract.initialize(SlotManager.address, opsideSlotsContract.address, globalRewardPoolContract.address);
        await slotAdapterContract.setZKEvmContract(chain.address);

        await chain.connect(deployer).setSlotAdapter(slotAdapterContract.address);
        
        await chain.connect(deployer).setDeposit(depositContract.address);
        await chain.connect(deployer).setMinDeposit(ethers.utils.parseEther('100'));
        await chain.connect(deployer).setNoProofPunishAmount(NoProofPunishAmount);
        await chain.connect(deployer).setIncorrectProofPunishAmount(IncorrectProofPunishAmount);
        
        await openRegistrarContract.connect(deployer).setRent(182, ethers.utils.parseEther('100'));
        await openRegistrarContract.connect(deployer).addRegistrant(deployer.address);
        await depositContract.connect(deployer).setSlotAdapter(slotAdapterContract.address);
        await openRegistrarContract.connect(deployer).request('test', deployer.address, ethers.BigNumber.from(182), ethers.utils.parseEther('100'), { value: ethers.utils.parseEther('100') });
        
        const regId = await openRegistrarContract.regId();
        await openRegistrarContract.connect(deployer).accept(regId);
        const slotId = await opsideSlotsContract.slotId();

        const chainId = 1111;
        await opsideSlotsContract.connect(deployer).setup(slotId, chainId, slotAdapterContract.address);
        await opsideSlotsContract.connect(deployer).start(slotId);
        await depositContract.connect(aggregator1).deposit(ethers.utils.parseEther('200'), { value: ethers.utils.parseEther('200') });
        await depositContract.connect(aggregator2).deposit(ethers.utils.parseEther('200'), { value: ethers.utils.parseEther('200') });
        await depositContract.connect(aggregator3).deposit(ethers.utils.parseEther('200'), { value: ethers.utils.parseEther('200') });

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

    it('one aggregator: commit two proof hashes and submit two proofs', async () => {
        
        const l2txData = '0x123456';
        const currentTimestamp = (await ethers.provider.getBlock()).timestamp;
        const sequence = {
            transactions: l2txData,
            globalExitRoot: ethers.constants.HashZero,
            timestamp: currentTimestamp,
            minForcedTimestamp: 0,
        };
        
        const lastBatchSequenced = await ZkEVMContract.lastBatchSequenced();
        
        // Sequence Batches
        chain.connect(deployer).
        
        await expect(ZkEVMContract.connect(trustedSequencer).sequenceBatches([sequence], trustedSequencer.address))
        .to.emit(ZkEVMContract, 'SequenceBatches')
        .withArgs(lastBatchSequenced.add(1));

        await expect(ZkEVMContract.connect(trustedSequencer).sequenceBatches([sequence, sequence], trustedSequencer.address))
            .to.emit(ZkEVMContract, 'SequenceBatches')
            .withArgs(lastBatchSequenced.add(3));

        // trustedAggregator forge the batch
        const newLocalExitRoot = '0x0000000000000000000000000000000000000000000000000000000000000000';
        const newStateRoot = '0x0000000000000000000000000000000000000000000000000000000000000002';
        
        let numBatch = (await ZkEVMContract.lastVerifiedBatch()).add(1);

        const zkProofFFlonk = '0x20227cbcef731b6cbdc0edd5850c63dc7fbc27fb58d12cd4d08298799cf66a0512c230867d3375a1f4669e7267dad2c31ebcddbaccea6abd67798ceae35ae7611c665b6069339e6812d015e239594aa71c4e217288e374448c358f6459e057c91ad2ef514570b5dea21508e214430daadabdd23433820000fe98b1c6fa81d5c512b86fbf87bd7102775f8ef1da7e8014dc7aab225503237c7927c032e589e9a01a0eab9fda82ffe834c2a4977f36cc9bcb1f2327bdac5fb48ffbeb9656efcdf70d2656c328903e9fb96e4e3f470c447b3053cc68d68cf0ad317fe10aa7f254222e47ea07f3c1c3aacb74e5926a67262f261c1ed3120576ab877b49a81fb8aac51431858662af6b1a8138a44e9d0812d032340369459ccc98b109347cc874c7202dceecc3dbb09d7f9e5658f1ca3a92d22be1fa28f9945205d853e2c866d9b649301ac9857b07b92e4865283d3d5e2b711ea5f85cb2da71965382ece050508d3d008bbe4df5458f70bd3e1bfcc50b34222b43cd28cbe39a3bab6e464664a742161df99c607638e415ced49d0cd719518539ed5f561f81d07fe40d3ce85508e0332465313e60ad9ae271d580022ffca4fbe4d72d38d18e7a6e20d020a1d1e5a8f411291ab95521386fa538ddfe6a391d4a3669cc64c40f07895f031550b32f7d73205a69c214a8ef3cdf996c495e3fd24c00873f30ea6b2bfabfd38de1c3da357d1fefe203573fdad22f675cb5cfabbec0a041b1b31274f70193da8e90cfc4d6dc054c7cd26d09c1dadd064ec52b6ddcfa0cb144d65d9e131c0c88f8004f90d363034d839aa7760167b5302c36d2c2f6714b41782070b10c51c178bd923182d28502f36e19b079b190008c46d19c399331fd60b6b6bde898bd1dd0a71ee7ec7ff7124cc3d374846614389e7b5975b77c4059bc42b810673dbb6f8b951e5b636bdf24afd2a3cbe96ce8600e8a79731b4a56c697596e0bff7b73f413bdbc75069b002b00d713fae8d6450428246f1b794d56717050fdb77bbe094ac2ee6af54a153e2fb8ce1d31a86c4fdd523783b910bedf7db58a46ba6ce48ac3ca194f3cf2275e';

        let blockNumber = await ethers.provider.getBlockNumber();

        let proofhash = ethers.utils.solidityKeccak256(['bytes', 'address'], [ethers.utils.keccak256(zkProofFFlonk), aggregator1.address]);

        await expect(ZkEVMContract.connect(aggregator1).submitProofHash(numBatch.sub(1), numBatch, proofhash)).to.emit(ZkEVMContract, 'SubmitProofHash').withArgs(aggregator1.address, numBatch.sub(1), numBatch, proofhash);

        await expect(ZkEVMContract.connect(aggregator1).submitProofHash(numBatch, numBatch.add(2), proofhash)).to.emit(ZkEVMContract, 'SubmitProofHash').withArgs(aggregator1.address, numBatch, numBatch.add(2), proofhash);

        const sequencedBatch = await ZkEVMContract.sequencedBatches(numBatch);
        expect(sequencedBatch.blockNumber).to.be.equal(blockNumber + 1);
        // revert
        // await ZkEVMContract.connect(aggregator1).verifyBatches(numBatch - 1, numBatch, newLocalExitRoot, newStateRoot, zkProofFFlonk);
        await expect(ZkEVMContract.connect(aggregator1).verifyBatches(numBatch.sub(1), numBatch, newLocalExitRoot, newStateRoot, zkProofFFlonk)).to.be.revertedWithCustomError(ZkEVMContract, 'SubmitProofEarly');
        for (let i = 0; i < 20 - 1; i++ ) {
            await hre.network.provider.request({
                method: "evm_mine",
            });
        }
        // await ZkEVMContract.connect(aggregator1).verifyBatches(numBatch - 1, numBatch, newLocalExitRoot, newStateRoot, zkProofFFlonk);
        await expect(ZkEVMContract.connect(aggregator1).verifyBatches(numBatch.sub(1), numBatch, newLocalExitRoot, newStateRoot, zkProofFFlonk)).to.emit(ZkEVMContract, 'VerifyBatchesTrustedAggregator').withArgs(numBatch, newStateRoot, aggregator1.address);
        await expect(ZkEVMContract.connect(aggregator1).verifyBatches(numBatch, numBatch.add(2), newLocalExitRoot, newStateRoot, zkProofFFlonk)).to.emit(ZkEVMContract, 'VerifyBatchesTrustedAggregator').withArgs(numBatch.add(2), newStateRoot, aggregator1.address);
    });
});
