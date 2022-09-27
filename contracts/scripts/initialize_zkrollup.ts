/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv"

import * as hre from "hardhat"
import { ethers } from "hardhat"
import { selectAddressFile } from "./utils"

dotenv.config();

const CHAIN_ID_L2 = process.env.CHAIN_ID_L2 || "none";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const ZKRollup = await ethers.getContractAt("ZKRollup", addressFile.get("ZKRollup.proxy"), deployer);

  // if ((await ZKRollup.owner()) === constants.AddressZero) {
  {
    const tx = await ZKRollup.initialize(CHAIN_ID_L2);
    console.log("initialize ZKRollup, hash:", tx.hash);
    const receipt = await tx.wait();
    console.log(`✅ Done, gas used: ${receipt.gasUsed}`);
  }

  const L1ScrollMessengerAddress = addressFile.get("L1ScrollMessenger.proxy");
  // if ((await ZKRollup.messenger()) === constants.AddressZero) {
  {
    const tx = await ZKRollup.updateMessenger(L1ScrollMessengerAddress);
    console.log("updateMessenger ZKRollup, hash:", tx.hash);
    const receipt = await tx.wait();
    console.log(`✅ Done, gas used: ${receipt.gasUsed}`);
  }

  const L1RollupOperatorAddress = process.env.L1_ROLLUP_OPERATOR_ADDR!;
  // if ((await ZKRollup.operator()) === constants.AddressZero) 
  {
    console.log("L1_ROLLUP_OPERATOR_ADDR", L1RollupOperatorAddress);
    const tx = await ZKRollup.updateOperator(L1RollupOperatorAddress);
    console.log("updateOperator ZKRollup, hash:", tx.hash);
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
