/* eslint-disable node/no-missing-import */
import { ethers } from "ethers";

import Contract from "circomlib/src/evmasm";
import * as constants from "circomlib/src/poseidon_constants";

const N_ROUNDS_F = 8;
const N_ROUNDS_P = [56, 57, 56, 60, 60, 63, 64, 63];

export function createCode(nInputs: number) {
  if (nInputs < 1 || nInputs > 8) throw new Error("Invalid number of inputs. Must be 1<=nInputs<=8");
  const t = nInputs + 1;
  const nRoundsF = N_ROUNDS_F;
  const nRoundsP = N_ROUNDS_P[t - 2];

  const C = new Contract();

  function saveM() {
    for (let i = 0; i < t; i++) {
      for (let j = 0; j < t; j++) {
        C.push(constants.M[t - 2][i][j]);
        C.push((1 + i * t + j) * 32);
        C.mstore();
      }
    }
  }

  function ark(r: number) {
    // st, q
    for (let i = 0; i < t; i++) {
      C.dup(t); // q, st, q
      C.push(constants.C[t - 2][r * t + i]); // K, q, st, q
      C.dup(2 + i); // st[i], K, q, st, q
      C.addmod(); // newSt[i], st, q
      C.swap(1 + i); // xx, st, q
      C.pop();
    }
  }

  function sigma(p: number) {
    // sq, q
    C.dup(t); // q, st, q
    C.dup(1 + p); // st[p] , q , st, q
    C.dup(1); // q, st[p] , q , st, q
    C.dup(0); // q, q, st[p] , q , st, q
    C.dup(2); // st[p] , q, q, st[p] , q , st, q
    C.dup(0); // st[p] , st[p] , q, q, st[p] , q , st, q
    C.mulmod(); // st2[p], q, st[p] , q , st, q
    C.dup(0); // st2[p], st2[p], q, st[p] , q , st, q
    C.mulmod(); // st4[p], st[p] , q , st, q
    C.mulmod(); // st5[p], st, q
    C.swap(1 + p);
    C.pop(); // newst, q
  }

  function mix() {
    C.label("mix");
    for (let i = 0; i < t; i++) {
      for (let j = 0; j < t; j++) {
        if (j === 0) {
          C.dup(i + t); // q, newSt, oldSt, q
          C.push((1 + i * t + j) * 32);
          C.mload(); // M, q, newSt, oldSt, q
          C.dup(2 + i + j); // oldSt[j], M, q, newSt, oldSt, q
          C.mulmod(); // acc, newSt, oldSt, q
        } else {
          C.dup(1 + i + t); // q, acc, newSt, oldSt, q
          C.push((1 + i * t + j) * 32);
          C.mload(); // M, q, acc, newSt, oldSt, q
          C.dup(3 + i + j); // oldSt[j], M, q, acc, newSt, oldSt, q
          C.mulmod(); // aux, acc, newSt, oldSt, q
          C.dup(2 + i + t); // q, aux, acc, newSt, oldSt, q
          C.swap(2); // acc, aux, q, newSt, oldSt, q
          C.addmod(); // acc, newSt, oldSt, q
        }
      }
    }
    for (let i = 0; i < t; i++) {
      C.swap(t - i + (t - i - 1));
      C.pop();
    }
    C.push(0);
    C.mload();
    C.jmp();
  }

  // Check selector
  C.push("0x0100000000000000000000000000000000000000000000000000000000");
  C.push(0);
  C.calldataload();
  C.div();
  C.dup(0);
  C.push(ethers.utils.keccak256(ethers.utils.toUtf8Bytes(`poseidon(uint256[${nInputs}],uint256)`)).slice(0, 10)); // poseidon(uint256[n],uint256)
  C.eq();
  C.swap(1);
  C.push(ethers.utils.keccak256(ethers.utils.toUtf8Bytes(`poseidon(bytes32[${nInputs}],bytes32)`)).slice(0, 10)); // poseidon(bytes32[n],bytes32)
  C.eq();
  C.or();
  C.jmpi("start");
  C.invalid();

  C.label("start");

  saveM();

  C.push("0x30644e72e131a029b85045b68181585d2833e84879b9709143e1f593f0000001"); // q

  // Load t values from the call data.
  // The function has a single array param param
  // [Selector (4)] [item1 (32)] [item2 (32)] .... [doman (32)]
  // Stack positions 0-nInputs.
  for (let i = 0; i < nInputs; i++) {
    C.push(0x04 + 0x20 * (nInputs - i - 1));
    C.calldataload();
  }
  C.push(0x04 + 0x20 * nInputs);
  C.calldataload();

  for (let i = 0; i < nRoundsF + nRoundsP; i++) {
    ark(i);
    if (i < nRoundsF / 2 || i >= nRoundsP + nRoundsF / 2) {
      for (let j = 0; j < t; j++) {
        sigma(j);
      }
    } else {
      sigma(0);
    }
    const strLabel = "aferMix" + i;
    C._pushLabel(strLabel);
    C.push(0);
    C.mstore();
    C.jmp("mix");
    C.label(strLabel);
  }

  C.push("0x00");
  C.mstore(); // Save it to pos 0;
  C.push("0x20");
  C.push("0x00");
  C.return();

  mix();

  return C.createTxData();
}

export function generateABI(nInputs: number) {
  return [
    {
      constant: true,
      inputs: [
        {
          internalType: `bytes32[${nInputs}]`,
          name: "input",
          type: `bytes32[${nInputs}]`,
        },
        {
          internalType: "bytes32",
          name: "domain",
          type: "bytes32",
        },
      ],
      name: "poseidon",
      outputs: [
        {
          internalType: "bytes32",
          name: "",
          type: "bytes32",
        },
      ],
      payable: false,
      stateMutability: "pure",
      type: "function",
    },
    {
      constant: true,
      inputs: [
        {
          internalType: `uint256[${nInputs}]`,
          name: "input",
          type: `uint256[${nInputs}]`,
        },
        {
          internalType: "uint256",
          name: "domain",
          type: "uint256",
        },
      ],
      name: "poseidon",
      outputs: [
        {
          internalType: "uint256",
          name: "",
          type: "uint256",
        },
      ],
      payable: false,
      stateMutability: "pure",
      type: "function",
    },
  ];
}
