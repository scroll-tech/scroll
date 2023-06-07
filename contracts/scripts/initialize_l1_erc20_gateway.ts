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

  const L1StandardERC20Gateway = await ethers.getContractAt(
    "L1StandardERC20Gateway",
    addressFile.get("L1StandardERC20Gateway.proxy"),
    deployer
  );

  const L1GatewayRouterAddress = addressFile.get("L1GatewayRouter.proxy");
  const L1ScrollMessengerAddress = addressFile.get("L1ScrollMessenger.proxy");
  const L2StandardERC20GatewayAddress = process.env.L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR!;
  const L2StandardERC20Impl = process.env.L2_SCROLL_STANDARD_ERC20_ADDR!;
  const L2StandardERC20FactoryAddress = process.env.L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR!;

  // if ((await L1StandardERC20Gateway.counterpart()) === constants.AddressZero) {
  const tx = await L1StandardERC20Gateway.initialize(
    L2StandardERC20GatewayAddress,
    L1GatewayRouterAddress,
    L1ScrollMessengerAddress,
    L2StandardERC20Impl,
    L2StandardERC20FactoryAddress
  );
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
