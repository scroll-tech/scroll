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

  const ScrollChain = await ethers.getContractAt(
    "ScrollChain",
    addressFile.get("ScrollChain.proxy"),
    deployer
  );

  const ZKRollupAddress = addressFile.get("ZKRollup.proxy");
  const L1MessageQueueAddress = addressFile.get("L1MessageQueue.proxy");
  const maxNumTxInChunk = 100

  const tx = await ScrollChain.initialize(L1MessageQueueAddress, ZKRollupAddress, maxNumTxInChunk);
  console.log("initialize ScrollChain, hash:", tx.hash);
  const receipt = await tx.wait();
  console.log(`✅ Done, gas used: ${receipt.gasUsed}`);
  
  const OWNER = process.env.owner || "0xe6F7E94BA0a1359D125e4226E390Ed9B419F72DF";
  const tx2 = await ScrollChain.addSequencer(OWNER);
  console.log("initialize ScrollChain addSequencer, hash:", tx2.hash);
  const receipt2 = await tx2.wait();
  console.log(`✅ Done, gas used: ${receipt2.gasUsed}`);

  const tx3 = await ScrollChain.addProver(OWNER);
  console.log("initialize ScrollChain addProver, hash:", tx3.hash);
  const receipt3 = await tx3.wait();
  console.log(`✅ Done, gas used: ${receipt3.gasUsed}`);

  
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
