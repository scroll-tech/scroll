/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const CHAIN_ID_L2 = process.env.CHAIN_ID_L2 || "none";

  const ProxyAdmin = await ethers.getContractAt("ProxyAdmin", addressFile.get("ProxyAdmin"), deployer);

  if (!addressFile.get("ScrollChain.verifier")) {
    console.log(">> Deploy RollupVerifier");
    const RollupVerifier = await ethers.getContractFactory("RollupVerifier", deployer);
    const verifier = await RollupVerifier.deploy();
    console.log(`>> waiting for transaction: ${verifier.deployTransaction.hash}`);
    await verifier.deployed();
    console.log(`✅ RollupVerifier deployed at ${verifier.address}`);
    addressFile.set("ScrollChain.verifier", verifier.address);
  }

  if (!addressFile.get("ScrollChain.multiple_verifier")) {
    console.log(">> Deploy MultipleVersionRollupVerifier");
    const multipleRollupVerifier = await ethers.getContractFactory("MultipleVersionRollupVerifier", deployer);
    const verifier = await multipleRollupVerifier.deploy(addressFile.get("ScrollChain.verifier"));
    console.log(`>> waiting for transaction: ${verifier.deployTransaction.hash}`);
    await verifier.deployed();
    console.log(`✅ MultipleVersionRollupVerifier deployed at ${verifier.address}`);
    addressFile.set("ScrollChain.multiple_verifier", verifier.address);
  }

  if (!addressFile.get("ScrollChain.implementation")) {
    console.log(">> Deploy ScrollChain implementation");
    const ScrollChain = await ethers.getContractFactory("ScrollChain", {
      libraries: {},
      signer: deployer,
    });
    const impl = await ScrollChain.deploy(CHAIN_ID_L2);
    console.log(`>> waiting for transaction: ${impl.deployTransaction.hash}`);
    await impl.deployed();
    console.log(`✅ ScrollChain implementation deployed at ${impl.address}`);
    addressFile.set("ScrollChain.implementation", impl.address);
  }

  const impl = addressFile.get("ScrollChain.implementation") as string;
  if (!addressFile.get("ScrollChain.proxy")) {
    console.log(">> Deploy ScrollChain proxy");
    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);
    const proxy = await TransparentUpgradeableProxy.deploy(impl, ProxyAdmin.address, "0x");
    console.log(`>> waiting for transaction: ${proxy.deployTransaction.hash}`);
    await proxy.deployed();
    console.log(`✅ ScrollChain proxy deployed at ${proxy.address}`);
    addressFile.set("ScrollChain.proxy", proxy.address);
  }

  // Export contract address to testnet.
  console.log(
    `testnet-export: 
    ScrollChain.implementation: ${addressFile.get("ScrollChain.implementation")};
    ScrollChain.proxy: ${addressFile.get("ScrollChain.proxy")}`
  );
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
