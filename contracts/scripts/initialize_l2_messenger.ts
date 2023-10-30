/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

dotenv.config();

async function main() {
  const addressFileL1 = selectAddressFile("l1geth");
  const addressFileL2 = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  // initialize L2ScrollMessenger
  const L2ScrollMessenger = await ethers.getContractAt(
    "L2ScrollMessenger",
    addressFileL2.get("L2ScrollMessenger.proxy"),
    deployer
  );
  const L1_SCROLL_MESSENGER_PROXY_ADDR = addressFileL1.get("L1ScrollMessenger.proxy");
  const tx = await L2ScrollMessenger.initialize(L1_SCROLL_MESSENGER_PROXY_ADDR);
  console.log("initialize L2ScrollMessenger, hash:", tx.hash);
  const receipt = await tx.wait();
  console.log(`✅ Done, gas used: ${receipt.gasUsed}`);
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
