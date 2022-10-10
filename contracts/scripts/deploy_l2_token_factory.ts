/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  if (!addressFile.get("ScrollStandardERC20")) {
    console.log(">> Deploy ScrollStandardERC20");
    const ScrollStandardERC20 = await ethers.getContractFactory("ScrollStandardERC20", deployer);
    const token = await ScrollStandardERC20.deploy();
    console.log(`>> waiting for transaction: ${token.deployTransaction.hash}`);
    await token.deployed();
    console.log(`✅ ScrollStandardERC20 deployed at ${token.address}`);
    addressFile.set("ScrollStandardERC20", token.address);
  }

  const tokenImpl = addressFile.get("ScrollStandardERC20") as string;

  if (!addressFile.get("ScrollStandardERC20Factory")) {
    console.log(">> Deploy ScrollStandardERC20Factory");
    const ScrollStandardERC20Factory = await ethers.getContractFactory("ScrollStandardERC20Factory", deployer);
    const token = await ScrollStandardERC20Factory.deploy(tokenImpl);
    console.log(`>> waiting for transaction: ${token.deployTransaction.hash}`);
    await token.deployed();
    console.log(`✅ ScrollStandardERC20Factory deployed at ${token.address}`);
    addressFile.set("ScrollStandardERC20Factory", token.address);
  }

  // Export contract address to testnet.
  console.log(
    `testnet-export: ${addressFile.get("ScrollStandardERC20")};${addressFile.get("ScrollStandardERC20Factory")}`
  );
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
