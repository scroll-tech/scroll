/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

dotenv.config();

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const L2GatewayRouter = await ethers.getContractAt(
    "L2GatewayRouter",
    addressFile.get("L2GatewayRouter.proxy"),
    deployer
  );

  const L2ETHGatewayAddress = addressFile.get("L2ETHGateway.proxy");
  const L2StandardERC20GatewayAddress = addressFile.get("L2StandardERC20Gateway.proxy");
  const tx = await L2GatewayRouter.initialize(L2ETHGatewayAddress, L2StandardERC20GatewayAddress);
  console.log("initialize L2GatewayRouter, hash:", tx.hash);
  const receipt = await tx.wait();
  console.log(`✅ Done, gas used: ${receipt.gasUsed}`);
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
