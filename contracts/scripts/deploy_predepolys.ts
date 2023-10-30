/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);
  const contractName = process.env.CONTRACT_NAME!;
  const [deployer] = await ethers.getSigners();
  const OWNER = process.env.owner;
  const txFeeVaultRecipient = process.env.TxFeeVaultRecipient;
  const minWithdrawalAmount = process.env.MinWithdrawalAmount;

  if (!addressFile.get(contractName)) {
    console.log(`>> Deploy ${contractName}`);
    const contractFactory = await ethers.getContractFactory(contractName, deployer);
    let contract;
    if (contractName === "L2TxFeeVault") {
      contract = await contractFactory.deploy(OWNER, txFeeVaultRecipient, minWithdrawalAmount);
    } else {
      contract = await contractFactory.deploy(OWNER);
    }

    console.log(`>> waiting for transaction: ${contract.deployTransaction.hash}`);
    await contract.deployed();
    console.log(`✅ ${contractName} deployed at ${contract.address}`);
    addressFile.set(contractName, contract.address);
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
