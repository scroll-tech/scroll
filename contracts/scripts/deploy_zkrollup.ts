/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";
import poseidonUnit from "circomlib/src/poseidon_gencontract";

dotenv.config();

async function main() {
  const [deployer] = await ethers.getSigners();
  const PoseidonUnit2Address = "0x0000000000000000000000000000000000000000"
  // if (!PoseidonUnit2Address) {
  //   const Poseidon2Elements = new ethers.ContractFactory(
  //     poseidonUnit.generateABI(2),
  //     poseidonUnit.createCode(2),
  //     deployer
  //   );

  //   const poseidon = await Poseidon2Elements.deploy();
  //   console.log("Deploy PoseidonUnit2 contract, hash:", poseidon.deployTransaction.hash);
  //   const receipt = await poseidon.deployTransaction.wait();
  //   console.log(`✅ Deploy PoseidonUnit2 contract at: ${poseidon.address}, gas used: ${receipt.gasUsed}`);
  //   PoseidonUnit2Address = poseidon.address;
  // }

  // deploy ZkEvmVerifier
  const ZkEvmVerifier = await ethers.getContractFactory("ZkEvmVerifierV1", deployer);
  const _ZkEvmVerifier = await ZkEvmVerifier.deploy(PoseidonUnit2Address);
  console.log(`✅ Deploy ZkEvmVerifier contract at: ${_ZkEvmVerifier.address}`);
  // deploy MultipleVersionRollupVerifier
  const MultipleVersionRollupVerifier = await ethers.getContractFactory("MultipleVersionRollupVerifier", deployer);
  const _MultipleVersionRollupVerifier = await MultipleVersionRollupVerifier.deploy(_ZkEvmVerifier.address);
  console.log(`✅ Deploy _MultipleVersionRollupVerifier contract at: ${_MultipleVersionRollupVerifier.address}`);
  const addressFile = selectAddressFile(hre.network.name);
  addressFile.set("ZKRollup.implementation",_MultipleVersionRollupVerifier.address);

  const impl = addressFile.get("ZkRollup.implementation") as string;

  if (!addressFile.get("ZKRollup.proxy")) {
    console.log(`>> Deploy ZKRollup proxy`);
    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);
    const ProxyAdmin = await ethers.getContractAt("ProxyAdmin", addressFile.get("ProxyAdmin"), deployer);
    console.log(impl)
    const proxy = await TransparentUpgradeableProxy.deploy(impl, ProxyAdmin.address, "0x");
    console.log(`>> waiting for transaction: ${proxy.deployTransaction.hash}`);
    await proxy.deployed();
    console.log(`✅ ZKRollup proxy deployed at ${proxy.address}`);
    addressFile.set(`ZKRollup.proxy`, proxy.address);
  }
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
