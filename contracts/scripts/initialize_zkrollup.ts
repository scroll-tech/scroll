/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

dotenv.config();

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const Rollup = await ethers.getContractAt(
    "MultipleVersionRollupVerifier",
    addressFile.get("ZKRollup.proxy"),
    deployer
  );
  const ScrollChainAddress = addressFile.get("ScrollChain.proxy");
  console.log(ScrollChainAddress)
  const tx = await Rollup.initialize(ScrollChainAddress);
  console.log("initialize ScrollChain, hash:", tx.hash);
  const receipt = await tx.wait();
  console.log(`✅ Done, gas used: ${receipt.gasUsed}`);
  
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
