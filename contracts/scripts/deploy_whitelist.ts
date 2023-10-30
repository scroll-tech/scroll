/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  if (!addressFile.get("Whitelist")) {
    console.log(">> Deploy ProxyAdmin");
    const whitelist = await ethers.getContractFactory("Whitelist", deployer);
    const _whitelist = await whitelist.deploy(deployer.address);
    console.log(`>> waiting for transaction: ${_whitelist.deployTransaction.hash}`);
    await _whitelist.deployed();
    console.log(`✅ Whitelist deployed at ${_whitelist.address}`);
    addressFile.set("Whitelist", _whitelist.address);
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
