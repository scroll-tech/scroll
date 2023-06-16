import { ethers } from "ethers";
import { SafeAbi__factory, SafeAbi } from "../safeAbi";

/* 
to get safe abi
* forge build 
* cat artifacts/src/Safe.sol/Safe.json| jq .abi >> safeabi.json
* mkdir safeAbi
* npx typechain --target=ethers-v5 artifacts/src/Safe.sol/Safe.json --out-dir safeAbi

*/

async function main() {
  const provider = new ethers.providers.JsonRpcProvider("http://localhost:1234");
  const safeAddress = "0xa513E6E4b8f2a923D98304ec87F64353C4D5C853";
  const safe = SafeAbi__factory.connect(safeAddress, provider);

  const wallet = new ethers.Wallet("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80");
  const message = "Hello, world!";
  const dataHash = ethers.utils.hashMessage(message);
  console.log(dataHash);
  const sigRaw = await wallet.signMessage(ethers.utils.arrayify(dataHash));
  const sig = editSig(sigRaw);

  await safe.checkNSignatures(dataHash, ethers.utils.arrayify("0x00"), sig, 1);
}

//  add 4 to the v byte at the end of the signature
function editSig(sig: string) {
  const v = parseInt(sig.slice(-2), 16);
  const newV = v + 4;
  const newSig = sig.slice(0, -2) + newV.toString(16);
  return newSig;
}

main();
