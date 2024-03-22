/* eslint-disable node/no-missing-import */
/* eslint-disable node/no-unpublished-import */
import { expect } from "chai";
import { randomBytes } from "crypto";
import { Contract, toBigInt } from "ethers";
import fs from "fs";
import { ethers } from "hardhat";

import PoseidonWithoutDomain from "circomlib/src/poseidon_gencontract";
import { generateABI, createCode } from "../scripts/poseidon";

describe("PoseidonHash.spec", async () => {
  // test against with circomlib's implementation.
  context("domain = zero", async () => {
    let poseidonCircom: Contract;
    let poseidon: Contract;

    beforeEach(async () => {
      const [deployer] = await ethers.getSigners();

      const PoseidonWithoutDomainFactory = new ethers.ContractFactory(
        PoseidonWithoutDomain.generateABI(2),
        PoseidonWithoutDomain.createCode(2),
        deployer
      );
      poseidonCircom = (await PoseidonWithoutDomainFactory.deploy()) as Contract;

      const PoseidonWithDomainFactory = new ethers.ContractFactory(generateABI(2), createCode(2), deployer);
      poseidon = (await PoseidonWithDomainFactory.deploy()) as Contract;
    });

    it("should succeed on zero inputs", async () => {
      expect(await poseidonCircom["poseidon(uint256[2])"]([0, 0])).to.eq(
        await poseidon["poseidon(uint256[2],uint256)"]([0, 0], 0)
      );
    });

    it("should succeed on random inputs", async () => {
      for (let bytes = 1; bytes <= 32; ++bytes) {
        for (let i = 0; i < 5; ++i) {
          const a = toBigInt(randomBytes(bytes));
          const b = toBigInt(randomBytes(bytes));
          expect(await poseidonCircom["poseidon(uint256[2])"]([a, b])).to.eq(
            await poseidon["poseidon(uint256[2],uint256)"]([a, b], 0)
          );
          expect(await poseidonCircom["poseidon(uint256[2])"]([a, 0])).to.eq(
            await poseidon["poseidon(uint256[2],uint256)"]([a, 0], 0)
          );
          expect(await poseidonCircom["poseidon(uint256[2])"]([0, b])).to.eq(
            await poseidon["poseidon(uint256[2],uint256)"]([0, b], 0)
          );
        }
      }
    });
  });

  // test against with scroll's go implementation.
  context("domain = nonzero", async () => {
    let poseidon: Contract;

    beforeEach(async () => {
      const [deployer] = await ethers.getSigners();
      const PoseidonWithDomainFactory = new ethers.ContractFactory(generateABI(2), createCode(2), deployer);
      poseidon = (await PoseidonWithDomainFactory.deploy()) as Contract;
    });

    it("should succeed on zero inputs", async () => {
      expect(await poseidon["poseidon(uint256[2],uint256)"]([0, 0], 6)).to.eq(
        toBigInt("17848312925884193353134534408113064827548730776291701343555436351962284922129")
      );
      expect(await poseidon["poseidon(uint256[2],uint256)"]([0, 0], 7)).to.eq(
        toBigInt("20994231331856095272861976502721128670019193481895476667943874333621461724676")
      );
    });

    it("should succeed on random inputs", async () => {
      const lines = String(fs.readFileSync("./integration-test/testdata/poseidon_hash_with_domain.data")).split("\n");
      for (const line of lines) {
        const [domain, a, b, hash] = line.split(" ");
        expect(await poseidon["poseidon(uint256[2],uint256)"]([a, b], domain)).to.eq(toBigInt(hash));
      }
    });
  });
});
