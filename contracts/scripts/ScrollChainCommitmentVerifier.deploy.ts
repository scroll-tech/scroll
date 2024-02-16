/* eslint-disable node/no-missing-import */

// Importing dotenv to load environment variables from the .env file
import * as dotenv from "dotenv";

// Importing ethers from hardhat for Ethereum smart contract interactions
import { ethers } from "hardhat";

// Importing functions generateABI and createCode from poseidon.js in the scripts folder
import { generateABI, createCode } from "../scripts/poseidon";

// Loading environment variables from the .env file
dotenv.config();

// Defining an asynchronous function main
async function main() {
  // Getting the deployer (sender) from ethers
  const [deployer] = await ethers.getSigners();

  // Getting the contract factory for ScrollChainCommitmentVerifier
  const ScrollChainCommitmentVerifier = await ethers.getContractFactory("ScrollChainCommitmentVerifier", deployer);

  // Getting the L1ScrollChain address from environment variables
  const L1ScrollChainAddress = process.env.L1_SCROLL_CHAIN_PROXY_ADDR!;

  // Getting the PoseidonUnit2 address from environment variables
  let PoseidonUnit2Address = process.env.POSEIDON_UNIT2_ADDR;

  // If PoseidonUnit2 address is not set in environment variables
  if (!PoseidonUnit2Address) {
    // Creating a contract factory for PoseidonUnit2
    const Poseidon2Elements = new ethers.ContractFactory(generateABI(2), createCode(2), deployer);

    // Deploying the PoseidonUnit2 contract
    const poseidon = await Poseidon2Elements.deploy();
    console.log("Deploy PoseidonUnit2 contract, hash:", poseidon.deployTransaction.hash);
    const receipt = await poseidon.deployTransaction.wait();
    console.log(`✅ Deploy PoseidonUnit2 contract at: ${poseidon.address}, gas used: ${receipt.gasUsed}`);
    // Setting the PoseidonUnit2 address
    PoseidonUnit2Address = poseidon.address;
  }

  // Deploying the ScrollChainCommitmentVerifier contract
  const verifier = await ScrollChainCommitmentVerifier.deploy(PoseidonUnit2Address, L1ScrollChainAddress, {
    gasPrice: 1e9,
  });
  console.log("Deploy ScrollChainCommitmentVerifier contract, hash:", verifier.deployTransaction.hash);
  const receipt = await verifier.deployTransaction.wait();
  console.log(`✅ Deploy ScrollChainCommitmentVerifier contract at: ${verifier.address}, gas used: ${receipt.gasUsed}`);
}

// Running the main function, handling errors
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
