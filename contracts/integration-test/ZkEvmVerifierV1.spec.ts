/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { expect } from "chai";
import { hexlify } from "ethers/lib/utils";
import { ethers } from "hardhat";
import { ZkEvmVerifierV1 } from "../typechain";
import { SignerWithAddress } from "@nomiclabs/hardhat-ethers/signers";
import fs from "fs";

describe("ZkEvmVerifierV1", async () => {
  let deployer: SignerWithAddress;

  let zkEvmVerifier: ZkEvmVerifierV1;

  beforeEach(async () => {
    [deployer] = await ethers.getSigners();

    const bytecode = hexlify(fs.readFileSync("./src/libraries/verifier/plonk-verifier/plonk_verifier_0.5.1.bin"));
    const tx = await deployer.sendTransaction({ data: bytecode });
    const receipt = await tx.wait();

    const ZkEvmVerifierV1 = await ethers.getContractFactory("ZkEvmVerifierV1", deployer);
    zkEvmVerifier = await ZkEvmVerifierV1.deploy(receipt.contractAddress);
    await zkEvmVerifier.deployed();
  });

  it("should succeed", async () => {
    const proof = hexlify(fs.readFileSync("./integration-test/testdata/plonk_verifier_0.5.1_proof.data"));
    const instances = fs.readFileSync("./integration-test/testdata/plonk_verifier_0.5.1_pi.data");

    const publicInputHash = new Uint8Array(32);
    for (let i = 0; i < 32; i++) {
      publicInputHash[i] = instances[i * 32 + 31];
    }

    // verify ok
    await zkEvmVerifier.verify(proof, publicInputHash);
    console.log("Gas Usage:", (await zkEvmVerifier.estimateGas.verify(proof, publicInputHash)).toString());

    // verify failed
    await expect(zkEvmVerifier.verify(proof, publicInputHash.reverse())).to.reverted;
  });
});
