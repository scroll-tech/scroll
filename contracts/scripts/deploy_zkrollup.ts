/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const ProxyAdmin = await ethers.getContractAt("ProxyAdmin", addressFile.get("ProxyAdmin"), deployer);

  if (!addressFile.get("ZKRollup.verifier")) {
    console.log(">> Deploy RollupVerifier");
    const RollupVerifier = await ethers.getContractFactory("RollupVerifier", deployer);
    const verifier = await RollupVerifier.deploy();
    console.log(`>> waiting for transaction: ${verifier.deployTransaction.hash}`);
    await verifier.deployed();
    console.log(`✅ RollupVerifier deployed at ${verifier.address}`);
    addressFile.set("ZKRollup.verifier", verifier.address);
  }

  if (!addressFile.get("ZKRollup.implementation")) {
    console.log(">> Deploy ZKRollup implementation");
    const ZKRollup = await ethers.getContractFactory("ZKRollup", {
      libraries: {
        RollupVerifier: addressFile.get("ZKRollup.verifier"),
      },
      signer: deployer,
    });
    const impl = await ZKRollup.deploy();
    console.log(`>> waiting for transaction: ${impl.deployTransaction.hash}`);
    await impl.deployed();
    console.log(`✅ ZKRollup implementation deployed at ${impl.address}`);
    addressFile.set("ZKRollup.implementation", impl.address);
  }

  const impl = addressFile.get("ZKRollup.implementation") as string;

  if (!addressFile.get("ZKRollup.proxy")) {
    console.log(">> Deploy ZKRollup proxy");
    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);
    const proxy = await TransparentUpgradeableProxy.deploy(impl, ProxyAdmin.address, "0x");
    console.log(`>> waiting for transaction: ${proxy.deployTransaction.hash}`);
    await proxy.deployed();
    console.log(`✅ ZKRollup proxy deployed at ${proxy.address}`);
    addressFile.set("ZKRollup.proxy", proxy.address);
  }

  // Export contract address to testnet.
  console.log(`testnet-export: ${addressFile.get("ZKRollup.implementation")};${addressFile.get("ZKRollup.proxy")}`);
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
