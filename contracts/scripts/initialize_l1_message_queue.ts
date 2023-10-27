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

  const L1MessageQueue = await ethers.getContractAt(
    "L1MessageQueue",
    addressFile.get("L1MessageQueue.proxy"),
    deployer
  );
  const L1_SCROLL_MESSENGER_PROXY_ADDR = addressFile.get("L1ScrollMessenger.proxy")
  const L1_SCROLL_CHAIN_PROXY_ADDR = addressFile.get("ScrollChain.proxy")
  const L1_ENFORCED_TX_GATEWAY_PROXY_ADDR = addressFile.get("EnforcedTxGateway.proxy")
  const L2_GAS_PRICE_ORACLE_PROXY_ADDR = addressFile.get("L2GasPriceOracle.proxy")
  const MAX_L1_MESSAGE_GAS_LIMIT = 1000000000000000
  const tx = await L1MessageQueue.initialize(
    L1_SCROLL_MESSENGER_PROXY_ADDR,
    L1_SCROLL_CHAIN_PROXY_ADDR,
    L1_ENFORCED_TX_GATEWAY_PROXY_ADDR,
    L2_GAS_PRICE_ORACLE_PROXY_ADDR,
    MAX_L1_MESSAGE_GAS_LIMIT
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
