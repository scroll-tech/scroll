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

  const L1ERC721Gateway = await ethers.getContractAt(
    "L1ERC721Gateway",
    addressFile.get("L1ERC721Gateway.proxy"),
    deployer
  );

  const L1ScrollMessengerAddress = addressFile.get("L1ScrollMessenger.proxy");
  const L2ERC721GatewayAddress = process.env.L2_ERC721_GATEWAY_PROXY_ADDR!;

  if ((await L1ERC721Gateway.counterpart()) === constants.AddressZero) {
    const tx = await L1ERC721Gateway.initialize(L2ERC721GatewayAddress, L1ScrollMessengerAddress);
    console.log("initialize L1ERC721Gateway, hash:", tx.hash);
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
