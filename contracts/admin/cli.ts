import yargs from "yargs";
import { ethers } from "ethers";
import { DomainDeployment, getConfig } from "./config";
import { approveHash } from "./tx";

// eslint-disable-next-line no-unused-expressions
yargs
  .command(
    "approveHash",
    "approve transaction hash in SAFE",
    (yargs) =>
      yargs
        .options({
          network: {
            alias: "n",
            describe: "name of network config to use, eg: {mainnet | goerli | testnet}",
            string: true,
          },
          domain: {
            describe: "L1 or L2",
            string: true,
            coerce: (arg) => arg.toUpperCase(),
          },
          targetAddress: {
            describe: "address of contract to call",
            string: true,
          },
          targetCalldata: {
            describe: "calldata to send to contract",
            string: true,
          },
        })
        .check((argv) => {
          if (!(argv.targetAddress && argv.targetCalldata) && !(argv.network && argv.domain)) {
            throw new Error("Must provide network, domain, targetAddress and targetCalldata");
          }
          return true; // If no error was thrown, validation passed and you can return true
        }),
    async (argv) => {
      // todo: validate
      const targetAddress = ethers.getAddress(argv.targetAddress!);
      const targetCalldata = argv.targetCalldata!;
      console.log("using target value from args: ", { targetAddress, targetCalldata });

      const conf = getConfig(argv.network!, argv.domain!);

      const fragment = await approveHash(
        targetAddress,
        ethers.getBytes(targetCalldata),
        conf.ScrollSafeAddress,
        conf.ForwarderAddress,
        conf.ScrollTimelockAddress
      );
      console.log(fragment);
    }
  )
  .help().argv;
