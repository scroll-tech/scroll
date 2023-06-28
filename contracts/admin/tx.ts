import { ethers } from "ethers";
import {
  Safe__factory,
  Safe,
  Forwarder__factory,
  Forwarder,
  Timelock__factory,
  Timelock,
} from "./types/ethers-contracts";

export interface RawTxFragment {
  to: string;
  callData: string;
  functionSig: string;
}

async function execTransaction(wallet: ethers.Wallet, safeContract: Safe, calldata: string, senders: string[]) {
  // ethers.AbiCoder.encode(
  // Safe__factory.abi
  let signatures = "0x0000000000000000000000000000000000000000";
  for (let i = 0; i < senders.length; i++) {
    signatures += encodeAddress(senders[i]);
  }

  await safeContract
    .connect(wallet)
    .execTransaction(
      "0x0000000000000000000000000000000000000000",
      0,
      calldata,
      0,
      0,
      0,
      0,
      ethers.ZeroAddress,
      ethers.ZeroAddress,
      signatures,
      { gasLimit: 1000000 }
    );
}

export async function approveHash(
  targetAddress: ethers.AddressLike,
  targetCalldata: ethers.BytesLike,
  safeAddress: ethers.AddressLike,
  forwarderAddress: ethers.AddressLike,
  timelockAddress: ethers.AddressLike
): Promise<RawTxFragment> {
  // either implement getTransactionHash in JS or make RPC call to get hash
  const provider = new ethers.JsonRpcProvider("http://localhost:1234");
  const safeContract = Safe__factory.connect(safeAddress.toString(), provider);
  const forwarderContract = Forwarder__factory.connect(forwarderAddress.toString());
  const timelockContract = Timelock__factory.connect(timelockAddress.toString());
  // const targetCalldata = targetContract.interface.encodeFunctionData("err");
  const forwarderCalldata = forwarderContract.interface.encodeFunctionData("forward", [
    targetAddress.toString(),
    targetCalldata,
  ]);
  const timelockScheduleCalldata = timelockContract.interface.encodeFunctionData("schedule", [
    forwarderAddress.toString(),
    0,
    forwarderCalldata,
    ethers.ZeroHash,
    ethers.ZeroHash,
    0,
  ]);
  const txHash = await safeContract.getTransactionHash(
    timelockAddress.toString(),
    0,
    timelockScheduleCalldata,
    0,
    0,
    0,
    0,
    ethers.ZeroAddress,
    ethers.ZeroAddress,
    0
  );

  return {
    to: safeAddress.toString(),
    callData: txHash,
    functionSig: "approveHash(bytes32)",
  };
}
// await safeContract.checkNSignatures(scheduleSafeTxHash, ethers.arrayify("0x00"), sigSchedule, 1);
// await timelockContract
//   .connect(wallet)
//   .execute(L2_FORWARDER_ADDR, 0, forwarderCalldata, ethers.HashZero, ethers.HashZero, {
//     gasLimit: 1000000,
//   });

// safe takes address as part of the signature
function encodeAddress(address: string) {
  const r = ethers.zeroPadValue(address, 32);
  const s = ethers.zeroPadValue("0x00", 32);
  const v = "0x01";
  return ethers.toBeHex(ethers.concat([r, s, v])).slice(-2);
}

//  add 4 to the v byte at the end of the signature
function editSig(sig: string) {
  const v = parseInt(sig.slice(-2), 16);
  const newV = v + 4;
  const newSig = sig.slice(0, -2) + newV.toString(16);
  return newSig;
}

console.log(encodeAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"));

module.exports = {
  approveHash,
};
