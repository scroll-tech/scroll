/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { concat } from "ethers/lib/utils";
import { constants } from "ethers";
const { ethers, upgrades } = require('hardhat');
import { MockScrollChain, L1MessageQueue } from "../typechain";
const hre = require('hardhat');
import { expect } from "chai";

describe("MockScrollChain", async () => {
    // Scroll contracts 
    let queue: L1MessageQueue;
    let chain_deployer: MockScrollChain;
    let chain_aggr1: MockScrollChain;
    let chain_aggr2: MockScrollChain;
    let chain_aggr3: MockScrollChain;

    // Opside contracts
    let deployer
    let SlotManager;
    let aggregator1: { address: string; };
    let aggregator2: { address: any; };
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
        [deployer, SlotManager, aggregator1, aggregator2, aggregator3] = await ethers.getSigners();
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

        const ScrollChain = await ethers.getContractFactory("MockScrollChain", deployer);
        const chainImpl = await ScrollChain.deploy(0);
        await chainImpl.deployed();
        const chainProxy = await TransparentUpgradeableProxy.deploy(chainImpl.address, admin.address, "0x");
        await chainProxy.deployed();
        chain_deployer = await ethers.getContractAt("MockScrollChain", chainProxy.address, deployer);
        chain_aggr1 = await ethers.getContractAt("MockScrollChain", chainProxy.address, aggregator1);
        chain_aggr2 = await ethers.getContractAt("MockScrollChain", chainProxy.address, aggregator2);
        chain_aggr3 = await ethers.getContractAt("MockScrollChain", chainProxy.address, aggregator3);

        await chain_deployer.initialize(queue.address, constants.AddressZero, 100);

        await chain_deployer.addSequencer(deployer.address);
        await expect(chain_deployer.setProofHashCommitEpoch(20)).to.emit(chain_deployer, 'SetProofHashCommitEpoch').withArgs(20);
        await expect(chain_deployer.setProofCommitEpoch(20)).to.emit(chain_deployer, 'SetProofCommitEpoch').withArgs(20);
        await queue.initialize(
            constants.AddressZero,
            chain_deployer.address,
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
        await slotAdapterContract.setZKEvmContract(chain_deployer.address);

        await chain_deployer.connect(deployer).setSlotAdapter(slotAdapterContract.address);
        await chain_deployer.connect(deployer).setDeposit(depositContract.address);
        await chain_deployer.connect(deployer).setMinDeposit(ethers.utils.parseEther('100'));
        await chain_deployer.connect(deployer).setNoProofPunishAmount(NoProofPunishAmount);
        await chain_deployer.connect(deployer).setIncorrectProofPunishAmount(IncorrectProofPunishAmount);
        
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
        await chain_deployer.importGenesisBatch(batchHeader0, "0x0000000000000000000000000000000000000000000000000000000000000001");
        const parentBatchHash = await chain_deployer.committedBatches(0);
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

                        const estimateGas = await chain_deployer.estimateGas.commitBatch(0, batchHeader0, chunks, "0x");

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

    it.skip('one aggregator: one commit one proof hash and submit one proof', async () => {
        // add sequencer/prover
        await chain_deployer.addSequencer(aggregator1.address);
        await chain_deployer.addProver(aggregator1.address);
        
        // import genesis 
        const genisis_parentBatchHeader = "0x0000000000000000000000000000000000000000000000000061a5de00a189b490960523626c576089401f5413e4ca6f5fe5f16004d764ccd00000000000000000000000000000000000000000000000000000000000000000"
        const _stateRoot = "0x08d535cc60f40af5dd3b31e0998d7567c2d568b224bed2ba26070aeb078d1339"
        await chain_deployer.importGenesisBatch(genisis_parentBatchHeader, _stateRoot);
        
        // Sequence Batches
        const version = 0
        const parentBatchHeader = "0x0000000000000000000000000000000000000000000000000061a5de00a189b490960523626c576089401f5413e4ca6f5fe5f16004d764ccd00000000000000000000000000000000000000000000000000000000000000000"
        var chunks
        const txs: Array<Uint8Array> = [];
        for (let i = 0; i < 20; i++) {
        const tx = new Uint8Array(4 + 100);
        let offset = 3;
        for (let x = 100; x > 0; x = Math.floor(x / 256)) {
            tx[offset] = x % 256;
            offset -= 1;
        }
        tx.fill(1, 4);
        txs.push(tx);
        }
        const chunk = new Uint8Array(1 + 60 * 1);
        chunk[0] = 1;
        for (let i = 0; i < 1; i++) {
        chunk[1 + i * 60 + 57] = 20;
        }
        chunks = [];
        for (let i = 0; i < 1; i++) {
        const txsInChunk: Array<Uint8Array> = [];
        for (let j = 0; j < 1; j++) {
            txsInChunk.push(concat(txs));
        }
        chunks.push(concat([chunk, concat(txsInChunk)]));
        }   
        await chain_aggr1.commitBatch(version, parentBatchHeader, chunks, "0x");

        // commit proof hash
        const batchIndex = 1
        const aggrProof = "0x0000000000000000000000000000000000000000009a53f7e9cfe41b13c3841c0000000000000000000000000000000000000000000a71e92dc0f7ec70d961a90000000000000000000000000000000000000000000011b9320e9a71777058090000000000000000000000000000000000000000002f780de098a2e133ac8e83000000000000000000000000000000000000000000cc97df3098b8c6acf22aae000000000000000000000000000000000000000000000905a9902c8cee0a218a000000000000000000000000000000000000000000bdab3b1b789280ac90d6740000000000000000000000000000000000000000009f6c2cb25710d30e52a392000000000000000000000000000000000000000000001f1c84d3acd6ec70d8bf00000000000000000000000000000000000000000094c031e6f78746862ac083000000000000000000000000000000000000000000b93466a296a4bdb0e5d5ae000000000000000000000000000000000000000000000bdb3de2c4423da15ae302f05a845ad6f200414ebc1613ba889275beecee980e9106d39e11f981f4a50c285faad90f3006c290623eace0bcb52c8f679e92f2b9152d13550ac6a70fbfa120f2b00bf31dafdd764a814d5fafcb93e6cb0d8f2018d1dbf3d17496a4e3d4d30ed399a8532b8956f668609db8767c073034f4fba3baa6249dad8382ea7aace40b3dc91da9aefb560f861f9e4905f5987a5dccd5ce5ea1005fa14a921a0a00e32f3da3e6b13ed176f29839e2556d48ad5534c50d5aad0b2b03df4913c8f002ab1c65005c5c001d5d299cb2e351def07da544bd51adb6d1b614630012a314f5de0b8abfe3cc752951df2a095757842c0f7259ff608cc8896f2990f847ef6577642b028e2bc10b0110afd48eccfe2a3e018739e7b67e930437edf818bfc49deaa9106019364a8cd1c288713c26002be01b252c742963d092b6419c64afb9ac71f6000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020881a8548f4145f7e0762273f164a399c5d5625f4ffeac6270e421e62c1aab150240131d5e547c732e96c4770bfb77fc33eabb48c288390d39ff8b62155040ba1b56eb3b12b5b2f0276eecb4316c625f87bc0c8db14417ff536bf67551e5c79b2d882922c93912c1238d11deb1508881ee95ef019339501fd9d0911e438d37dd0d13fcd3afd30787ade50f2aa952a08cf18c41e5209026b1adc738439734ba990e7ad4df140e5c6e6270af5f01fce9fc4f6218b5df5fdde0902167822c49157a01e54d3e14f0e5cf132988e141cd06c96ad514a7722f4c2fbd972eb3e1039c071d0242cc4c3674899d81b3b7a1f790bad42155afa34b9b9860c4f26c2007f83a2de3fed404a3d8ce62e4fcd5cb0a68be6ae4fb0c5f5499ed8648ef4008a0de4003f7f0496585f955a236ce6411b57b5affd630840306536e73c3f271d201352d09ca8109bf17fd6b500705448cd96437eae6c6da52d0d6ce1e9bdbc951d13a48142695006219a2498e9af06fc8c31a952ec26bfc34e80821e60561226ebf6c83049e064283ed6c432ed0c278c03b949eeb1f3cd0636d851cc0f69b1d8895a4af111a86d992a37a8121fadbf600e2876e3553d062f2715e27d92fa2c7359077af18686295314c7992037b9caa277ee7c86b07d6bdae816ff47e76f6fa167b81bd0250c7342c813eabb8787f8de77a67f9ed7201060fbe027d49af30b7e38e8f2d00000000000000000000000000000000000000000000000000000000000000010f88cd2c11e8cffa8cadfdfc19ce258d47e16d4e0761b0637aaec6b1f985b6971cde71f7f4608b9a7c199b3ec4c7649cebff13561b66bef60252d2c04e2de6251e0465d8d2995cc2cf08a99200756dfaf1c108226eea7e663984ded302ebd2c902eade1d83644ed50425517e9741e9ea243e1aec8c23ca4e5efd7b084f02877d0983f78bb96761d4a687bea7c4523f674816bd9b03bd10cfb93d9a54cc0965d62cfcc0ff4b8c3ccc6782eaa810548df1578dd7b775d9ec5144cda48851ca70f91b49539e2c65e63123e387db3fe4d62f285cb543e10accf800ec669e75955057210affe8da69741ebfed025c42094daba53febb562f0676f7567880b0689cd5415db910979658ebe6b02dc8e020a124b675856944132ef44760ecfc1bd1cf1d8028f277f2697744296191804a9f3c519e2397a83aacf41413a0905a66b76c94e2501f0c76fb4f0cd912de85174df56f2c3ed7716b47be4374de22bd997c234011befec650d1cc761c99663f2b7ead336370f5e2c427da4dd13e1fc860894212620d6b54bc0f37b33e4b8cf9c26bde29b132bcb9cc59d318f2580bbb5fa3500be1319e0930eb69d70c92918ec03bf98492d3eb6edc7c984e7e21fb364ea3ea0e0"
        let proofhash = ethers.utils.solidityKeccak256(['bytes', 'address'], [ethers.utils.keccak256(aggrProof), aggregator1.address]);
        await chain_aggr1.submitProofHash(batchIndex, proofhash);

        // commit proof
        const batchHeader1 = "0x00000000000000000100000000000000000000000000000000539b0378e8b39bb27d931ad3885de2310d91baa2a828badbb84ec2fa69550b585aaeb6101a47fc16866e80d77ffe090b6a7b3cf7d988be981646ab6aedfa2c42"
        const prevStateRoot = "0x08d535cc60f40af5dd3b31e0998d7567c2d568b224bed2ba26070aeb078d1339"
        const postStateRoot = "0x2b3219c3d89d50b5aa4e56743c4e22501d34b885e468365ba3b1cc818297db74"
        const withdrawRoot = "0x0000000000000000000000000000000000000000000000000000000000000000"
        for (let i = 0; i < 20 - 1; i++ ) {
            await hre.network.provider.request({
                method: "evm_mine",
            });
        }
        await chain_aggr1.finalizeBatchWithProof(batchHeader1, prevStateRoot, postStateRoot, withdrawRoot, aggrProof);

    });

    it.skip('two aggregator: one commit two proof hashs and submit two proofs', async () => {
        // add sequencer/prover
        await chain_deployer.addSequencer(aggregator1.address);
        await chain_deployer.addSequencer(aggregator2.address);
        await chain_deployer.addProver(aggregator1.address);
        await chain_deployer.addProver(aggregator2.address);
        
        // import genesis 
        const genisis_parentBatchHeader = "0x0000000000000000000000000000000000000000000000000061a5de00a189b490960523626c576089401f5413e4ca6f5fe5f16004d764ccd00000000000000000000000000000000000000000000000000000000000000000"
        const _stateRoot = "0x08d535cc60f40af5dd3b31e0998d7567c2d568b224bed2ba26070aeb078d1339"
        await chain_deployer.importGenesisBatch(genisis_parentBatchHeader, _stateRoot);
        
        // Sequence Batches
        const version = 0
        const parentBatchHeader_1 = "0x0000000000000000000000000000000000000000000000000061a5de00a189b490960523626c576089401f5413e4ca6f5fe5f16004d764ccd00000000000000000000000000000000000000000000000000000000000000000"
        const parentBatchHeader_2 = "0x00000000000000000100000000000000000000000000000000539b0378e8b39bb27d931ad3885de2310d91baa2a828badbb84ec2fa69550b585aaeb6101a47fc16866e80d77ffe090b6a7b3cf7d988be981646ab6aedfa2c42"
        var chunks
        const txs: Array<Uint8Array> = [];
        for (let i = 0; i < 20; i++) {
        const tx = new Uint8Array(4 + 100);
        let offset = 3;
        for (let x = 100; x > 0; x = Math.floor(x / 256)) {
            tx[offset] = x % 256;
            offset -= 1;
        }
        tx.fill(1, 4);
        txs.push(tx);
        }
        const chunk = new Uint8Array(1 + 60 * 1);
        chunk[0] = 1;
        for (let i = 0; i < 1; i++) {
        chunk[1 + i * 60 + 57] = 20;
        }
        chunks = [];
        for (let i = 0; i < 1; i++) {
        const txsInChunk: Array<Uint8Array> = [];
        for (let j = 0; j < 1; j++) {
            txsInChunk.push(concat(txs));
        }
        chunks.push(concat([chunk, concat(txsInChunk)]));
        }   
        await chain_aggr1.commitBatch(version, parentBatchHeader_1, chunks, "0x");
        await chain_aggr2.commitBatch(version, parentBatchHeader_2, chunks, "0x");

        // commit proof hash
        const batchIndex = 1
        const aggrProof = "0x0000000000000000000000000000000000000000009a53f7e9cfe41b13c3841c0000000000000000000000000000000000000000000a71e92dc0f7ec70d961a90000000000000000000000000000000000000000000011b9320e9a71777058090000000000000000000000000000000000000000002f780de098a2e133ac8e83000000000000000000000000000000000000000000cc97df3098b8c6acf22aae000000000000000000000000000000000000000000000905a9902c8cee0a218a000000000000000000000000000000000000000000bdab3b1b789280ac90d6740000000000000000000000000000000000000000009f6c2cb25710d30e52a392000000000000000000000000000000000000000000001f1c84d3acd6ec70d8bf00000000000000000000000000000000000000000094c031e6f78746862ac083000000000000000000000000000000000000000000b93466a296a4bdb0e5d5ae000000000000000000000000000000000000000000000bdb3de2c4423da15ae302f05a845ad6f200414ebc1613ba889275beecee980e9106d39e11f981f4a50c285faad90f3006c290623eace0bcb52c8f679e92f2b9152d13550ac6a70fbfa120f2b00bf31dafdd764a814d5fafcb93e6cb0d8f2018d1dbf3d17496a4e3d4d30ed399a8532b8956f668609db8767c073034f4fba3baa6249dad8382ea7aace40b3dc91da9aefb560f861f9e4905f5987a5dccd5ce5ea1005fa14a921a0a00e32f3da3e6b13ed176f29839e2556d48ad5534c50d5aad0b2b03df4913c8f002ab1c65005c5c001d5d299cb2e351def07da544bd51adb6d1b614630012a314f5de0b8abfe3cc752951df2a095757842c0f7259ff608cc8896f2990f847ef6577642b028e2bc10b0110afd48eccfe2a3e018739e7b67e930437edf818bfc49deaa9106019364a8cd1c288713c26002be01b252c742963d092b6419c64afb9ac71f6000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020881a8548f4145f7e0762273f164a399c5d5625f4ffeac6270e421e62c1aab150240131d5e547c732e96c4770bfb77fc33eabb48c288390d39ff8b62155040ba1b56eb3b12b5b2f0276eecb4316c625f87bc0c8db14417ff536bf67551e5c79b2d882922c93912c1238d11deb1508881ee95ef019339501fd9d0911e438d37dd0d13fcd3afd30787ade50f2aa952a08cf18c41e5209026b1adc738439734ba990e7ad4df140e5c6e6270af5f01fce9fc4f6218b5df5fdde0902167822c49157a01e54d3e14f0e5cf132988e141cd06c96ad514a7722f4c2fbd972eb3e1039c071d0242cc4c3674899d81b3b7a1f790bad42155afa34b9b9860c4f26c2007f83a2de3fed404a3d8ce62e4fcd5cb0a68be6ae4fb0c5f5499ed8648ef4008a0de4003f7f0496585f955a236ce6411b57b5affd630840306536e73c3f271d201352d09ca8109bf17fd6b500705448cd96437eae6c6da52d0d6ce1e9bdbc951d13a48142695006219a2498e9af06fc8c31a952ec26bfc34e80821e60561226ebf6c83049e064283ed6c432ed0c278c03b949eeb1f3cd0636d851cc0f69b1d8895a4af111a86d992a37a8121fadbf600e2876e3553d062f2715e27d92fa2c7359077af18686295314c7992037b9caa277ee7c86b07d6bdae816ff47e76f6fa167b81bd0250c7342c813eabb8787f8de77a67f9ed7201060fbe027d49af30b7e38e8f2d00000000000000000000000000000000000000000000000000000000000000010f88cd2c11e8cffa8cadfdfc19ce258d47e16d4e0761b0637aaec6b1f985b6971cde71f7f4608b9a7c199b3ec4c7649cebff13561b66bef60252d2c04e2de6251e0465d8d2995cc2cf08a99200756dfaf1c108226eea7e663984ded302ebd2c902eade1d83644ed50425517e9741e9ea243e1aec8c23ca4e5efd7b084f02877d0983f78bb96761d4a687bea7c4523f674816bd9b03bd10cfb93d9a54cc0965d62cfcc0ff4b8c3ccc6782eaa810548df1578dd7b775d9ec5144cda48851ca70f91b49539e2c65e63123e387db3fe4d62f285cb543e10accf800ec669e75955057210affe8da69741ebfed025c42094daba53febb562f0676f7567880b0689cd5415db910979658ebe6b02dc8e020a124b675856944132ef44760ecfc1bd1cf1d8028f277f2697744296191804a9f3c519e2397a83aacf41413a0905a66b76c94e2501f0c76fb4f0cd912de85174df56f2c3ed7716b47be4374de22bd997c234011befec650d1cc761c99663f2b7ead336370f5e2c427da4dd13e1fc860894212620d6b54bc0f37b33e4b8cf9c26bde29b132bcb9cc59d318f2580bbb5fa3500be1319e0930eb69d70c92918ec03bf98492d3eb6edc7c984e7e21fb364ea3ea0e0"
        let proofhash1 = ethers.utils.solidityKeccak256(['bytes', 'address'], [ethers.utils.keccak256(aggrProof), aggregator1.address]);
        let proofhash2 = ethers.utils.solidityKeccak256(['bytes', 'address'], [ethers.utils.keccak256(aggrProof), aggregator2.address]);
        await chain_aggr1.submitProofHash(batchIndex, proofhash1);
        await chain_aggr2.submitProofHash(batchIndex, proofhash2);

        // commit proof
        const batchHeader1 = "0x00000000000000000100000000000000000000000000000000539b0378e8b39bb27d931ad3885de2310d91baa2a828badbb84ec2fa69550b585aaeb6101a47fc16866e80d77ffe090b6a7b3cf7d988be981646ab6aedfa2c42"
        const prevStateRoot = "0x08d535cc60f40af5dd3b31e0998d7567c2d568b224bed2ba26070aeb078d1339"
        const postStateRoot = "0x2b3219c3d89d50b5aa4e56743c4e22501d34b885e468365ba3b1cc818297db74"
        const withdrawRoot = "0x0000000000000000000000000000000000000000000000000000000000000000"
        for (let i = 0; i <20 - 1; i++ ) {
            await hre.network.provider.request({
                method: "evm_mine",
            });
        }
        await chain_aggr1.finalizeBatchWithProof(batchHeader1, prevStateRoot, postStateRoot, withdrawRoot, aggrProof);
        await chain_aggr2.finalizeBatchWithProof(batchHeader1, prevStateRoot, postStateRoot, withdrawRoot, aggrProof);

    });

    it('two aggregator: two commits one proof hash one proof ', async () => {
        // add sequencer/prover
        await chain_deployer.addSequencer(aggregator1.address);
        await chain_deployer.addSequencer(aggregator2.address);
        await chain_deployer.addProver(aggregator1.address);
        await chain_deployer.addProver(aggregator2.address);
        
        // import genesis 
        const genisis_parentBatchHeader = "0x0000000000000000000000000000000000000000000000000061a5de00a189b490960523626c576089401f5413e4ca6f5fe5f16004d764ccd00000000000000000000000000000000000000000000000000000000000000000"
        const _stateRoot = "0x08d535cc60f40af5dd3b31e0998d7567c2d568b224bed2ba26070aeb078d1339"
        await chain_deployer.importGenesisBatch(genisis_parentBatchHeader, _stateRoot);
        
        // Sequence Batches
        const version = 0
        const parentBatchHeader_1 = "0x0000000000000000000000000000000000000000000000000061a5de00a189b490960523626c576089401f5413e4ca6f5fe5f16004d764ccd00000000000000000000000000000000000000000000000000000000000000000"
        const parentBatchHeader_2 = "0x00000000000000000100000000000000000000000000000000539b0378e8b39bb27d931ad3885de2310d91baa2a828badbb84ec2fa69550b585aaeb6101a47fc16866e80d77ffe090b6a7b3cf7d988be981646ab6aedfa2c42"
        var chunks
        const txs: Array<Uint8Array> = [];
        for (let i = 0; i < 20; i++) {
        const tx = new Uint8Array(4 + 100);
        let offset = 3;
        for (let x = 100; x > 0; x = Math.floor(x / 256)) {
            tx[offset] = x % 256;
            offset -= 1;
        }
        tx.fill(1, 4);
        txs.push(tx);
        }
        const chunk = new Uint8Array(1 + 60 * 1);
        chunk[0] = 1;
        for (let i = 0; i < 1; i++) {
        chunk[1 + i * 60 + 57] = 20;
        }
        chunks = [];
        for (let i = 0; i < 1; i++) {
        const txsInChunk: Array<Uint8Array> = [];
        for (let j = 0; j < 1; j++) {
            txsInChunk.push(concat(txs));
        }
        chunks.push(concat([chunk, concat(txsInChunk)]));
        }   
        await chain_aggr1.commitBatch(version, parentBatchHeader_1, chunks, "0x");
        await chain_aggr2.commitBatch(version, parentBatchHeader_2, chunks, "0x");

        // commit proof hash
        const batchIndex1 = 1
        const batchIndex2 = 2
        const aggrProof = "0x0000000000000000000000000000000000000000009a53f7e9cfe41b13c3841c0000000000000000000000000000000000000000000a71e92dc0f7ec70d961a90000000000000000000000000000000000000000000011b9320e9a71777058090000000000000000000000000000000000000000002f780de098a2e133ac8e83000000000000000000000000000000000000000000cc97df3098b8c6acf22aae000000000000000000000000000000000000000000000905a9902c8cee0a218a000000000000000000000000000000000000000000bdab3b1b789280ac90d6740000000000000000000000000000000000000000009f6c2cb25710d30e52a392000000000000000000000000000000000000000000001f1c84d3acd6ec70d8bf00000000000000000000000000000000000000000094c031e6f78746862ac083000000000000000000000000000000000000000000b93466a296a4bdb0e5d5ae000000000000000000000000000000000000000000000bdb3de2c4423da15ae302f05a845ad6f200414ebc1613ba889275beecee980e9106d39e11f981f4a50c285faad90f3006c290623eace0bcb52c8f679e92f2b9152d13550ac6a70fbfa120f2b00bf31dafdd764a814d5fafcb93e6cb0d8f2018d1dbf3d17496a4e3d4d30ed399a8532b8956f668609db8767c073034f4fba3baa6249dad8382ea7aace40b3dc91da9aefb560f861f9e4905f5987a5dccd5ce5ea1005fa14a921a0a00e32f3da3e6b13ed176f29839e2556d48ad5534c50d5aad0b2b03df4913c8f002ab1c65005c5c001d5d299cb2e351def07da544bd51adb6d1b614630012a314f5de0b8abfe3cc752951df2a095757842c0f7259ff608cc8896f2990f847ef6577642b028e2bc10b0110afd48eccfe2a3e018739e7b67e930437edf818bfc49deaa9106019364a8cd1c288713c26002be01b252c742963d092b6419c64afb9ac71f6000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020881a8548f4145f7e0762273f164a399c5d5625f4ffeac6270e421e62c1aab150240131d5e547c732e96c4770bfb77fc33eabb48c288390d39ff8b62155040ba1b56eb3b12b5b2f0276eecb4316c625f87bc0c8db14417ff536bf67551e5c79b2d882922c93912c1238d11deb1508881ee95ef019339501fd9d0911e438d37dd0d13fcd3afd30787ade50f2aa952a08cf18c41e5209026b1adc738439734ba990e7ad4df140e5c6e6270af5f01fce9fc4f6218b5df5fdde0902167822c49157a01e54d3e14f0e5cf132988e141cd06c96ad514a7722f4c2fbd972eb3e1039c071d0242cc4c3674899d81b3b7a1f790bad42155afa34b9b9860c4f26c2007f83a2de3fed404a3d8ce62e4fcd5cb0a68be6ae4fb0c5f5499ed8648ef4008a0de4003f7f0496585f955a236ce6411b57b5affd630840306536e73c3f271d201352d09ca8109bf17fd6b500705448cd96437eae6c6da52d0d6ce1e9bdbc951d13a48142695006219a2498e9af06fc8c31a952ec26bfc34e80821e60561226ebf6c83049e064283ed6c432ed0c278c03b949eeb1f3cd0636d851cc0f69b1d8895a4af111a86d992a37a8121fadbf600e2876e3553d062f2715e27d92fa2c7359077af18686295314c7992037b9caa277ee7c86b07d6bdae816ff47e76f6fa167b81bd0250c7342c813eabb8787f8de77a67f9ed7201060fbe027d49af30b7e38e8f2d00000000000000000000000000000000000000000000000000000000000000010f88cd2c11e8cffa8cadfdfc19ce258d47e16d4e0761b0637aaec6b1f985b6971cde71f7f4608b9a7c199b3ec4c7649cebff13561b66bef60252d2c04e2de6251e0465d8d2995cc2cf08a99200756dfaf1c108226eea7e663984ded302ebd2c902eade1d83644ed50425517e9741e9ea243e1aec8c23ca4e5efd7b084f02877d0983f78bb96761d4a687bea7c4523f674816bd9b03bd10cfb93d9a54cc0965d62cfcc0ff4b8c3ccc6782eaa810548df1578dd7b775d9ec5144cda48851ca70f91b49539e2c65e63123e387db3fe4d62f285cb543e10accf800ec669e75955057210affe8da69741ebfed025c42094daba53febb562f0676f7567880b0689cd5415db910979658ebe6b02dc8e020a124b675856944132ef44760ecfc1bd1cf1d8028f277f2697744296191804a9f3c519e2397a83aacf41413a0905a66b76c94e2501f0c76fb4f0cd912de85174df56f2c3ed7716b47be4374de22bd997c234011befec650d1cc761c99663f2b7ead336370f5e2c427da4dd13e1fc860894212620d6b54bc0f37b33e4b8cf9c26bde29b132bcb9cc59d318f2580bbb5fa3500be1319e0930eb69d70c92918ec03bf98492d3eb6edc7c984e7e21fb364ea3ea0e0"
        let proofhash1 = ethers.utils.solidityKeccak256(['bytes', 'address'], [ethers.utils.keccak256(aggrProof), aggregator1.address]);
        let proofhash2 = ethers.utils.solidityKeccak256(['bytes', 'address'], [ethers.utils.keccak256(aggrProof), aggregator2.address]);
        await chain_aggr1.submitProofHash(batchIndex1, proofhash1);
        await chain_aggr2.submitProofHash(batchIndex2, proofhash2);

        // commit proof
        const batchHeader1 = "0x00000000000000000100000000000000000000000000000000539b0378e8b39bb27d931ad3885de2310d91baa2a828badbb84ec2fa69550b585aaeb6101a47fc16866e80d77ffe090b6a7b3cf7d988be981646ab6aedfa2c42"
        const batchHeader2 = "0x00000000000000000200000000000000000000000000000000f881c891b5349d1c5e706f7d9c74e1cbd85782bfe2801ae3d2be306143e69198aa8181f04f8e305328a6117fa6bc13fa2093a3c4c990c5281df95a1cb85ca18f"
        const prevStateRoot = "0x08d535cc60f40af5dd3b31e0998d7567c2d568b224bed2ba26070aeb078d1339"
        const postStateRoot = "0x2b3219c3d89d50b5aa4e56743c4e22501d34b885e468365ba3b1cc818297db74"
        const withdrawRoot = "0x0000000000000000000000000000000000000000000000000000000000000000"
        for (let i = 0; i <20 - 1; i++ ) {
            await hre.network.provider.request({
                method: "evm_mine",
            });
        }
        await chain_aggr1.finalizeBatchWithProof(batchHeader1, prevStateRoot, postStateRoot, withdrawRoot, aggrProof);
        await chain_aggr2.finalizeBatchWithProof(batchHeader2, prevStateRoot, postStateRoot, withdrawRoot, aggrProof);

    });

    it.skip('one aggregator: one commit, check reward and punish', async () => {
        // Mock OpsideSlot contract, Couldn't do this test now

        // add sequencer/prover
        await chain_deployer.addSequencer(aggregator1.address);
        await chain_deployer.addProver(aggregator1.address);
        
        // import genesis 
        const genisis_parentBatchHeader = "0x0000000000000000000000000000000000000000000000000061a5de00a189b490960523626c576089401f5413e4ca6f5fe5f16004d764ccd00000000000000000000000000000000000000000000000000000000000000000"
        const _stateRoot = "0x08d535cc60f40af5dd3b31e0998d7567c2d568b224bed2ba26070aeb078d1339"
        await chain_deployer.importGenesisBatch(genisis_parentBatchHeader, _stateRoot);
        
        // Sequence Batches
        const version = 0
        const parentBatchHeader = "0x0000000000000000000000000000000000000000000000000061a5de00a189b490960523626c576089401f5413e4ca6f5fe5f16004d764ccd00000000000000000000000000000000000000000000000000000000000000000"
        var chunks
        const txs: Array<Uint8Array> = [];
        for (let i = 0; i < 20; i++) {
        const tx = new Uint8Array(4 + 100);
        let offset = 3;
        for (let x = 100; x > 0; x = Math.floor(x / 256)) {
            tx[offset] = x % 256;
            offset -= 1;
        }
        tx.fill(1, 4);
        txs.push(tx);
        }
        const chunk = new Uint8Array(1 + 60 * 1);
        chunk[0] = 1;
        for (let i = 0; i < 1; i++) {
        chunk[1 + i * 60 + 57] = 20;
        }
        chunks = [];
        for (let i = 0; i < 1; i++) {
        const txsInChunk: Array<Uint8Array> = [];
        for (let j = 0; j < 1; j++) {
            txsInChunk.push(concat(txs));
        }
        chunks.push(concat([chunk, concat(txsInChunk)]));
        }   
        await chain_aggr1.commitBatch(version, parentBatchHeader, chunks, "0x");

        // commit proof hash
        const batchIndex = 1
        const aggrProof = "0x0000000000000000000000000000000000000000009a53f7e9cfe41b13c3841c0000000000000000000000000000000000000000000a71e92dc0f7ec70d961a90000000000000000000000000000000000000000000011b9320e9a71777058090000000000000000000000000000000000000000002f780de098a2e133ac8e83000000000000000000000000000000000000000000cc97df3098b8c6acf22aae000000000000000000000000000000000000000000000905a9902c8cee0a218a000000000000000000000000000000000000000000bdab3b1b789280ac90d6740000000000000000000000000000000000000000009f6c2cb25710d30e52a392000000000000000000000000000000000000000000001f1c84d3acd6ec70d8bf00000000000000000000000000000000000000000094c031e6f78746862ac083000000000000000000000000000000000000000000b93466a296a4bdb0e5d5ae000000000000000000000000000000000000000000000bdb3de2c4423da15ae302f05a845ad6f200414ebc1613ba889275beecee980e9106d39e11f981f4a50c285faad90f3006c290623eace0bcb52c8f679e92f2b9152d13550ac6a70fbfa120f2b00bf31dafdd764a814d5fafcb93e6cb0d8f2018d1dbf3d17496a4e3d4d30ed399a8532b8956f668609db8767c073034f4fba3baa6249dad8382ea7aace40b3dc91da9aefb560f861f9e4905f5987a5dccd5ce5ea1005fa14a921a0a00e32f3da3e6b13ed176f29839e2556d48ad5534c50d5aad0b2b03df4913c8f002ab1c65005c5c001d5d299cb2e351def07da544bd51adb6d1b614630012a314f5de0b8abfe3cc752951df2a095757842c0f7259ff608cc8896f2990f847ef6577642b028e2bc10b0110afd48eccfe2a3e018739e7b67e930437edf818bfc49deaa9106019364a8cd1c288713c26002be01b252c742963d092b6419c64afb9ac71f6000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020881a8548f4145f7e0762273f164a399c5d5625f4ffeac6270e421e62c1aab150240131d5e547c732e96c4770bfb77fc33eabb48c288390d39ff8b62155040ba1b56eb3b12b5b2f0276eecb4316c625f87bc0c8db14417ff536bf67551e5c79b2d882922c93912c1238d11deb1508881ee95ef019339501fd9d0911e438d37dd0d13fcd3afd30787ade50f2aa952a08cf18c41e5209026b1adc738439734ba990e7ad4df140e5c6e6270af5f01fce9fc4f6218b5df5fdde0902167822c49157a01e54d3e14f0e5cf132988e141cd06c96ad514a7722f4c2fbd972eb3e1039c071d0242cc4c3674899d81b3b7a1f790bad42155afa34b9b9860c4f26c2007f83a2de3fed404a3d8ce62e4fcd5cb0a68be6ae4fb0c5f5499ed8648ef4008a0de4003f7f0496585f955a236ce6411b57b5affd630840306536e73c3f271d201352d09ca8109bf17fd6b500705448cd96437eae6c6da52d0d6ce1e9bdbc951d13a48142695006219a2498e9af06fc8c31a952ec26bfc34e80821e60561226ebf6c83049e064283ed6c432ed0c278c03b949eeb1f3cd0636d851cc0f69b1d8895a4af111a86d992a37a8121fadbf600e2876e3553d062f2715e27d92fa2c7359077af18686295314c7992037b9caa277ee7c86b07d6bdae816ff47e76f6fa167b81bd0250c7342c813eabb8787f8de77a67f9ed7201060fbe027d49af30b7e38e8f2d00000000000000000000000000000000000000000000000000000000000000010f88cd2c11e8cffa8cadfdfc19ce258d47e16d4e0761b0637aaec6b1f985b6971cde71f7f4608b9a7c199b3ec4c7649cebff13561b66bef60252d2c04e2de6251e0465d8d2995cc2cf08a99200756dfaf1c108226eea7e663984ded302ebd2c902eade1d83644ed50425517e9741e9ea243e1aec8c23ca4e5efd7b084f02877d0983f78bb96761d4a687bea7c4523f674816bd9b03bd10cfb93d9a54cc0965d62cfcc0ff4b8c3ccc6782eaa810548df1578dd7b775d9ec5144cda48851ca70f91b49539e2c65e63123e387db3fe4d62f285cb543e10accf800ec669e75955057210affe8da69741ebfed025c42094daba53febb562f0676f7567880b0689cd5415db910979658ebe6b02dc8e020a124b675856944132ef44760ecfc1bd1cf1d8028f277f2697744296191804a9f3c519e2397a83aacf41413a0905a66b76c94e2501f0c76fb4f0cd912de85174df56f2c3ed7716b47be4374de22bd997c234011befec650d1cc761c99663f2b7ead336370f5e2c427da4dd13e1fc860894212620d6b54bc0f37b33e4b8cf9c26bde29b132bcb9cc59d318f2580bbb5fa3500be1319e0930eb69d70c92918ec03bf98492d3eb6edc7c984e7e21fb364ea3ea0e0"
        let proofhash = ethers.utils.solidityKeccak256(['bytes', 'address'], [ethers.utils.keccak256(aggrProof), aggregator1.address]);
        await chain_aggr1.submitProofHash(batchIndex, proofhash);

        // commit proof
        const batchHeader1 = "0x00000000000000000100000000000000000000000000000000539b0378e8b39bb27d931ad3885de2310d91baa2a828badbb84ec2fa69550b585aaeb6101a47fc16866e80d77ffe090b6a7b3cf7d988be981646ab6aedfa2c42"
        const prevStateRoot = "0x08d535cc60f40af5dd3b31e0998d7567c2d568b224bed2ba26070aeb078d1339"
        const postStateRoot = "0x2b3219c3d89d50b5aa4e56743c4e22501d34b885e468365ba3b1cc818297db74"
        const withdrawRoot = "0x0000000000000000000000000000000000000000000000000000000000000000"
        for (let i = 0; i < 20 - 1; i++ ) {
            await hre.network.provider.request({
                method: "evm_mine",
            });
        }
        
        await chain_aggr1.finalizeBatchWithProof(batchHeader1, prevStateRoot, postStateRoot, withdrawRoot, aggrProof);

    });
});
