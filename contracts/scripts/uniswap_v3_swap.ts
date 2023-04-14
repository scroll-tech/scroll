/* eslint-disable node/no-missing-import */
import { Contract, providers, utils, Wallet } from "ethers";
import * as dotenv from "dotenv";

import ERC20ABI from "./abi/ERC20.json";
import UniswapV3RouterABI from "./abi/UniswapV3Router.json";
import UniswapV3FactoryABI from "./abi/UniswapV3Factory.json";

dotenv.config();

async function main() {
  const provider = new providers.JsonRpcProvider("https://staging-rpc.scroll.io/l2");
  const fee = parseInt(process.env.FEE!);
  const signer = new Wallet(process.env.PRIVATE_KEY!, provider);
  const UniswapV3Router = new Contract("0xC18394cf6555B541Efdb83083F720eCB1dF4692e", UniswapV3RouterABI, signer);

  const USDC: string = "0x429C6C7b33Edb8cB19BD8cC0d1940B13Cca746C4";
  const WETH: string = await UniswapV3Router.WETH9();
  const factory: string = await UniswapV3Router.factory();
  const UniswapV3Factory = new Contract(factory, UniswapV3FactoryABI, signer);

  const usdc = new Contract(USDC, ERC20ABI, signer);
  const weth = new Contract(WETH, ERC20ABI, signer);

  const pool = await UniswapV3Factory.getPool(WETH, USDC, fee);

  console.log("Signer address:", signer.address);
  console.log("Pool address:", pool);
  console.log("USDC in pool:", (await usdc.balanceOf(pool)).toString());
  console.log("WETH in pool:", (await weth.balanceOf(pool)).toString());

  let nonce = await signer.getTransactionCount();
  for (let i = 0; i < 100; i++) {
    const data = UniswapV3Router.interface.encodeFunctionData("exactInputSingle", [
      {
        tokenIn: WETH,
        tokenOut: USDC,
        fee,
        recipient: signer.address,
        amountIn: utils.parseEther("0.0001"),
        amountOutMinimum: 0,
        sqrtPriceLimitX96: 0,
      },
    ]);
    const tx = await signer.sendTransaction({
      to: UniswapV3Router.address,
      data,
      nonce,
      value: utils.parseEther("0.0001"),
    });
    nonce = nonce + 1;
    console.log(tx);
    console.log("send tx with hash:", tx.hash, "nonce:", nonce);
  }
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
