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

  const L2USDCGateway = await ethers.getContractAt("L2USDCGateway", addressFile.get("L2USDCGateway.proxy"), deployer);

  const L2GatewayRouterAddress = addressFile.get("L2GatewayRouter.proxy");
  const L2ScrollMessengerAddress = addressFile.get("L2ScrollMessenger.proxy");
  const L1USDCGatewayAddress = process.env.L1_USDC_GATEWAY_PROXY_ADDR!;

  if ((await L2USDCGateway.counterpart()) === constants.AddressZero) {
    const tx = await L2USDCGateway.initialize(L1USDCGatewayAddress, L2GatewayRouterAddress, L2ScrollMessengerAddress);
    console.log("initialize L2USDCGateway, hash:", tx.hash);
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
