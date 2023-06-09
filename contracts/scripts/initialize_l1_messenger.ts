/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";

import { constants } from "ethers";
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

dotenv.config();

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const L1ScrollMessenger = await ethers.getContractAt(
    "L1ScrollMessenger",
    addressFile.get("L1ScrollMessenger.proxy"),
    deployer
  );

  const ZKRollupAddress = addressFile.get("ZKRollup.proxy");

  // if ((await L1ScrollMessenger.rollup()) === constants.AddressZero) {
  const tx = await L1ScrollMessenger.initialize(ZKRollupAddress);
  console.log("initialize L1StandardERC20Gateway, hash:", tx.hash);
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
