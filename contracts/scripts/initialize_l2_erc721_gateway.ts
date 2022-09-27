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

  const L2ERC721Gateway = await ethers.getContractAt(
    "L2ERC721Gateway",
    addressFile.get("L2ERC721Gateway.proxy"),
    deployer
  );

  const L2ScrollMessengerAddress = addressFile.get("L2ScrollMessenger");
  const L1ERC721GatewayAddress = process.env.L1_ERC721_GATEWAY_PROXY_ADDR!;

  if ((await L2ERC721Gateway.counterpart()) === constants.AddressZero) {
    const tx = await L2ERC721Gateway.initialize(L1ERC721GatewayAddress, L2ScrollMessengerAddress);
    console.log("initialize L2ERC721Gateway, hash:", tx.hash);
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
