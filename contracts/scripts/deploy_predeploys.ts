/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);
  const contractName = process.env.CONTRACT_NAME!;
  const [deployer] = await ethers.getSigners();
  const OWNER = process.env.owner
  const TxFeeVault_recipient = process.env.TxFeeVault_recipient
  const minWithdrawalAmount = process.env.minWithdrawalAmount

  if (!addressFile.get(contractName)) {
    console.log(`>> Deploy ${contractName}`);
    var contract = await ethers.getContractFactory(contractName, deployer);
    if (contractName=="L2TxFeeVault"){
        var contract_ = await contract.deploy(OWNER, TxFeeVault_recipient, minWithdrawalAmount);
    } else {
        var contract_ = await contract.deploy(OWNER);
    }

    console.log(`>> waiting for transaction: ${contract_.deployTransaction.hash}`);
    await contract_.deployed();
    console.log(`✅ ${contractName} deployed at ${contract_.address}`);
    addressFile.set(contractName, contract_.address);
  }

  // Export contract address to testnet.
  console.log(`testnet-export: ${addressFile.get("ProxyAdmin")}`);
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
