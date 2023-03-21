/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const CHAIN_ID_L2 = process.env.CHAIN_ID_L2 || "none";
  const MAX_TX_IN_BATCH = process.env.MAX_TX_IN_BATCH || 25;
  const PADDING_TX_HASH =
    process.env.PADDING_TX_HASH || "0xb5baa665b2664c3bfed7eb46e00ebc110ecf2ebd257854a9bf2b9dbc9b2c08f6";

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

  if (!addressFile.get("ScrollChain.implementation")) {
    console.log(">> Deploy ScrollChain implementation");
    const ScrollChain = await ethers.getContractFactory("ScrollChain", {
      libraries: {
        RollupVerifier: addressFile.get("ScrollChain.verifier"),
      },
      signer: deployer,
    });
    const impl = await ScrollChain.deploy(CHAIN_ID_L2, MAX_TX_IN_BATCH, PADDING_TX_HASH);
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
    `testnet-export: ${addressFile.get("ScrollChain.implementation")};${addressFile.get("ScrollChain.proxy")}`
  );
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
