/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  if (!addressFile.get("ProxyAdmin")) {
    console.log(">> Deploy ProxyAdmin");
    const ProxyAdmin = await ethers.getContractFactory("ProxyAdmin", deployer);
    const proxyAdmin = await ProxyAdmin.deploy();
    console.log(`>> waiting for transaction: ${proxyAdmin.deployTransaction.hash}`);
    await proxyAdmin.deployed();
    console.log(`✅ ProxyAdmin deployed at ${proxyAdmin.address}`);
    addressFile.set("ProxyAdmin", proxyAdmin.address);
  }

  // Export contract address to testnet.
  console.log(`testnet-export: ${addressFile.get("ProxyAdmin")}`);
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
