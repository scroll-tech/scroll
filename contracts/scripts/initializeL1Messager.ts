/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";

import { constants } from "ethers";
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

dotenv.config();

async function main() {
  const addressFile = selectAddressFile(hre.network.name);
  const addressFileL2 = selectAddressFile("l2geth");
  console.log(hre.network.name)
  const [deployer] = await ethers.getSigners();

  const L1Messageer = await ethers.getContractAt(
    "L1ScrollMessenger",
    addressFile.get("L1ScrollMessenger.proxy"),
    deployer
  );

  const L2_SCROLL_MESSENGER_PROXY_ADDR = addressFileL2.get("L2ScrollMessenger.proxy")
  const L1_SCROLL_CHAIN_PROXY_ADDR = addressFile.get("ScrollChain.proxy")
  const L1_MESSAGE_QUEUE_PROXY_ADDR = addressFile.get("L1MessageQueue.proxy")
  const L2_FEE_VAULT_ADDR = addressFile.get("L2TxFeeVault")

  const tx = await L1Messageer.initialize(
    L2_SCROLL_MESSENGER_PROXY_ADDR,
    L2_FEE_VAULT_ADDR,
    L1_SCROLL_CHAIN_PROXY_ADDR,
    L1_MESSAGE_QUEUE_PROXY_ADDR
  )

  console.log("initialize L2GasPriceOracle, hash:", tx.hash);
  const receipt = await tx.wait();
  console.log(`✅ Done, gas used: ${receipt.gasUsed}`);

}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
