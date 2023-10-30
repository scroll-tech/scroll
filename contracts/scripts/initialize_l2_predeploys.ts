/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

dotenv.config();

async function main() {
  const addressFileL2 = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  // initialize L2MessageQueue
  const L2MessageQueue = await ethers.getContractAt("L2MessageQueue", addressFileL2.get("L2MessageQueue"), deployer);
  const L2_SCROLL_MESSENGER_PROXY_ADDR = addressFileL2.get("L2ScrollMessenger.proxy");
  const tx = await L2MessageQueue.initialize(L2_SCROLL_MESSENGER_PROXY_ADDR);
  console.log("initialize L2MessageQueue, hash:", tx.hash);
  const receipt = await tx.wait();
  console.log(`✅ Done, gas used: ${receipt.gasUsed}`);

  // initialize L2TxFeeVault
  const L2TxFeeVault = await ethers.getContractAt("L2TxFeeVault", addressFileL2.get("L2TxFeeVault"), deployer);
  const tx2 = await L2TxFeeVault.updateMessenger(L2_SCROLL_MESSENGER_PROXY_ADDR);
  console.log("initialize L2TxFeeVault, hash:", tx2.hash);
  const receipt2 = await tx2.wait();
  console.log(`✅ Done, gas used: ${receipt2.gasUsed}`);

  // initialize L1GasPriceOracle
  const L1GasPriceOracle = await ethers.getContractAt(
    "L1GasPriceOracle",
    addressFileL2.get("L1GasPriceOracle"),
    deployer
  );
  const L2_WHITELIST_ADDR = addressFileL2.get("Whitelist");
  const tx3 = await L1GasPriceOracle.updateWhitelist(L2_WHITELIST_ADDR);
  console.log("initialize L1GasPriceOracle, hash:", tx3.hash);
  const receipt3 = await tx3.wait();
  console.log(`✅ Done, gas used: ${receipt3.gasUsed}`);
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
