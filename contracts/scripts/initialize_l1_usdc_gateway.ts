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

  const L1USDCGateway = await ethers.getContractAt("L1USDCGateway", addressFile.get("L1USDCGateway.proxy"), deployer);

  const L1GatewayRouterAddress = addressFile.get("L1GatewayRouter.proxy");
  const L1ScrollMessengerAddress = addressFile.get("L1ScrollMessenger.proxy");
  const L2USDCGatewayAddress = process.env.L2_USDC_GATEWAY_PROXY_ADDR!;

  if ((await L1USDCGateway.counterpart()) === constants.AddressZero) {
    const tx = await L1USDCGateway.initialize(L2USDCGatewayAddress, L1GatewayRouterAddress, L1ScrollMessengerAddress);
    console.log("initialize L1USDCGateway, hash:", tx.hash);
    const receipt = await tx.wait();
    console.log(`✅ Done, gas used: ${receipt.gasUsed}`);
  }
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
