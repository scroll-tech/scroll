/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";

import { ethers } from "hardhat";
import { generateABI, createCode } from "../scripts/poseidon";

dotenv.config();

async function main() {
  const [deployer] = await ethers.getSigners();

  const ScrollChainCommitmentVerifier = await ethers.getContractFactory("ScrollChainCommitmentVerifier", deployer);

  const L1ScrollChainAddress = process.env.L1_SCROLL_CHAIN_PROXY_ADDR!;
  let PoseidonUnit2Address = process.env.POSEIDON_UNIT2_ADDR;

  if (!PoseidonUnit2Address) {
    const Poseidon2Elements = new ethers.ContractFactory(generateABI(2), createCode(2), deployer);

    const poseidon = await Poseidon2Elements.deploy();
    console.log("Deploy PoseidonUnit2 contract, hash:", poseidon.deployTransaction.hash);
    const receipt = await poseidon.deployTransaction.wait();
    console.log(`✅ Deploy PoseidonUnit2 contract at: ${poseidon.address}, gas used: ${receipt.gasUsed}`);
    PoseidonUnit2Address = poseidon.address;
  }

  const verifier = await ScrollChainCommitmentVerifier.deploy(PoseidonUnit2Address, L1ScrollChainAddress, {
    gasPrice: 1e9,
  });
  console.log("Deploy ScrollChainCommitmentVerifier contract, hash:", verifier.deployTransaction.hash);
  const receipt = await verifier.deployTransaction.wait();
  console.log(`✅ Deploy ScrollChainCommitmentVerifier contract at: ${verifier.address}, gas used: ${receipt.gasUsed}`);
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
