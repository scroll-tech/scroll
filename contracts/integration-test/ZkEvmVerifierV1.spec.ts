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

    const bytecode = hexlify(fs.readFileSync("./src/libraries/verifier/plonk-verifier/plonk_verifier_0.9.8.bin"));
    const tx = await deployer.sendTransaction({ data: bytecode });
    const receipt = await tx.wait();

    const ZkEvmVerifierV1 = await ethers.getContractFactory("ZkEvmVerifierV1", deployer);
    zkEvmVerifier = await ZkEvmVerifierV1.deploy(receipt.contractAddress);
    await zkEvmVerifier.deployed();
  });

  it("should succeed", async () => {
    const proof = hexlify(fs.readFileSync("./integration-test/testdata/plonk_verifier_0.9.8_proof.data"));
    const instances = fs.readFileSync("./integration-test/testdata/plonk_verifier_0.9.8_pi.data");

    const publicInputHash = new Uint8Array(32);
    for (let i = 0; i < 32; i++) {
      publicInputHash[i] = instances[i * 32 + 31];
    }

    // chunk1: https://github.com/scroll-tech/test-traces/blob/674ad743beab04b57da369fa5958fb6824155bfe/erc20/1_transfer.json
    // 0000000000000005 blockNumber
    // 0000000064c3ca7c timestamp
    // 0000000000000000000000000000000000000000000000000000000000000000 baseFee
    // 00000000007a1200 gasLimit
    // 0001 numTransactions
    // 8da3fedb103b6da8ccc2514094336d1a76df166238f4d8e8558fbe54cce2516a tx hash 0
    // chunk2: https://github.com/scroll-tech/test-traces/blob/674ad743beab04b57da369fa5958fb6824155bfe/erc20/10_transfer.json
    // 0000000000000006 blockNumber
    // 0000000064c3ca7f timestamp
    // 0000000000000000000000000000000000000000000000000000000000000000 baseFee
    // 00000000007a1200 gasLimit
    // 000a numTransactions
    // 419164c1a7213e4e52f8578463c47a01549f69a7ff220d93221ce02909f5b919 tx hash 0
    // 6c1b03d1a9b5156e189ad2e7ba73ba71d9a83b24f9830f38dd7a597fe1e67167 tx hash 1
    // 94f981938d02b2c1d91ff370b3ed759dadc617c7347cd4b8552b275edbffd767 tx hash 2
    // bfe98147fc808a916bdff90e838e77609fd59634787443f6fc58f9a371790d09 tx hash 3
    // beb9dd0259e7c4f0a8d5ac3ba6aa3940c3e53947395f64e8ee88c7067c6d210e tx hash 4
    // 208c6c767356552ad8085fa77a99d9154e0c8cf8777e329cb76bcbc969d21fca tx hash 5
    // 37c8969833fbc6cbb88a63ccef324d7b42d0607ac0094f14e1f6d4e50f84d87f tx hash 6
    // 088c5ad45a990694ac783207fe6bda9bf97da40e1f3eb468c73941d51b99932c tx hash 7
    // c3d8ddbdfc67877a253255b9357aabfd062ce80d39eba67547f964c288660065 tx hash 8
    // ff26ca52c02b97b1a6677263d5d6dec0321fb7b49be44ae0a66ba5482b1180b4 tx hash 9
    // => chunk 0 data hash: 9390886a7d22aa43aae87e62a350c904fabc5db4487d9b25bdca446ba7ed15a1
    // => chunk 1 data hash: a8846bf9bc53f30a391ae452b5fd456cb86a99ab7bd2e1e47898ffbe3509e8eb
    // => batch data hash: ee64d77c2f2e0b2c4ac952a0f54fdba4a217c42eb26a07b28de9fbc7b009acae
    // 000000000000cf55 layer2ChainId
    // 02040e949809e8d2e56d35b4dfb876e08ee7b4608d22f23f52052425857c31ba prevStateRoot
    // 1532cdb7732da0a4ca3044914c6959b7e2b7ba4e913a9f5f0b55051e467412d9 postStateRoot
    // 0000000000000000000000000000000000000000000000000000000000000000 withdrawRoot
    // ee64d77c2f2e0b2c4ac952a0f54fdba4a217c42eb26a07b28de9fbc7b009acae batchDataHash
    // public input hash: 9ea439164727042e029464a40901e52800095c1ade301b63b4b7453880f5723e
    expect(hexlify(publicInputHash)).to.eq("0x9ea439164727042e029464a40901e52800095c1ade301b63b4b7453880f5723e");

    // verify ok
    await zkEvmVerifier.verify(proof, publicInputHash);
    console.log("Gas Usage:", (await zkEvmVerifier.estimateGas.verify(proof, publicInputHash)).toString());

    // verify failed
    await expect(zkEvmVerifier.verify(proof, publicInputHash.reverse())).to.reverted;
  });
});
