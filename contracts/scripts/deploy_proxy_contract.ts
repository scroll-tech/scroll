/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

dotenv.config();

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  if (process.env.CONTRACT_NAME === undefined) {
    throw new Error("env CONTRACT_NAME undefined");
  }

  const contractName = process.env.CONTRACT_NAME!;
  const ProxyAdmin = await ethers.getContractAt("ProxyAdmin", addressFile.get("ProxyAdmin"), deployer);

  if (!addressFile.get(`${contractName}.implementation`)) {
    console.log(`>> Deploy ${contractName} implementation`);
    const ContractImpl = await ethers.getContractFactory(contractName, deployer);
    const impl = await ContractImpl.deploy(process.env.L1_USDC_ADDR, process.env.L2_USDC_ADDR);
    console.log(`>> waiting for transaction: ${impl.deployTransaction.hash}`);
    await impl.deployed();
    console.log(`✅ ${contractName} implementation deployed at ${impl.address}`);
    addressFile.set(`${contractName}.implementation`, impl.address);
  }

  const impl = addressFile.get(`${contractName}.implementation`) as string;

  if (!addressFile.get(`${contractName}.proxy`)) {
    console.log(`>> Deploy ${contractName} proxy`);
    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);
    const proxy = await TransparentUpgradeableProxy.deploy(impl, ProxyAdmin.address, "0x");
    console.log(`>> waiting for transaction: ${proxy.deployTransaction.hash}`);
    await proxy.deployed();
    console.log(`✅ ${contractName} proxy deployed at ${proxy.address}`);
    addressFile.set(`${contractName}.proxy`, proxy.address);
  }

  // Export contract address to testnet.
  console.log(
    `testnet-export: ${addressFile.get(`${contractName}.implementation`)};${addressFile.get(`${contractName}.proxy`)}`
  );
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
