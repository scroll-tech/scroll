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

  const L2GasPriceOracle = await ethers.getContractAt(
    "L2GasPriceOracle",
    addressFile.get("L2GasPriceOracle.proxy"),
    deployer
  );

  const tx = await L2GasPriceOracle.initialize( 
    21000, // _txGas
    53000, // _txGasContractCreation
    4, // _zeroGas
    16
  )
  console.log("initialize L2GasPriceOracle, hash:", tx.hash);
  const receipt = await tx.wait();
  console.log(`✅ Done, gas used: ${receipt.gasUsed}`);


  const WhitelistAddress = addressFile.get("Whitelist")
  console.log(WhitelistAddress)
  const tx2 = await L2GasPriceOracle.updateWhitelist(WhitelistAddress)
  console.log("initialize L2GasPriceOracle, hash:", tx2.hash);
  const receipt2 = await tx2.wait();
  console.log(`✅ Done, gas used: ${receipt2.gasUsed}`);

  
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
