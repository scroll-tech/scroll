import { ethers } from "ethers";
import { Safeabi__factory, Forwarder__factory, Target__factory, Timelock__factory } from "../safeAbi";

const L2_SCROLL_SAFE_ADDR = "0xa513E6E4b8f2a923D98304ec87F64353C4D5C853";
const L2_SCROLL_TIMELOCK_ADDR = "0x8A791620dd6260079BF849Dc5567aDC3F2FdC318";
const L2_FORWARDER_ADDR = "0xA51c1fc2f0D1a1b8494Ed1FE312d7C3a78Ed91C0";
const L2_TARGET_ADDR = "0x0DCd1Bf9A1b36cE34237eEaFef220932846BCD82";
const L2_DEPLOYER_PRIVATE_KEY = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80";

/* 
TODO:
* read from env
* use approve hash flow
* read nonce from safe
* split script into schedule and execute
* add gas limit
* document how to use 
* how to get addresses from deployment? 
* get abis in a reasonable way
*/

/* 
to get safe abi
* forge build 
* cat artifacts/src/Safe.sol/Safe.json| jq .abi >> safeabi.json
* mkdir safeAbi
* npx typechain --target=ethers-v5 safeabi.json --out-dir safeAbi

repeat for forwarder, timelock, target
*/

async function main() {
  const provider = new ethers.providers.JsonRpcProvider("http://localhost:1234");
  const wallet = new ethers.Wallet(L2_DEPLOYER_PRIVATE_KEY, provider);

  const safeContract = Safeabi__factory.connect(L2_SCROLL_SAFE_ADDR, provider);
  const forwarderContract = Forwarder__factory.connect(L2_FORWARDER_ADDR, provider);
  const timelockContract = Timelock__factory.connect(L2_SCROLL_TIMELOCK_ADDR, provider);
  const targetContract = Target__factory.connect(L2_TARGET_ADDR, provider);

  const targetCalldata = targetContract.interface.encodeFunctionData("err");
  const forwarderCalldata = forwarderContract.interface.encodeFunctionData("forward", [L2_TARGET_ADDR, targetCalldata]);
  const timelockScheduleCalldata = timelockContract.interface.encodeFunctionData("schedule", [
    L2_FORWARDER_ADDR,
    0,
    forwarderCalldata,
    ethers.constants.HashZero,
    ethers.constants.HashZero,
    0,
  ]);

  const scheduleSafeTxHash = await safeContract.getTransactionHash(
    L2_SCROLL_TIMELOCK_ADDR,
    0,
    timelockScheduleCalldata,
    0,
    0,
    0,
    0,
    ethers.constants.AddressZero,
    ethers.constants.AddressZero,
    0
  );

  const sigRawSchedule = await wallet.signMessage(ethers.utils.arrayify(scheduleSafeTxHash));
  const sigSchedule = editSig(sigRawSchedule);

  await safeContract.checkNSignatures(scheduleSafeTxHash, ethers.utils.arrayify("0x00"), sigSchedule, 1);

  await safeContract
    .connect(wallet)
    .execTransaction(
      L2_SCROLL_TIMELOCK_ADDR,
      0,
      timelockScheduleCalldata,
      0,
      0,
      0,
      0,
      ethers.constants.AddressZero,
      ethers.constants.AddressZero,
      sigSchedule,
      { gasLimit: 1000000 }
    );
  console.log("scheduled");

  await timelockContract
    .connect(wallet)
    .execute(L2_FORWARDER_ADDR, 0, forwarderCalldata, ethers.constants.HashZero, ethers.constants.HashZero, {
      gasLimit: 1000000,
    });
}

//  add 4 to the v byte at the end of the signature
function editSig(sig: string) {
  const v = parseInt(sig.slice(-2), 16);
  const newV = v + 4;
  const newSig = sig.slice(0, -2) + newV.toString(16);
  return newSig;
}

main();
