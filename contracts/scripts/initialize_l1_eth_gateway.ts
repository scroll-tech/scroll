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

  const [deployer] = await ethers.getSigners();

  const L1ETHGateway = await ethers.getContractAt(
    "L1ETHGateway",
    addressFile.get("L1ETHGateway.proxy"),
    deployer
  );
  const L2_ETH_GATEWAY_PROXY_ADDR = addressFileL2.get("L2ETHGateway.proxy")
  const L1_GATEWAY_ROUTER_PROXY_ADDR = addressFile.get("L1GatewayRouter.proxy")
  const L1_SCROLL_MESSENGER_PROXY_ADDR = addressFile.get("L1ScrollMessenger.proxy")
  const tx = await L1ETHGateway.initialize(
    L2_ETH_GATEWAY_PROXY_ADDR,
    L1_GATEWAY_ROUTER_PROXY_ADDR,
    L1_SCROLL_MESSENGER_PROXY_ADDR
  )
  console.log("initialize L1ETHGateway, hash:", tx.hash);
  const receipt = await tx.wait();
  console.log(`✅ Done, gas used: ${receipt.gasUsed}`);

}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
