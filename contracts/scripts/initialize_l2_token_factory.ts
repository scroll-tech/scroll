/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";

import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

dotenv.config();

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const ScrollStandardERC20Factory = await ethers.getContractAt(
    "ScrollStandardERC20Factory",
    addressFile.get("ScrollStandardERC20Factory"),
    deployer
  );

  const L2StandardERC20GatewayAddress = addressFile.get("L2StandardERC20Gateway.proxy");

  // if ((await ScrollStandardERC20Factory.owner()) !== L2StandardERC20GatewayAddress) {
  const tx = await ScrollStandardERC20Factory.transferOwnership(L2StandardERC20GatewayAddress);
  console.log("transfer ownernship ScrollStandardERC20Factory, hash:", tx.hash);
  const receipt = await tx.wait();
  console.log(`✅ Done, gas used: ${receipt.gasUsed}`);
  // }
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
