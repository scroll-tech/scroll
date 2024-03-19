/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { ethers } from "hardhat";
import { GasSwap, ERC2771Forwarder, MockERC20, MockGasSwapTarget, MockGasSwapNormalPermit } from "../typechain";
import { SignerWithAddress } from "@nomiclabs/hardhat-ethers/signers";
import { expect } from "chai";
import { BigNumber, constants } from "ethers";
import { splitSignature } from "ethers/lib/utils";

describe("GasSwap.spec", async () => {
  let deployer: SignerWithAddress;
  let signer: SignerWithAddress;

  let forwarder: ERC2771Forwarder;
  let swap: GasSwap;
  let mockSwap: MockGasSwapNormalPermit;
  let target: MockGasSwapTarget;
  let token: MockERC20;

  beforeEach(async () => {
    [deployer, signer] = await ethers.getSigners();

    const ERC2771Forwarder = await ethers.getContractFactory("ERC2771Forwarder", deployer);
    forwarder = await ERC2771Forwarder.deploy("ERC2771Forwarder");
    await forwarder.deployed();

    const GasSwap = await ethers.getContractFactory("GasSwap", deployer);
    swap = await GasSwap.deploy(forwarder.address);
    await swap.deployed();

    const MockGasSwapTarget = await ethers.getContractFactory("MockGasSwapTarget", deployer);
    target = await MockGasSwapTarget.deploy();
    await target.deployed();

    const MockGasSwapNormalPermit = await ethers.getContractFactory("MockGasSwapNormalPermit", deployer);
    mockSwap = await MockGasSwapNormalPermit.deploy(forwarder.address);
    await mockSwap.deployed();

    const MockERC20 = await ethers.getContractFactory("MockERC20", deployer);
    token = await MockERC20.deploy("x", "y", 18);
    await token.deployed();
  });

  context("auth", async () => {
    it("should initialize correctly", async () => {
      expect(await swap.owner()).to.eq(deployer.address);
    });

    context("#updateFeeRatio", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(swap.connect(signer).updateFeeRatio(1)).to.revertedWith("caller is not the owner");
      });

      it("should succeed", async () => {
        expect(await swap.feeRatio()).to.eq(constants.AddressZero);
        await expect(swap.updateFeeRatio(100)).to.emit(swap, "UpdateFeeRatio").withArgs(100);
        expect(await swap.feeRatio()).to.eq(100);
      });
    });

    context("#updateApprovedTarget", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(swap.connect(signer).updateApprovedTarget(target.address, false)).to.revertedWith(
          "caller is not the owner"
        );
      });

      it("should succeed", async () => {
        expect(await swap.approvedTargets(target.address)).to.eq(false);
        await expect(swap.updateApprovedTarget(target.address, true))
          .to.emit(swap, "UpdateApprovedTarget")
          .withArgs(target.address, true);
        expect(await swap.approvedTargets(target.address)).to.eq(true);
        await expect(swap.updateApprovedTarget(target.address, false))
          .to.emit(swap, "UpdateApprovedTarget")
          .withArgs(target.address, false);
        expect(await swap.approvedTargets(target.address)).to.eq(false);
      });
    });

    context("#withdraw", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(swap.connect(signer).withdraw(constants.AddressZero, 0)).to.revertedWith(
          "caller is not the owner"
        );
      });

      it("should succeed, when withdraw ETH", async () => {
        await deployer.sendTransaction({ to: swap.address, value: ethers.utils.parseEther("1") });
        const balanceBefore = await deployer.getBalance();
        const tx = await swap.withdraw(constants.AddressZero, ethers.utils.parseEther("1"));
        const receipt = await tx.wait();
        const balanceAfter = await deployer.getBalance();
        expect(balanceAfter.sub(balanceBefore)).to.eq(
          ethers.utils.parseEther("1").sub(receipt.gasUsed.mul(receipt.effectiveGasPrice))
        );
      });

      it("should succeed, when withdraw token", async () => {
        await token.mint(swap.address, ethers.utils.parseEther("1"));
        const balanceBefore = await token.balanceOf(deployer.address);
        await swap.withdraw(token.address, ethers.utils.parseEther("1"));
        const balanceAfter = await token.balanceOf(deployer.address);
        expect(balanceAfter.sub(balanceBefore)).to.eq(ethers.utils.parseEther("1"));
      });
    });
  });

  const permit = async (amount: BigNumber) => {
    const value = {
      owner: signer.address,
      spender: swap.address,
      value: amount,
      nonce: await token.nonces(signer.address),
      deadline: constants.MaxUint256,
    };

    const domain = {
      name: await token.name(),
      version: "1",
      chainId: (await ethers.provider.getNetwork()).chainId,
      verifyingContract: token.address,
    };

    const types = {
      Permit: [
        {
          name: "owner",
          type: "address",
        },
        {
          name: "spender",
          type: "address",
        },
        {
          name: "value",
          type: "uint256",
        },
        {
          name: "nonce",
          type: "uint256",
        },
        {
          name: "deadline",
          type: "uint256",
        },
      ],
    };

    const signature = splitSignature(await signer._signTypedData(domain, types, value));
    return signature;
  };

  context("swap", async () => {
    it("should revert, when target not approved", async () => {
      await expect(
        swap.swap(
          {
            token: token.address,
            value: 0,
            deadline: 0,
            r: constants.HashZero,
            s: constants.HashZero,
            v: 0,
          },
          {
            target: target.address,
            data: "0x",
            minOutput: 0,
          }
        )
      ).to.revertedWith("target not approved");
    });

    it("should revert, when insufficient output amount", async () => {
      const amountIn = ethers.utils.parseEther("1");
      const amountOut = ethers.utils.parseEther("2");
      await token.mint(signer.address, amountIn);
      await deployer.sendTransaction({ to: target.address, value: amountOut });
      const signature = await permit(amountIn);

      await target.setToken(token.address);
      await target.setAmountIn(amountIn);

      await swap.updateApprovedTarget(target.address, true);
      await expect(
        swap.connect(signer).swap(
          {
            token: token.address,
            value: amountIn,
            deadline: constants.MaxUint256,
            r: signature.r,
            s: signature.s,
            v: signature.v,
          },
          {
            target: target.address,
            data: "0x8119c065",
            minOutput: amountOut.add(1),
          }
        )
      ).to.revertedWith("insufficient output amount");
    });

    it("should succeed, when attacker frontrun permit", async () => {
      const amountIn = ethers.utils.parseEther("1");
      const amountOut = ethers.utils.parseEther("2");
      await token.mint(signer.address, amountIn);
      await deployer.sendTransaction({ to: target.address, value: amountOut });
      const signature = await permit(amountIn);

      await token.permit(
        signer.address,
        swap.address,
        amountIn,
        constants.MaxUint256,
        signature.v,
        signature.r,
        signature.s
      );

      await target.setToken(token.address);
      await target.setAmountIn(amountIn);

      await swap.updateApprovedTarget(target.address, true);
      await swap.connect(signer).swap(
        {
          token: token.address,
          value: amountIn,
          deadline: constants.MaxUint256,
          r: signature.r,
          s: signature.s,
          v: signature.v,
        },
        {
          target: target.address,
          data: "0x8119c065",
          minOutput: 0,
        }
      );
    });

    it("should failed, when attacker frontrun normal gasSwap permit", async () => {
      const amountIn = ethers.utils.parseEther("1");
      const amountOut = ethers.utils.parseEther("2");
      await token.mint(signer.address, amountIn);
      await deployer.sendTransaction({ to: target.address, value: amountOut });
      const signature = await permit(amountIn);

      await token.permit(
        signer.address,
        mockSwap.address,
        amountIn,
        constants.MaxUint256,
        signature.v,
        signature.r,
        signature.s
      );

      await target.setToken(token.address);
      await target.setAmountIn(amountIn);

      await mockSwap.updateApprovedTarget(target.address, true);
      await mockSwap.connect(signer).swap(
        {
          token: token.address,
          value: amountIn,
          deadline: constants.MaxUint256,
          r: signature.r,
          s: signature.s,
          v: signature.v,
        },
        {
          target: target.address,
          data: "0x8119c065",
          minOutput: 0,
        }
      );
    });

    for (const refundRatio of ["0", "1", "5"]) {
      for (const feeRatio of ["0", "5", "50"]) {
        it(`should succeed, when swap by signer directly, with feeRatio[${feeRatio}%] refundRatio[${refundRatio}%]`, async () => {
          const amountIn = ethers.utils.parseEther("1");
          const amountOut = ethers.utils.parseEther("2");
          await token.mint(signer.address, amountIn);
          await deployer.sendTransaction({ to: target.address, value: amountOut });
          const signature = await permit(amountIn);

          await target.setToken(token.address);
          await target.setAmountIn(amountIn);
          await target.setRefund(amountIn.mul(refundRatio).div(100));

          await swap.updateApprovedTarget(target.address, true);
          await swap.updateFeeRatio(ethers.utils.parseEther(feeRatio).div(100));
          const fee = amountOut.mul(feeRatio).div(100);

          const balanceBefore = await signer.getBalance();
          const tx = await swap.connect(signer).swap(
            {
              token: token.address,
              value: amountIn,
              deadline: constants.MaxUint256,
              r: signature.r,
              s: signature.s,
              v: signature.v,
            },
            {
              target: target.address,
              data: "0x8119c065",
              minOutput: amountOut.sub(fee),
            }
          );
          const receipt = await tx.wait();
          const balanceAfter = await signer.getBalance();
          expect(balanceAfter.sub(balanceBefore)).to.eq(
            amountOut.sub(fee).sub(receipt.gasUsed.mul(receipt.effectiveGasPrice))
          );
          expect(await token.balanceOf(signer.address)).to.eq(amountIn.mul(refundRatio).div(100));
        });

        it(`should succeed, when swap by signer with forwarder, with feeRatio[${feeRatio}%] refundRatio[${refundRatio}%]`, async () => {
          const amountIn = ethers.utils.parseEther("1");
          const amountOut = ethers.utils.parseEther("2");
          await token.mint(signer.address, amountIn);
          await deployer.sendTransaction({ to: target.address, value: amountOut });
          const permitSignature = await permit(amountIn);

          await target.setToken(token.address);
          await target.setAmountIn(amountIn);
          await target.setRefund(amountIn.mul(refundRatio).div(100));

          await swap.updateApprovedTarget(target.address, true);
          await swap.updateFeeRatio(ethers.utils.parseEther(feeRatio).div(100));
          const fee = amountOut.mul(feeRatio).div(100);

          const reqWithoutSignature = {
            from: signer.address,
            to: swap.address,
            value: constants.Zero,
            gas: 1000000,
            nonce: await forwarder.nonces(signer.address),
            deadline: 2000000000,
            data: swap.interface.encodeFunctionData("swap", [
              {
                token: token.address,
                value: amountIn,
                deadline: constants.MaxUint256,
                r: permitSignature.r,
                s: permitSignature.s,
                v: permitSignature.v,
              },
              {
                target: target.address,
                data: "0x8119c065",
                minOutput: amountOut.sub(fee),
              },
            ]),
          };

          const signature = await signer._signTypedData(
            {
              name: "ERC2771Forwarder",
              version: "1",
              chainId: (await ethers.provider.getNetwork()).chainId,
              verifyingContract: forwarder.address,
            },
            {
              ForwardRequest: [
                {
                  name: "from",
                  type: "address",
                },
                {
                  name: "to",
                  type: "address",
                },
                {
                  name: "value",
                  type: "uint256",
                },
                {
                  name: "gas",
                  type: "uint256",
                },
                {
                  name: "nonce",
                  type: "uint256",
                },
                {
                  name: "deadline",
                  type: "uint48",
                },
                {
                  name: "data",
                  type: "bytes",
                },
              ],
            },
            reqWithoutSignature
          );

          const balanceBefore = await signer.getBalance();
          await forwarder.execute({
            from: reqWithoutSignature.from,
            to: reqWithoutSignature.to,
            value: reqWithoutSignature.value,
            gas: reqWithoutSignature.gas,
            deadline: reqWithoutSignature.deadline,
            data: reqWithoutSignature.data,
            signature,
          });
          const balanceAfter = await signer.getBalance();
          expect(balanceAfter.sub(balanceBefore)).to.eq(amountOut.sub(fee));
          expect(await token.balanceOf(signer.address)).to.eq(amountIn.mul(refundRatio).div(100));
        });
      }
    }
  });
});
