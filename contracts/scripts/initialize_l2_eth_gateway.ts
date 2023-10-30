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

  const L2ETHGateway = await ethers.getContractAt("L2ETHGateway", addressFileL2.get("L2ETHGateway.proxy"), deployer);
  const L1_ETH_GATEWAY_PROXY_ADDR = addressFileL1.get("L1ETHGateway.proxy");
  const L2_GATEWAY_ROUTER_PROXY_ADDR = addressFileL2.get("L2GatewayRouter.proxy");
  const L2_SCROLL_MESSENGER_PROXY_ADDR = addressFileL2.get("L2ScrollMessenger.proxy");
  const tx = await L2ETHGateway.initialize(
    L1_ETH_GATEWAY_PROXY_ADDR,
    L2_GATEWAY_ROUTER_PROXY_ADDR,
    L2_SCROLL_MESSENGER_PROXY_ADDR
  );
  console.log("initialize L2ETHGateway, hash:", tx.hash);
  const receipt = await tx.wait();
  console.log(`✅ Done, gas used: ${receipt.gasUsed}`);
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
