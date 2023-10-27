/* eslint-disable node/no-missing-import */
import * as dotenv from "dotenv";

import { constants } from "ethers";
import * as hre from "hardhat";
import { ethers } from "hardhat";
import { selectAddressFile } from "./utils";

dotenv.config();

//    // initialize L2MessageQueue
//    L2MessageQueue(L2_MESSAGE_QUEUE_ADDR).initialize(L2_SCROLL_MESSENGER_PROXY_ADDR);

//    // initialize L2TxFeeVault
//    L2TxFeeVault(payable(L2_TX_FEE_VAULT_ADDR)).updateMessenger(L2_SCROLL_MESSENGER_PROXY_ADDR);

//    // initialize L1GasPriceOracle
//    L1GasPriceOracle(L1_GAS_PRICE_ORACLE_ADDR).updateWhitelist(L2_WHITELIST_ADDR);


async function main() {
    const addressFile = selectAddressFile(hre.network.name);
    const addressFileL2 = selectAddressFile("l1geth");

    const [deployer] = await ethers.getSigners();
 
    // initialize L2ScrollMessenger
    const L2ScrollMessenger = await ethers.getContractAt(
        "L2ScrollMessengengeQueue",
        addressFile.get("L2ScrollMessengengeQueue.proxy"),
        deployer
    );
    const L1_SCROLL_MESSENGER_PROXY_ADDR = addressFileL2.get("L1ScrollMessenger.proxy")
    const tx = await L2ScrollMessenger.initialize(L1_SCROLL_MESSENGER_PROXY_ADDR)
    console.log("initialize L2ScrollMessenger, hash:", tx.hash);
    const receipt = await tx.wait();
    console.log(`✅ Done, gas used: ${receipt.gasUsed}`);

}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});
