/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";

import * as hre from "hardhat";
import { ethers } from "hardhat";
import fs from "fs";
import { selectAddressFile } from "./utils";

dotenv.config();

const GENESIS_FILE_PATH = process.env.GENESIS_FILE_PATH || "genesis.json";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const rollupAddr = process.env.L1_ZK_ROLLUP_PROXY_ADDR || addressFile.get("ZKRollup.proxy") || "0x";
  console.log("Using rollup proxy address:", rollupAddr);

  const ZKRollup = await ethers.getContractAt("ZKRollup", rollupAddr, deployer);
  const genesis = JSON.parse(fs.readFileSync(GENESIS_FILE_PATH, 'utf8'));
  console.log("Using genesis block:", genesis.blockHash);

  const tx = await ZKRollup.importGenesisBlock(genesis);

  console.log("importGenesisBlock ZKRollup, hash:", tx.hash);
  const receipt = await tx.wait();
  console.log(`✅ Done, gas used: ${receipt.gasUsed}`);
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
