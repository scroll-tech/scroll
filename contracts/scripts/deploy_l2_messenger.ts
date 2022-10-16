/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  if (!addressFile.get("L2ScrollMessenger")) {
    console.log(">> Deploy L2ScrollMessenger implementation");
    const L2ScrollMessenger = await ethers.getContractFactory("L2ScrollMessenger", deployer);
    const impl = await L2ScrollMessenger.deploy(deployer.address);
    console.log(`>> waiting for transaction: ${impl.deployTransaction.hash}`);
    await impl.deployed();
    console.log(`✅ L2ScrollMessenger implementation deployed at ${impl.address}`);
    addressFile.set("L2ScrollMessenger", impl.address);
  }

  // Export contract address to testnet.
  console.log(`testnet-export: ${addressFile.get("L2ScrollMessenger")}`);
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
