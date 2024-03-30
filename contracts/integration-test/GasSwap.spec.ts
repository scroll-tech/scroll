/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { HardhatEthersSigner } from "@nomicfoundation/hardhat-ethers/signers";
import { expect } from "chai";
import { MaxUint256, Signature, ZeroAddress, ZeroHash, toBigInt } from "ethers";
import { ethers } from "hardhat";

import { GasSwap, ERC2771Forwarder, MockERC20, MockGasSwapTarget } from "../typechain";

describe("GasSwap.spec", async () => {
  let deployer: HardhatEthersSigner;
  let signer: HardhatEthersSigner;

  let forwarder: ERC2771Forwarder;
  let swap: GasSwap;
  let target: MockGasSwapTarget;
  let token: MockERC20;

  beforeEach(async () => {
    [deployer, signer] = await ethers.getSigners();

    const ERC2771Forwarder = await ethers.getContractFactory("ERC2771Forwarder", deployer);
    forwarder = await ERC2771Forwarder.deploy("ERC2771Forwarder");

    const GasSwap = await ethers.getContractFactory("GasSwap", deployer);
    swap = await GasSwap.deploy(forwarder.getAddress());

    const MockGasSwapTarget = await ethers.getContractFactory("MockGasSwapTarget", deployer);
    target = await MockGasSwapTarget.deploy();

    const MockERC20 = await ethers.getContractFactory("MockERC20", deployer);
    token = await MockERC20.deploy("x", "y", 18);
  });

  context("auth", async () => {
    it("should initialize correctly", async () => {
      expect(await swap.owner()).to.eq(deployer.address);
    });

    context("#updateFeeRatio", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(swap.connect(signer).updateFeeRatio(1)).to.revertedWith("Ownable: caller is not the owner");
      });

      it("should succeed", async () => {
        expect(await swap.feeRatio()).to.eq(ZeroAddress);
        await expect(swap.updateFeeRatio(100)).to.emit(swap, "UpdateFeeRatio").withArgs(100);
        expect(await swap.feeRatio()).to.eq(100);
      });
    });

    context("#updateApprovedTarget", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(swap.connect(signer).updateApprovedTarget(target.getAddress(), false)).to.revertedWith(
          "Ownable: caller is not the owner"
        );
      });

      it("should succeed", async () => {
        expect(await swap.approvedTargets(target.getAddress())).to.eq(false);
        await expect(swap.updateApprovedTarget(target.getAddress(), true))
          .to.emit(swap, "UpdateApprovedTarget")
          .withArgs(await target.getAddress(), true);
        expect(await swap.approvedTargets(target.getAddress())).to.eq(true);
        await expect(swap.updateApprovedTarget(target.getAddress(), false))
          .to.emit(swap, "UpdateApprovedTarget")
          .withArgs(await target.getAddress(), false);
        expect(await swap.approvedTargets(target.getAddress())).to.eq(false);
      });
    });

    context("#withdraw", async () => {
      it("should revert, when non-owner call", async () => {
        await expect(swap.connect(signer).withdraw(ZeroAddress, 0)).to.revertedWith("Ownable: caller is not the owner");
      });

      it("should succeed, when withdraw ETH", async () => {
        await deployer.sendTransaction({ to: swap.getAddress(), value: ethers.parseEther("1") });
        const balanceBefore = await ethers.provider.getBalance(deployer.address);
        const tx = await swap.withdraw(ZeroAddress, ethers.parseEther("1"));
        const receipt = await tx.wait();
        const balanceAfter = await ethers.provider.getBalance(deployer.address);
        expect(balanceAfter - balanceBefore).to.eq(ethers.parseEther("1") - receipt!.gasUsed * receipt!.gasPrice);
      });

      it("should succeed, when withdraw token", async () => {
        await token.mint(swap.getAddress(), ethers.parseEther("1"));
        const balanceBefore = await token.balanceOf(deployer.address);
        await swap.withdraw(token.getAddress(), ethers.parseEther("1"));
        const balanceAfter = await token.balanceOf(deployer.address);
        expect(balanceAfter - balanceBefore).to.eq(ethers.parseEther("1"));
      });
    });
  });

  const permit = async (amount: bigint) => {
    const value = {
      owner: signer.address,
      spender: await swap.getAddress(),
      value: amount,
      nonce: await token.nonces(signer.address),
      deadline: MaxUint256,
    };

    const domain = {
      name: await token.name(),
      version: "1",
      chainId: (await ethers.provider.getNetwork()).chainId,
      verifyingContract: await token.getAddress(),
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

    const signature = Signature.from(await signer.signTypedData(domain, types, value));
    return signature;
  };

  context("swap", async () => {
    it("should revert, when target not approved", async () => {
      await expect(
        swap.swap(
          {
            token: token.getAddress(),
            value: 0,
            deadline: 0,
            r: ZeroHash,
            s: ZeroHash,
            v: 0,
          },
          {
            target: target.getAddress(),
            data: "0x",
            minOutput: 0,
          }
        )
      ).to.revertedWith("target not approved");
    });

    it("should revert, when insufficient output amount", async () => {
      const amountIn = ethers.parseEther("1");
      const amountOut = ethers.parseEther("2");
      await token.mint(signer.address, amountIn);
      await deployer.sendTransaction({ to: target.getAddress(), value: amountOut });
      const signature = await permit(amountIn);

      await target.setToken(token.getAddress());
      await target.setAmountIn(amountIn);

      await swap.updateApprovedTarget(target.getAddress(), true);
      await expect(
        swap.connect(signer).swap(
          {
            token: await token.getAddress(),
            value: amountIn,
            deadline: MaxUint256,
            r: signature.r,
            s: signature.s,
            v: signature.v,
          },
          {
            target: target.getAddress(),
            data: "0x8119c065",
            minOutput: amountOut + 1n,
          }
        )
      ).to.revertedWith("insufficient output amount");
    });

    it("should succeed, when attacker frontrun permit", async () => {
      const amountIn = ethers.parseEther("1");
      const amountOut = ethers.parseEther("2");
      await token.mint(signer.address, amountIn);
      await deployer.sendTransaction({ to: target.getAddress(), value: amountOut });
      const signature = await permit(amountIn);

      await token.permit(
        signer.address,
        swap.getAddress(),
        amountIn,
        MaxUint256,
        signature.v,
        signature.r,
        signature.s
      );

      await target.setToken(token.getAddress());
      await target.setAmountIn(amountIn);

      await swap.updateApprovedTarget(target.getAddress(), true);

      swap.connect(signer).swap(
        {
          token: token.getAddress(),
          value: amountIn,
          deadline: MaxUint256,
          r: signature.r,
          s: signature.s,
          v: signature.v,
        },
        {
          target: target.getAddress(),
          data: "0x8119c065",
          minOutput: 0,
        }
      );
    });

    for (const refundRatio of [0n, 1n, 5n]) {
      for (const feeRatio of ["0", "5", "50"]) {
        it(`should succeed, when swap by signer directly, with feeRatio[${feeRatio}%] refundRatio[${refundRatio}%]`, async () => {
          const amountIn = ethers.parseEther("1");
          const amountOut = ethers.parseEther("2");
          await token.mint(signer.address, amountIn);
          await deployer.sendTransaction({ to: target.getAddress(), value: amountOut });
          const signature = await permit(amountIn);

          await target.setToken(token.getAddress());
          await target.setAmountIn(amountIn);
          await target.setRefund((amountIn * refundRatio) / 100n);

          await swap.updateApprovedTarget(target.getAddress(), true);
          await swap.updateFeeRatio(ethers.parseEther(feeRatio) / 100n);
          const fee = (amountOut * toBigInt(feeRatio)) / 100n;

          const balanceBefore = await ethers.provider.getBalance(signer.address);
          const tx = await swap.connect(signer).swap(
            {
              token: await token.getAddress(),
              value: amountIn,
              deadline: MaxUint256,
              r: signature.r,
              s: signature.s,
              v: signature.v,
            },
            {
              target: target.getAddress(),
              data: "0x8119c065",
              minOutput: amountOut - fee,
            }
          );
          const receipt = await tx.wait();
          const balanceAfter = await ethers.provider.getBalance(signer.address);
          expect(balanceAfter - balanceBefore).to.eq(amountOut - fee - receipt!.gasUsed * receipt!.gasPrice);
          expect(await token.balanceOf(signer.address)).to.eq((amountIn * refundRatio) / 100n);
        });

        it(`should succeed, when swap by signer with forwarder, with feeRatio[${feeRatio}%] refundRatio[${refundRatio}%]`, async () => {
          const amountIn = ethers.parseEther("1");
          const amountOut = ethers.parseEther("2");
          await token.mint(signer.address, amountIn);
          await deployer.sendTransaction({ to: await target.getAddress(), value: amountOut });
          const permitSignature = await permit(amountIn);

          await target.setToken(token.getAddress());
          await target.setAmountIn(amountIn);
          await target.setRefund((amountIn * refundRatio) / 100n);

          await swap.updateApprovedTarget(target.getAddress(), true);
          await swap.updateFeeRatio(ethers.parseEther(feeRatio) / 100n);
          const fee = (amountOut * toBigInt(feeRatio)) / 100n;

          const reqWithoutSignature = {
            from: signer.address,
            to: await swap.getAddress(),
            value: 0n,
            gas: 1000000,
            nonce: await forwarder.nonces(signer.address),
            deadline: 2000000000,
            data: swap.interface.encodeFunctionData("swap", [
              {
                token: await token.getAddress(),
                value: amountIn,
                deadline: MaxUint256,
                r: permitSignature.r,
                s: permitSignature.s,
                v: permitSignature.v,
              },
              {
                target: await target.getAddress(),
                data: "0x8119c065",
                minOutput: amountOut - fee,
              },
            ]),
          };

          const signature = await signer.signTypedData(
            {
              name: "ERC2771Forwarder",
              version: "1",
              chainId: (await ethers.provider.getNetwork()).chainId,
              verifyingContract: await forwarder.getAddress(),
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

          const balanceBefore = await ethers.provider.getBalance(signer.address);
          await forwarder.execute({
            from: reqWithoutSignature.from,
            to: reqWithoutSignature.to,
            value: reqWithoutSignature.value,
            gas: reqWithoutSignature.gas,
            deadline: reqWithoutSignature.deadline,
            data: reqWithoutSignature.data,
            signature,
          });
          const balanceAfter = await ethers.provider.getBalance(signer.address);
          expect(balanceAfter - balanceBefore).to.eq(amountOut - fee);
          expect(await token.balanceOf(signer.address)).to.eq((amountIn * refundRatio) / 100n);
        });
      }
    }
  });
});
