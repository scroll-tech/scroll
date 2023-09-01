import * as dotenv from "dotenv";

import { HardhatUserConfig, subtask } from "hardhat/config";
import * as toml from "toml";
import "@nomiclabs/hardhat-etherscan";
import "@nomiclabs/hardhat-waffle";
import "@typechain/hardhat";
import "@primitivefi/hardhat-dodoc";
import "hardhat-gas-reporter";
import "solidity-coverage";
import { readFileSync } from "fs";
import { TASK_COMPILE_SOLIDITY_GET_SOURCE_PATHS } from "hardhat/builtin-tasks/task-names";

dotenv.config();

// default values here to avoid failures when running hardhat
const RINKEBY_RPC = process.env.RINKEBY_RPC || "1".repeat(32);
const SCROLL_L1_RPC = process.env.SCROLL_L1_RPC || "1".repeat(32);
const SCROLL_L2_RPC = process.env.SCROLL_L2_RPC || "1".repeat(32);

const RINKEBY_PRIVATE_KEY = process.env.RINKEBY_PRIVATE_KEY || "1".repeat(64);
const L1_DEPLOYER_PRIVATE_KEY = process.env.L1_DEPLOYER_PRIVATE_KEY || "1".repeat(64);
const L2_DEPLOYER_PRIVATE_KEY = process.env.L2_DEPLOYER_PRIVATE_KEY || "1".repeat(64);

const SOLC_DEFAULT = "0.8.16";

// try use forge config
let foundry: any;
try {
  foundry = toml.parse(readFileSync("./foundry.toml").toString());
  foundry.default.solc = foundry.default["solc-version"] ? foundry.default["solc-version"] : SOLC_DEFAULT;
} catch (error) {
  foundry = {
    default: {
      solc: SOLC_DEFAULT,
    },
  };
}

// prune forge style tests from hardhat paths
subtask(TASK_COMPILE_SOLIDITY_GET_SOURCE_PATHS).setAction(async (_, __, runSuper) => {
  const paths = await runSuper();
  return paths.filter((p: string) => !p.endsWith(".t.sol")).filter((p: string) => !p.includes("test/mocks"));
});

const config: HardhatUserConfig = {
  solidity: {
    version: foundry.default?.solc || SOLC_DEFAULT,
    settings: {
      optimizer: {
        enabled: foundry.default?.optimizer || true,
        runs: foundry.default?.optimizer_runs || 200,
      },
    },
  },
  networks: {
    rinkeby: {
      url: RINKEBY_RPC,
      accounts: [RINKEBY_PRIVATE_KEY],
    },
    l1geth: {
      url: SCROLL_L1_RPC,
      gasPrice: 20000000000,
      gasMultiplier: 1.1,
      accounts: [L1_DEPLOYER_PRIVATE_KEY],
    },
    l2geth: {
      url: SCROLL_L2_RPC,
      gasPrice: 20000000000,
      gasMultiplier: 1.1,
      accounts: [L2_DEPLOYER_PRIVATE_KEY],
    },
  },
  paths: {
    cache: "./cache-hardhat",
    sources: "./src",
    tests: "./integration-test",
  },
  gasReporter: {
    enabled: process.env.REPORT_GAS !== undefined,
    excludeContracts: ["src/test"],
    currency: "USD",
  },
  etherscan: {
    apiKey: process.env.ETHERSCAN_API_KEY,
  },
  mocha: {
    timeout: 10000000,
  },
  dodoc: {
    runOnCompile: true,
    keepFileStructure: false,
    include: [
      "ZKRollup",
      "L1ScrollMessenger",
      "L2ScrollMessenger",
      "L1GatewayRouter",
      "L2GatewayRouter",
      "L1StandardERC20Gateway",
      "L2StandardERC20Gateway",
      "L1ERC721Gateway",
      "L2ERC721Gateway",
      "L1ERC1155Gateway",
      "L2ERC1155Gateway",
      "L1WETHGateway",
      "L2WETHGateway",
      "ScrollStandardERC20Factory",
    ],
    outputDir: "docs/apis",
    exclude: [
      "IERC677Receiver",
      "IL1ScrollMessenger",
      "IL2ScrollMessenger",
      "IL1GatewayRouter",
      "IL2GatewayRouter",
      "IL1ERC721Gateway",
      "IL2ERC721Gateway",
      "IL1ERC1155Gateway",
      "IL2ERC1155Gateway",
      "IScrollStandardERC20Factory",
      "IZKRollup",
      "WrappedEther",
    ],
  },
};

export default config;
