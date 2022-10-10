/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const ProxyAdmin = await ethers.getContractAt("ProxyAdmin", addressFile.get("ProxyAdmin"), deployer);

  if (!addressFile.get("L1ScrollMessenger.implementation")) {
    console.log(">> Deploy L1ScrollMessenger implementation");
    const L1ScrollMessenger = await ethers.getContractFactory("L1ScrollMessenger", deployer);
    const impl = await L1ScrollMessenger.deploy();
    console.log(`>> waiting for transaction: ${impl.deployTransaction.hash}`);
    await impl.deployed();
    console.log(`✅ L1ScrollMessenger implementation deployed at ${impl.address}`);
    addressFile.set("L1ScrollMessenger.implementation", impl.address);
  }

  const impl = addressFile.get("L1ScrollMessenger.implementation") as string;

  if (!addressFile.get("L1ScrollMessenger.proxy")) {
    console.log(">> Deploy L1ScrollMessenger proxy");
    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);
    const proxy = await TransparentUpgradeableProxy.deploy(impl, ProxyAdmin.address, "0x");
    console.log(`>> waiting for transaction: ${proxy.deployTransaction.hash}`);
    await proxy.deployed();
    console.log(`✅ L1ScrollMessenger proxy deployed at ${proxy.address}`);
    addressFile.set("L1ScrollMessenger.proxy", proxy.address);
  }

  // Export contract address to testnet.
  console.log(
    `testnet-export: ${addressFile.get("L1ScrollMessenger.implementation")};${addressFile.get(
      "L1ScrollMessenger.proxy"
    )}`
  );
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
