/* eslint-disable node/no-missing-import */
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

async function main() {
  const addressFile = selectAddressFile(hre.network.name);

  const [deployer] = await ethers.getSigners();

  const ProxyAdmin = await ethers.getContractAt("ProxyAdmin", addressFile.get("ProxyAdmin"), deployer);

  if (!addressFile.get("L2StandardERC20Gateway.implementation")) {
    console.log(">> Deploy L2StandardERC20Gateway implementation");
    const L2StandardERC20Gateway = await ethers.getContractFactory("L2StandardERC20Gateway", deployer);
    const impl = await L2StandardERC20Gateway.deploy();
    console.log(`>> waiting for transaction: ${impl.deployTransaction.hash}`);
    await impl.deployed();
    console.log(`✅ L2StandardERC20Gateway implementation deployed at ${impl.address}`);
    addressFile.set("L2StandardERC20Gateway.implementation", impl.address);
  }

  const impl = addressFile.get("L2StandardERC20Gateway.implementation") as string;

  if (!addressFile.get("L2StandardERC20Gateway.proxy")) {
    console.log(">> Deploy L2StandardERC20Gateway proxy");
    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);
    const proxy = await TransparentUpgradeableProxy.deploy(impl, ProxyAdmin.address, "0x");
    console.log(`>> waiting for transaction: ${proxy.deployTransaction.hash}`);
    await proxy.deployed();
    console.log(`✅ L2StandardERC20Gateway proxy deployed at ${proxy.address}`);
    addressFile.set("L2StandardERC20Gateway.proxy", proxy.address);
  }

  // Export contract address to testnet.
  console.log(
    `testnet-export: ${addressFile.get("L2StandardERC20Gateway.implementation")};${addressFile.get(
      "L2StandardERC20Gateway.proxy"
    )}`
  );
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
