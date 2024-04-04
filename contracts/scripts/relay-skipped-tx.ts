/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";

import { ethers } from "hardhat";

dotenv.config();

async function main() {
  const [deployer] = await ethers.getSigners();

  const l1ScrollMessengerAddress = process.env.L1_SCROLL_MESSENGER_PROXY_ADDR!;
  const l2EccContractAddress = process.env.L2_ECC_ADDR!;
  const payload = process.env.SKIPPED_TX_PAYLOAD!; // TODO: calc the payload, parse as bytes

  const L1ScrollMessenger = await ethers.getContractAt("L1ScrollMessenger", l1ScrollMessengerAddress, deployer);

  const tx = await L1ScrollMessenger.sendMessage(
    l2EccContractAddress, // address _to
    0, // uint256 _value
    payload, // bytes memory _message
    100000000 // uint256 _gasLimit
  );

  console.log(`calling ${l2EccContractAddress} with payload from l1, hash:`, tx.hash);
  const receipt = await tx.wait();
  console.log(`✅ Done, gas used: ${receipt.gasUsed}`);
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
