/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";

import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

dotenv.config();

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const contractName = process.env.CONTRACT_NAME_TO_UPGRADE!;

  const ProxyAdmin = await ethers.getContractAt("ProxyAdmin", addressFile.get("ProxyAdmin"), deployer);
  const proxy = await ethers.getContractAt(contractName, addressFile.get(`${contractName}.proxy`), deployer);
  const contractImplAddress = addressFile.get(`${contractName}.implementation`);

  if ((await ProxyAdmin.getProxyImplementation(proxy.address)) !== contractImplAddress) {
    const tx = await ProxyAdmin.upgrade(proxy.address, contractImplAddress);
    console.log(`upgrade impl for ${contractName}, hash:`, tx.hash);
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
