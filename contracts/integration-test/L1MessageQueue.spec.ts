/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { expect } from "chai";
import { BigNumber, constants } from "ethers";
import { concat, getAddress, hexlify, keccak256, randomBytes, RLP, stripZeros } from "ethers/lib/utils";
import { ethers } from "hardhat";
import { L1MessageQueue, L2GasPriceOracle } from "../typechain";
import { SignerWithAddress } from "@nomiclabs/hardhat-ethers/signers";

describe("L1MessageQueue", async () => {
  let deployer: SignerWithAddress;
  let scrollChain: SignerWithAddress;
  let messenger: SignerWithAddress;
  let gateway: SignerWithAddress;
  let signer: SignerWithAddress;

  let oracle: L2GasPriceOracle;
  let queue: L1MessageQueue;

  const deployProxy = async (name: string, admin: string, args: any[]): Promise<string> => {
    const TransparentUpgradeableProxy = await ethers.getContractFactory("TransparentUpgradeableProxy", deployer);
    const Factory = await ethers.getContractFactory(name, deployer);
    const impl = args.length > 0 ? await Factory.deploy(...args) : await Factory.deploy();
    await impl.deployed();
    const proxy = await TransparentUpgradeableProxy.deploy(impl.address, admin, "0x");
    await proxy.deployed();
    return proxy.address;
  };

  beforeEach(async () => {
    [deployer, scrollChain, messenger, gateway, signer] = await ethers.getSigners();

    const ProxyAdmin = await ethers.getContractFactory("ProxyAdmin", deployer);
    const admin = await ProxyAdmin.deploy();
    await admin.deployed();

    queue = await ethers.getContractAt(
      "L1MessageQueue",
      await deployProxy("L1MessageQueue", admin.address, [messenger.address, scrollChain.address, gateway.address]),
      deployer
    );

    oracle = await ethers.getContractAt(
      "L2GasPriceOracle",
      await deployProxy("L2GasPriceOracle", admin.address, []),
      deployer
    );

    await oracle.initialize(21000, 50000, 8, 16);
    await queue.initialize(messenger.address, scrollChain.address, constants.AddressZero, oracle.address, 10000000);
  });

  context("auth", async () => {
    it("should initialize correctly", async () => {
      expect(await queue.owner()).to.eq(deployer.address);
      expect(await queue.messenger()).to.eq(messenger.address);
      expect(await queue.scrollChain()).to.eq(scrollChain.address);
      expect(await queue.enforcedTxGateway()).to.eq(gateway.address);
      expect(await queue.gasOracle()).to.eq(oracle.address);
      expect(await queue.maxGasLimit()).to.eq(10000000);
    });

    it("should revert, when initialize again", async () => {
      await expect(
        queue.initialize(constants.AddressZero, constants.AddressZero, constants.AddressZero, constants.AddressZero, 0)
      ).to.revertedWith("Initializable: contract is already initialized");
    });

    context("#updateGasOracle", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(queue.connect(signer).updateGasOracle(constants.AddressZero)).to.revertedWith(
          "Ownable: caller is not the owner"
        );
      });

      it("should succeed", async () => {
        expect(await queue.gasOracle()).to.eq(oracle.address);
        await expect(queue.updateGasOracle(deployer.address))
          .to.emit(queue, "UpdateGasOracle")
          .withArgs(oracle.address, deployer.address);
        expect(await queue.gasOracle()).to.eq(deployer.address);
      });
    });

    context("#updateMaxGasLimit", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(queue.connect(signer).updateMaxGasLimit(0)).to.revertedWith("Ownable: caller is not the owner");
      });

      it("should succeed", async () => {
        expect(await queue.maxGasLimit()).to.eq(10000000);
        await expect(queue.updateMaxGasLimit(0)).to.emit(queue, "UpdateMaxGasLimit").withArgs(10000000, 0);
        expect(await queue.maxGasLimit()).to.eq(0);
      });
    });
  });

  context("#computeTransactionHash", async () => {
    it("should succeed", async () => {
      const sender = "0xb2a70fab1a45b1b9be443b6567849a1702bc1232";
      const target = "0xcb18150e4efefb6786130e289a5f61a82a5b86d7";
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
              const tests = [randomBytes(dataLen)];
              if (dataLen === 1) {
                for (const byte of [0, 1, 127, 128]) {
                  tests.push(Uint8Array.from([byte]));
                }
              }
              for (const data of tests) {
                const transactionPayload = RLP.encode([
                  stripZeros(nonce.toHexString()),
                  stripZeros(gasLimit.toHexString()),
                  target,
                  stripZeros(value.toHexString()),
                  data,
                  sender,
                ]);
                const payload = concat([transactionType, transactionPayload]);
                const expectedHash = keccak256(payload);
                const computedHash = await queue.computeTransactionHash(sender, nonce, value, target, gasLimit, data);
                if (computedHash !== expectedHash) {
                  console.log(hexlify(transactionPayload));
                  console.log(nonce, gasLimit, target, value, data, sender);
                }
                expect(expectedHash).to.eq(computedHash);
              }
            }
          }
        }
      }
    });
  });

  context("#appendCrossDomainMessage", async () => {
    it("should revert, when non-messenger call", async () => {
      await expect(queue.connect(signer).appendCrossDomainMessage(constants.AddressZero, 0, "0x")).to.revertedWith(
        "Only callable by the L1ScrollMessenger"
      );
    });

    it("should revert, when exceed maxGasLimit", async () => {
      await expect(
        queue.connect(messenger).appendCrossDomainMessage(constants.AddressZero, 10000001, "0x")
      ).to.revertedWith("Gas limit must not exceed maxGasLimit");
    });

    it("should revert, when below intrinsic gas", async () => {
      await expect(queue.connect(messenger).appendCrossDomainMessage(constants.AddressZero, 0, "0x")).to.revertedWith(
        "Insufficient gas limit, must be above intrinsic gas"
      );
    });

    it("should succeed", async () => {
      expect(await queue.nextCrossDomainMessageIndex()).to.eq(constants.Zero);
      const sender = getAddress(
        BigNumber.from(messenger.address)
          .add("0x1111000000000000000000000000000000001111")
          .mod(BigNumber.from(2).pow(160))
          .toHexString()
          .slice(2)
          .padStart(40, "0")
      );
      const hash = await queue.computeTransactionHash(sender, 0, 0, signer.address, 100000, "0x01");
      await expect(queue.connect(messenger).appendCrossDomainMessage(signer.address, 100000, "0x01"))
        .to.emit(queue, "QueueTransaction")
        .withArgs(sender, signer.address, 0, 0, 100000, "0x01");
      expect(await queue.nextCrossDomainMessageIndex()).to.eq(constants.One);
      expect(await queue.getCrossDomainMessage(0)).to.eq(hash);
    });
  });

  context("#appendEnforcedTransaction", async () => {
    it("should revert, when non-gateway call", async () => {
      await expect(
        queue.connect(signer).appendEnforcedTransaction(signer.address, constants.AddressZero, 0, 0, "0x")
      ).to.revertedWith("Only callable by the EnforcedTxGateway");
    });

    it("should revert, when sender is not EOA", async () => {
      await expect(
        queue.connect(gateway).appendEnforcedTransaction(queue.address, constants.AddressZero, 0, 0, "0x")
      ).to.revertedWith("only EOA");
    });

    it("should revert, when exceed maxGasLimit", async () => {
      await expect(
        queue.connect(gateway).appendEnforcedTransaction(signer.address, constants.AddressZero, 0, 10000001, "0x")
      ).to.revertedWith("Gas limit must not exceed maxGasLimit");
    });

    it("should revert, when below intrinsic gas", async () => {
      await expect(
        queue.connect(gateway).appendEnforcedTransaction(signer.address, constants.AddressZero, 0, 0, "0x")
      ).to.revertedWith("Insufficient gas limit, must be above intrinsic gas");
    });

    it("should succeed", async () => {
      expect(await queue.nextCrossDomainMessageIndex()).to.eq(constants.Zero);
      const sender = signer.address;
      const hash = await queue.computeTransactionHash(sender, 0, 200, signer.address, 100000, "0x01");
      await expect(
        queue.connect(gateway).appendEnforcedTransaction(signer.address, signer.address, 200, 100000, "0x01")
      )
        .to.emit(queue, "QueueTransaction")
        .withArgs(sender, signer.address, 200, 0, 100000, "0x01");
      expect(await queue.nextCrossDomainMessageIndex()).to.eq(constants.One);
      expect(await queue.getCrossDomainMessage(0)).to.eq(hash);
    });
  });

  context("#popCrossDomainMessage", async () => {
    it("should revert, when non-scrollChain call", async () => {
      await expect(queue.connect(signer).popCrossDomainMessage(0, 0, 0)).to.revertedWith(
        "Only callable by the ScrollChain"
      );
    });

    it("should revert, when pop too many messages", async () => {
      await expect(queue.connect(scrollChain).popCrossDomainMessage(0, 257, 0)).to.revertedWith(
        "pop too many messages"
      );
    });

    it("should revert, when start index mismatch", async () => {
      await expect(queue.connect(scrollChain).popCrossDomainMessage(1, 256, 0)).to.revertedWith("start index mismatch");
    });

    it("should succeed", async () => {
      // append 512 messages
      for (let i = 0; i < 256 * 2; i++) {
        await queue.connect(messenger).appendCrossDomainMessage(constants.AddressZero, 1000000, "0x");
      }

      // pop 50 messages with no skip
      await expect(queue.connect(scrollChain).popCrossDomainMessage(0, 50, 0))
        .to.emit(queue, "DequeueTransaction")
        .withArgs(0, 50, 0);
      for (let i = 0; i < 50; i++) {
        expect(await queue.isMessageSkipped(i)).to.eq(false);
        expect(await queue.isMessageDropped(i)).to.eq(false);
      }
      expect(await queue.pendingQueueIndex()).to.eq(50);

      // pop 10 messages all skip
      await expect(queue.connect(scrollChain).popCrossDomainMessage(50, 10, 1023))
        .to.emit(queue, "DequeueTransaction")
        .withArgs(50, 10, 1023);
      expect(await queue.pendingQueueIndex()).to.eq(60);
      for (let i = 50; i < 60; i++) {
        expect(await queue.isMessageSkipped(i)).to.eq(true);
        expect(await queue.isMessageDropped(i)).to.eq(false);
      }

      // pop 20 messages, skip first 5
      await expect(queue.connect(scrollChain).popCrossDomainMessage(60, 20, 31))
        .to.emit(queue, "DequeueTransaction")
        .withArgs(60, 20, 31);
      expect(await queue.pendingQueueIndex()).to.eq(80);
      for (let i = 60; i < 65; i++) {
        expect(await queue.isMessageSkipped(i)).to.eq(true);
        expect(await queue.isMessageDropped(i)).to.eq(false);
      }
      for (let i = 65; i < 80; i++) {
        expect(await queue.isMessageSkipped(i)).to.eq(false);
        expect(await queue.isMessageDropped(i)).to.eq(false);
      }

      // pop 256 messages with random skip
      const bitmap = BigNumber.from("0x496525059c3f33758d17030403e45afe067b8a0ae1317cda0487fd2932cbea1a");
      const tx = await queue.connect(scrollChain).popCrossDomainMessage(80, 256, bitmap);
      await expect(tx).to.emit(queue, "DequeueTransaction").withArgs(80, 256, bitmap);
      console.log("gas used:", (await tx.wait()).gasUsed.toString());
      for (let i = 80; i < 80 + 256; i++) {
        expect(await queue.isMessageSkipped(i)).to.eq(
          bitmap
            .shr(i - 80)
            .and(1)
            .eq(1)
        );
        expect(await queue.isMessageDropped(i)).to.eq(false);
      }
    });

    // @note skip this random benchmark tests
    for (const count1 of [1, 2, 128, 129, 256]) {
      for (const count2 of [1, 2, 128, 129, 256]) {
        for (const count3 of [1, 2, 128, 129, 256]) {
          it.skip(`should succeed on random tests, pop three times each with ${count1} ${count2} ${count3} msgs`, async () => {
            // append count1 + count2 + count3 messages
            for (let i = 0; i < count1 + count2 + count3; i++) {
              await queue.connect(messenger).appendCrossDomainMessage(constants.AddressZero, 1000000, "0x");
            }

            // first pop `count1` messages
            const bitmap1 = BigNumber.from(randomBytes(32));
            let tx = await queue.connect(scrollChain).popCrossDomainMessage(0, count1, bitmap1);
            await expect(tx)
              .to.emit(queue, "DequeueTransaction")
              .withArgs(0, count1, bitmap1.and(constants.One.shl(count1).sub(1)));
            for (let i = 0; i < count1; i++) {
              expect(await queue.isMessageSkipped(i)).to.eq(bitmap1.shr(i).and(1).eq(1));
              expect(await queue.isMessageDropped(i)).to.eq(false);
            }

            // then pop `count2` messages
            const bitmap2 = BigNumber.from(randomBytes(32));
            tx = await queue.connect(scrollChain).popCrossDomainMessage(count1, count2, bitmap2);
            await expect(tx)
              .to.emit(queue, "DequeueTransaction")
              .withArgs(count1, count2, bitmap2.and(constants.One.shl(count2).sub(1)));
            for (let i = 0; i < count2; i++) {
              expect(await queue.isMessageSkipped(i + count1)).to.eq(bitmap2.shr(i).and(1).eq(1));
              expect(await queue.isMessageDropped(i + count1)).to.eq(false);
            }

            // last pop `count3` messages
            const bitmap3 = BigNumber.from(randomBytes(32));
            tx = await queue.connect(scrollChain).popCrossDomainMessage(count1 + count2, count3, bitmap3);
            await expect(tx)
              .to.emit(queue, "DequeueTransaction")
              .withArgs(count1 + count2, count3, bitmap3.and(constants.One.shl(count3).sub(1)));
            for (let i = 0; i < count3; i++) {
              expect(await queue.isMessageSkipped(i + count1 + count2)).to.eq(bitmap3.shr(i).and(1).eq(1));
              expect(await queue.isMessageDropped(i + count1 + count2)).to.eq(false);
            }
          });
        }
      }
    }
  });

  context("#dropCrossDomainMessage", async () => {
    it("should revert, when non-messenger call", async () => {
      await expect(queue.connect(signer).dropCrossDomainMessage(0)).to.revertedWith(
        "Only callable by the L1ScrollMessenger"
      );
    });

    it("should revert, when drop non-skipped message", async () => {
      // append 10 messages
      for (let i = 0; i < 10; i++) {
        await queue.connect(messenger).appendCrossDomainMessage(constants.AddressZero, 1000000, "0x");
      }
      // pop 5 messages with no skip
      await expect(queue.connect(scrollChain).popCrossDomainMessage(0, 5, 0))
        .to.emit(queue, "DequeueTransaction")
        .withArgs(0, 5, 0);
      for (let i = 0; i < 5; i++) {
        expect(await queue.isMessageSkipped(i)).to.eq(false);
        expect(await queue.isMessageDropped(i)).to.eq(false);
      }
      expect(await queue.pendingQueueIndex()).to.eq(5);

      for (let i = 0; i < 5; i++) {
        await expect(queue.connect(messenger).dropCrossDomainMessage(i)).to.revertedWith("drop non-skipped message");
      }

      // drop pending message
      for (let i = 6; i < 10; i++) {
        await expect(queue.connect(messenger).dropCrossDomainMessage(i)).to.revertedWith("cannot drop pending message");
      }
    });

    it("should succeed", async () => {
      // append 10 messages
      for (let i = 0; i < 10; i++) {
        await queue.connect(messenger).appendCrossDomainMessage(constants.AddressZero, 1000000, "0x");
      }
      // pop 10 messages, all skipped
      await expect(queue.connect(scrollChain).popCrossDomainMessage(0, 10, 0x3ff))
        .to.emit(queue, "DequeueTransaction")
        .withArgs(0, 10, 0x3ff);

      for (let i = 0; i < 10; i++) {
        expect(await queue.isMessageSkipped(i)).to.eq(true);
        expect(await queue.isMessageDropped(i)).to.eq(false);
        await expect(queue.connect(messenger).dropCrossDomainMessage(i)).to.emit(queue, "DropTransaction").withArgs(i);
        await expect(queue.connect(messenger).dropCrossDomainMessage(i)).to.revertedWith("message already dropped");
        expect(await queue.isMessageSkipped(i)).to.eq(true);
        expect(await queue.isMessageDropped(i)).to.eq(true);
      }
    });
  });
});
