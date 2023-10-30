/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

dotenv.config();

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const L1GatewayRouter = await ethers.getContractAt(
    "L1GatewayRouter",
    addressFile.get("L1GatewayRouter.proxy"),
    deployer
  );

  const L1ETHGatewayAddress = addressFile.get("L1ETHGateway.proxy");
  const L1StandardERC20GatewayAddress = addressFile.get("L1StandardERC20Gateway.proxy");
  const tx = await L1GatewayRouter.initialize(L1ETHGatewayAddress, L1StandardERC20GatewayAddress);
  console.log("initialize L1GatewayRouter, hash:", tx.hash);
  const receipt = await tx.wait();
  console.log(`✅ Done, gas used: ${receipt.gasUsed}`);
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
