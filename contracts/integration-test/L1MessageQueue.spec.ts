/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { expect } from "chai";
import { BigNumber, constants } from "ethers";
import { concat, hexlify, keccak256, randomBytes, RLP } from "ethers/lib/utils";
import { ethers } from "hardhat";
import { L1MessageQueue } from "../typechain";

describe("L1MessageQueue", async () => {
  let queue: L1MessageQueue;

  beforeEach(async () => {
    const [deployer] = await ethers.getSigners();

    const L1MessageQueue = await ethers.getContractFactory("L1MessageQueue", deployer);
    queue = await L1MessageQueue.deploy();
    await queue.deployed();

    await queue.initialize(constants.AddressZero, constants.AddressZero);
  });

  it("should succeed", async () => {
    const sender = hexlify(randomBytes(20));
    const target = hexlify(randomBytes(20));
    const transactionType = "0x7E";

    for (const nonce of [
      BigNumber.from(0),
      BigNumber.from(1),
      BigNumber.from(127),
      BigNumber.from(128),
      BigNumber.from(22334455),
      constants.MaxUint256,
    ]) {
      for (const value of [
        BigNumber.from(0),
        BigNumber.from(1),
        BigNumber.from(127),
        BigNumber.from(128),
        BigNumber.from(22334455),
        constants.MaxUint256,
      ]) {
        for (const gasLimit of [
          BigNumber.from(0),
          BigNumber.from(1),
          BigNumber.from(127),
          BigNumber.from(128),
          BigNumber.from(22334455),
          constants.MaxUint256,
        ]) {
          for (const dataLen of [0, 1, 2, 3, 4, 55, 56, 100]) {
            const data = randomBytes(dataLen);
            const transactionPayload = RLP.encode([
              nonce.toHexString(),
              gasLimit.toHexString(),
              target,
              value.toHexString(),
              data,
              sender,
            ]);
            const payload = concat([transactionType, transactionPayload]);
            const expectedHash = keccak256(payload);
            const computedHash = await queue.computeTransactionHash(sender, nonce, value, target, gasLimit, data);
            expect(expectedHash).to.eq(computedHash);
          }
        }
      }
    }
  });
});
