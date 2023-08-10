/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { expect } from "chai";
import { BigNumber, BigNumberish, constants } from "ethers";
import { concat, RLP } from "ethers/lib/utils";
import { ethers } from "hardhat";
import { L1BlockContainer } from "../typechain";

interface IImportTestConfig {
  hash: string;
  parentHash: string;
  uncleHash: string;
  coinbase: string;
  stateRoot: string;
  transactionsRoot: string;
  receiptsRoot: string;
  logsBloom: string;
  difficulty: BigNumberish;
  blockHeight: number;
  gasLimit: BigNumberish;
  gasUsed: BigNumberish;
  blockTimestamp: number;
  extraData: string;
  mixHash: string;
  blockNonce: string;
  baseFee: BigNumberish;
}

const testcases: Array<IImportTestConfig> = [
  {
    hash: "0x02250e97ef862444dd1d70acbe925c289bb2acf20a808cb8f4d1409d3adcfa1b",
    parentHash: "0x95e612b2a734f5a8c6aad3f6662b18f983ce8b653854d7c307bf999d9be323af",
    uncleHash: "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
    coinbase: "0x690b9a9e9aa1c9db991c7721a92d351db4fac990",
    stateRoot: "0x8d77db2a63cee63ae6d793f839a7513dfc50194f325b96a5326d724f5dc16320",
    transactionsRoot: "0xe4ce5f0e2fc5fd8a7ad55c2a31c522ded4054b89065c627d26230b45cd585fed",
    receiptsRoot: "0x10b2f34da3e6a1db9498ab36bb17b063763b8eb33492ccc621491b33bcb62bdd",
    logsBloom:
      "0x18b80159addab073ac340045c4ef982442653840c8074a50159bd9626ae0590740d07273d0c859005b634059c8ca9bb18364573e7ebe79a40aa08225942370c3dc6c0af2ea33cba07900961de2b011aabb8024270d4626d1028a2f0dcd780c60ce933b169b02c8c329c18b000aaf08c98245d8ad949e7d61102d5516489fa924f390c3a71642d7e6044c85a20952568d60cf24c38baff04c244b10eac87a6da8bb32c1535ea2613064a246d598c02444624a8d5a1b201a4270a7868a97aa4530838c2e7a192a88e329daf0334c728b7c057f684f1d28c07d0d2c1dc63868a1088010ae0b661073142e468ae062151e00e5108400e1a99c4111153828610874bb",
    difficulty: "0x0",
    blockHeight: 0xf766a8,
    gasLimit: "0x1c9c380",
    gasUsed: "0xe6f194",
    blockTimestamp: 0x639f69e3,
    extraData: "0x406275696c64657230783639",
    mixHash: "0xc1e37ce2b7ece4556ec87ea6d420a1a3610d49c58dfccec6998222fbf9cd64a2",
    blockNonce: "0x0000000000000000",
    baseFee: "0x2b96fa5cc",
  },
  {
    hash: "0x2da4bf7cef55d6207af2095db5543df16acbd95dc66eef02d9764277c5b0895d",
    parentHash: "0xde18012932b21820fbb48ef85b46774873383e75b062bc0c6a4761fbe87bad13",
    uncleHash: "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
    coinbase: "0x690b9a9e9aa1c9db991c7721a92d351db4fac990",
    stateRoot: "0x1f101f54c3df5630c9d45224c95d71a57479992e174cdbda0c4ada30e657a465",
    transactionsRoot: "0xc2b29438a5f55998879356cbc8006a90d2ba88a9841b3894c8da5840dd797f19",
    receiptsRoot: "0xbd3608b6af5464b446db44fd289a980f417447b31ff15dd6d48c72fc8f4fef8d",
    logsBloom:
      "0xd9e5f4f1e559388eb8193295ab2d3aab30c588d31e381c4060715d0a7ce607360b15d7a0d88e406c60135e0abcecd1d816c11f8cbbb2a80a9b4a00375d6cf356cb78f2934261ab09ea03df29dab5dbe4aefea506f7fd0eaa1a8b1fc8db5079613a49d80ca7e7997a20c7158399022c1dc9853f5b401b86587249fc96ca6fbc2dab1fdeb203ca258c94dd0bc821b38f9f60128591f3cd224c5c207b76b754e537bef8ebe731effae356235dd71bd7b5494bead124a8b5bb0ba02e46721d3ec3c20608880b1d35a17f6a1027d20c7b902e5d7b2ec8177b1aff9dcfbb4729d1e3201e78fa1b3c30e66a590cb5a7cac7afe0b0b1a6c94d5e39c9a20908358b805c81",
    difficulty: "0x0",
    blockHeight: 0xf766d8,
    gasLimit: "0x1c9c380",
    gasUsed: "0xf8adad",
    blockTimestamp: 0x639f6c23,
    extraData: "0x6275696c64657230783639",
    mixHash: "0x6066061b78b385483d960faa29ee40e79ea67769f5e697ecb70a0fce677804af",
    blockNonce: "0x0000000000000000",
    baseFee: "0x2aca8b608",
  },
  {
    hash: "0x4ddeee3e8d62e961080711e48d8083f164789e78cc90e4362c133063b566d64a",
    parentHash: "0x9d190c6d49352d628e321853967dd499d78c521daad73652ed1978db5652f58a",
    uncleHash: "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
    coinbase: "0xcd458d7f11023556cc9058f729831a038cb8df9c",
    stateRoot: "0x3620665f9d094aac16e0762b733e814f4e09177a232f85d406271b60e4f2b58f",
    transactionsRoot: "0x200f5acb65631c48c32c94ae95afe095134132939a01422da5c7c6d0e7f62cb3",
    receiptsRoot: "0xc140420782bc76ff326d18b13427c991e9434a554b9ae82bbf09cca7b6ae4036",
    logsBloom:
      "0x00a8cd20c1402037d2a51100c0895279410502288134d22313912bb7b42e504f850f417d9000000a41949b284b40210406019c0e28122d462c05c11120ac2c680800c0348066a23e7a9e042a9d20e4e0041114830d443160a46b5e02ec300d41330cf0652602140e1580b4c82d1228c000005be72c900f7152093d93ca4880062185952cacc6c8d1405a0c5823bb4284a04a44c92b41462c2420a870685438809a99850acc936c408c24e882a01517086a20a067a2e4e01a20e106078828706c7c00a0234e6830c80b911900291a134475208a4335ab0018a9048d4628186043303b722a79645a104c0e12a506404f45c428660a105d105010482852540b9a6b",
    difficulty: "0x2ae28b0d3154b6",
    blockHeight: 0xecb6fc,
    gasLimit: "0x1c9c30d",
    gasUsed: "0xb93955",
    blockTimestamp: 0x631d8207,
    extraData: "0x706f6f6c696e2e636f6d2050cabdd319bf3175",
    mixHash: "0x18d61005875e902e1bbba1045fd6701df170230c0ffb37f2e77fbc2051b987cf",
    blockNonce: "0xe8775f73466671e3",
    baseFee: "0x18c9de157",
  },
];

function encodeHeader(test: IImportTestConfig): string {
  return RLP.encode([
    test.parentHash,
    test.uncleHash,
    test.coinbase,
    test.stateRoot,
    test.transactionsRoot,
    test.receiptsRoot,
    test.logsBloom,
    BigNumber.from(test.difficulty).isZero() ? "0x" : BigNumber.from(test.difficulty).toHexString(),
    BigNumber.from(test.blockHeight).toHexString(),
    BigNumber.from(test.gasLimit).toHexString(),
    BigNumber.from(test.gasUsed).toHexString(),
    BigNumber.from(test.blockTimestamp).toHexString(),
    test.extraData,
    test.mixHash,
    test.blockNonce,
    BigNumber.from(test.baseFee).toHexString(),
  ]);
}

describe("L1BlockContainer", async () => {
  let container: L1BlockContainer;

  for (const test of testcases) {
    context(`import block[${test.hash}] height[${test.blockHeight}]`, async () => {
      beforeEach(async () => {
        const [deployer] = await ethers.getSigners();
        const L1BlockContainer = await ethers.getContractFactory("L1BlockContainer", deployer);
        container = await L1BlockContainer.deploy(deployer.address);

        const Whitelist = await ethers.getContractFactory("Whitelist", deployer);
        const whitelist = await Whitelist.deploy(deployer.address);
        await whitelist.updateWhitelistStatus([deployer.address], true);

        await container.updateWhitelist(whitelist.address);
      });

      it("should revert, when sender not allowed", async () => {
        const [, signer] = await ethers.getSigners();
        await container.initialize(
          test.parentHash,
          test.blockHeight - 1,
          test.blockTimestamp - 1,
          test.baseFee,
          test.stateRoot
        );

        await expect(container.connect(signer).importBlockHeader(constants.HashZero, [], false)).to.revertedWith(
          "Not whitelisted sender"
        );
      });

      it("should revert, when block hash mismatch", async () => {
        await container.initialize(
          test.parentHash,
          test.blockHeight - 1,
          test.blockTimestamp - 1,
          test.baseFee,
          test.stateRoot
        );
        const headerRLP = encodeHeader(test);
        await expect(container.importBlockHeader(test.parentHash, headerRLP, false)).to.revertedWith(
          "Block hash mismatch"
        );
      });

      it("should revert, when has extra bytes", async () => {
        await container.initialize(
          test.parentHash,
          test.blockHeight - 1,
          test.blockTimestamp - 1,
          test.baseFee,
          test.stateRoot
        );
        const headerRLP = encodeHeader(test);
        await expect(container.importBlockHeader(test.hash, concat([headerRLP, "0x00"]), false)).to.revertedWith(
          "Header RLP length mismatch"
        );
      });

      it("should revert, when parent not imported", async () => {
        await container.initialize(
          constants.HashZero,
          test.blockHeight - 1,
          test.blockTimestamp - 1,
          test.baseFee,
          test.stateRoot
        );
        const headerRLP = encodeHeader(test);
        await expect(container.importBlockHeader(test.hash, headerRLP, false)).to.revertedWith("Parent not imported");
      });

      it("should revert, when block height mismatch", async () => {
        await container.initialize(
          test.parentHash,
          test.blockHeight,
          test.blockTimestamp - 1,
          test.baseFee,
          test.stateRoot
        );
        const headerRLP = encodeHeader(test);
        await expect(container.importBlockHeader(test.hash, headerRLP, false)).to.revertedWith("Block height mismatch");
      });

      it("should revert, when parent block has larger timestamp", async () => {
        await container.initialize(
          test.parentHash,
          test.blockHeight - 1,
          test.blockTimestamp + 1,
          test.baseFee,
          test.stateRoot
        );
        const headerRLP = encodeHeader(test);
        await expect(container.importBlockHeader(test.hash, headerRLP, false)).to.revertedWith(
          "Parent block has larger timestamp"
        );
      });

      it(`should succeed`, async () => {
        await container.initialize(
          test.parentHash,
          test.blockHeight - 1,
          test.blockTimestamp - 1,
          test.baseFee,
          test.stateRoot
        );
        expect(await container.latestBlockHash()).to.eq(test.parentHash);
        const headerRLP = encodeHeader(test);
        await expect(container.importBlockHeader(test.hash, headerRLP, false))
          .to.emit(container, "ImportBlock")
          .withArgs(test.hash, test.blockHeight, test.blockTimestamp, test.baseFee, test.stateRoot);
        expect(await container.getStateRoot(test.hash)).to.eq(test.stateRoot);
        expect(await container.getBlockTimestamp(test.hash)).to.eq(test.blockTimestamp);
        expect(await container.latestBlockHash()).to.eq(test.hash);
        expect(await container.latestBaseFee()).to.eq(test.baseFee);
        expect(await container.latestBlockNumber()).to.eq(test.blockHeight);
        expect(await container.latestBlockTimestamp()).to.eq(test.blockTimestamp);
      });
    });
  }
});
