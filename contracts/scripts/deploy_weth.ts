/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  if (!addressFile.get("WETH")) {
    console.log(">> Deploy WETH");
    const WrappedEther = await ethers.getContractFactory("WrappedEther", deployer);
    const weth = await WrappedEther.deploy();
    console.log(`>> waiting for transaction: ${weth.deployTransaction.hash}`);
    await weth.deployed();
    console.log(`✅ WETH deployed at ${weth.address}`);
    addressFile.set("WETH", weth.address);
  }

  // Export contract address to testnet.
  console.log(`testnet-export: ${addressFile.get("WETH")}`);
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
