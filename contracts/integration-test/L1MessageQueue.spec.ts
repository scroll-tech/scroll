/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { expect } from "chai";
import { BigNumber, constants, utils } from "ethers";
import { concat, hexlify, keccak256, randomBytes, RLP, UnsignedTransaction } from "ethers/lib/utils";
import { L1MessageQueue } from "../typechain";
import { ethers } from "hardhat";

describe.only("L1MessageQueue", async () => {
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

  it("should give example transaction hash to match to geth", async () => {
    const sender = hexlify("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266");
    const target = hexlify("0x70997970C51812dc3A010C7d01b50e0d17dc79C8");
    const nonce = BigNumber.from(1);
    const value = BigNumber.from(2);
    const gasLimit = BigNumber.from(3);
    const data = Uint8Array.from([1, 2, 3, 4]);

    const computedHash = await queue.computeTransactionHash(sender, nonce, value, target, gasLimit, data);
    expect(computedHash).to.eq("0x1cebed6d90ef618f60eec1b7edc0df36b298a237c219f0950081acfb72eac6be");
  });
});
