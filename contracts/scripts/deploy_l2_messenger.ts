/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const ProxyAdmin = await ethers.getContractAt("ProxyAdmin", addressFile.get("ProxyAdmin"), deployer);

  const queue = process.env.L2_MESSAGE_QUEUE_ADDR!;

  if (!addressFile.get("L2ScrollMessenger.implementation")) {
    console.log(">> Deploy L2ScrollMessenger implementation");
    const L2ScrollMessenger = await ethers.getContractFactory("L2ScrollMessenger", deployer);
    const impl = await L2ScrollMessenger.deploy(queue);
    console.log(`>> waiting for transaction: ${impl.deployTransaction.hash}`);
    await impl.deployed();
    console.log(`✅ L2ScrollMessenger implementation deployed at ${impl.address}`);
    addressFile.set("L2ScrollMessenger.implementation", impl.address);
  }

  const impl = addressFile.get("L2ScrollMessenger.implementation") as string;

  if (!addressFile.get("L2ScrollMessenger.proxy")) {
    console.log(">> Deploy L2ScrollMessenger proxy");
    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);
    const proxy = await TransparentUpgradeableProxy.deploy(impl, ProxyAdmin.address, "0x");
    console.log(`>> waiting for transaction: ${proxy.deployTransaction.hash}`);
    await proxy.deployed();
    console.log(`✅ L2ScrollMessenger proxy deployed at ${proxy.address}`);
    addressFile.set(`L2ScrollMessenger.proxy`, proxy.address);
  }

  // Export contract address to testnet.
  console.log(
    `testnet-export: ${addressFile.get("L2ScrollMessenger.implementation")};${addressFile.get(
      "L2ScrollMessenger.proxy"
    )}`
  );
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
