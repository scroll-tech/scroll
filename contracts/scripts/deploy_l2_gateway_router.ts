/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const ProxyAdmin = await ethers.getContractAt("ProxyAdmin", addressFile.get("ProxyAdmin"), deployer);

  if (!addressFile.get("L2GatewayRouter.implementation")) {
    console.log(">> Deploy L2GatewayRouter implementation");
    const L2GatewayRouter = await ethers.getContractFactory("L2GatewayRouter", deployer);
    const impl = await L2GatewayRouter.deploy();
    console.log(`>> waiting for transaction: ${impl.deployTransaction.hash}`);
    await impl.deployed();
    console.log(`✅ L2GatewayRouter implementation deployed at ${impl.address}`);
    addressFile.set("L2GatewayRouter.implementation", impl.address);
  }

  const impl = addressFile.get("L2GatewayRouter.implementation") as string;

  if (!addressFile.get("L2GatewayRouter.proxy")) {
    console.log(">> Deploy L2GatewayRouter proxy");
    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);
    const proxy = await TransparentUpgradeableProxy.deploy(impl, ProxyAdmin.address, "0x");
    console.log(`>> waiting for transaction: ${proxy.deployTransaction.hash}`);
    await proxy.deployed();
    console.log(`✅ L2GatewayRouter proxy deployed at ${proxy.address}`);
    addressFile.set("L2GatewayRouter.proxy", proxy.address);
  }

  // Export contract address to testnet.
  console.log(
    `testnet-export: ${addressFile.get("L2GatewayRouter.implementation")};${addressFile.get("L2GatewayRouter.proxy")}`
  );
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
