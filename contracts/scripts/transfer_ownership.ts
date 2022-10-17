/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";

import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

dotenv.config();

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  if (process.env.CONTRACT_NAME === undefined) {
    throw new Error("env CONTRACT_NAME undefined");
  }
  const contractName = process.env.CONTRACT_NAME!;
  const contractAddress = addressFile.get(`${contractName}.proxy`) || addressFile.get(`${contractName}`);
  const Contract = await ethers.getContractAt(contractName, contractAddress, deployer);

  const owner = process.env.CONTRACT_OWNER || deployer.address;
  if ((await Contract.owner()).toLowerCase() !== owner.toLowerCase()) {
    const tx = await Contract.transferOwnership(owner);
    console.log(`${contractName} transfer ownership to ${owner}, hash: ${tx.hash}`);
    const receipt = await tx.wait();
    console.log(`✅ Done, gas used: ${receipt.gasUsed}`);
  }
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
