/* eslint-disable node/no-missing-import */
/* eslint-disable node/no-unpublished-import */
import { HardhatEthersSigner } from "@nomicfoundation/hardhat-ethers/signers";
import { expect } from "chai";
import { BigNumberish, ContractTransactionResponse, MaxUint256, keccak256, toQuantity } from "ethers";
import { ethers, network } from "hardhat";

import {
  ProxyAdmin,
  L1GatewayRouter,
  L2ScrollMessenger,
  L1ScrollMessenger,
  L1MessageQueueWithGasPriceOracle,
  L2GatewayRouter,
} from "../typechain";

describe("GasOptimizationUpgrade.spec", async () => {
  const L1_ROUTER = "0xF8B1378579659D8F7EE5f3C929c2f3E332E41Fd6";
  const L2_ROUTER = "0x4C0926FF5252A435FD19e10ED15e5a249Ba19d79";
  const L1_MESSENGER = "0x6774Bcbd5ceCeF1336b5300fb5186a12DDD8b367";
  const L2_MESSENGER = "0x781e90f1c8Fc4611c9b7497C3B47F99Ef6969CbC";
  const L1_MESSAGE_QUEUE = "0x0d7E906BD9cAFa154b048cFa766Cc1E54E39AF9B";
  const L2_MESSAGE_QUEUE = "0x5300000000000000000000000000000000000000";
  const SCROLL_CHAIN = "0xa13BAF47339d63B743e7Da8741db5456DAc1E556";

  let deployer: HardhatEthersSigner;

  let proxyAdmin: ProxyAdmin;

  const mockERC20Balance = async (tokenAddress: string, balance: bigint, slot: BigNumberish) => {
    const storageSlot = keccak256(
      ethers.AbiCoder.defaultAbiCoder().encode(["address", "uint256"], [deployer.address, slot])
    );
    await ethers.provider.send("hardhat_setStorageAt", [tokenAddress, storageSlot, toQuantity(balance)]);
    const token = await ethers.getContractAt("MockERC20", tokenAddress, deployer);
    expect(await token.balanceOf(deployer.address)).to.eq(balance);
  };

  const mockETHBalance = async (balance: bigint) => {
    await network.provider.send("hardhat_setBalance", [deployer.address, toQuantity(balance)]);
    expect(await ethers.provider.getBalance(deployer.address)).to.eq(balance);
  };

  const showGasUsage = async (tx: ContractTransactionResponse, desc: string) => {
    const receipt = await tx.wait();
    console.log(`${desc}: GasUsed[${receipt!.gasUsed}]`);
  };

  context("L1 upgrade", async () => {
    let forkBlock: number;
    let router: L1GatewayRouter;
    let messenger: L1ScrollMessenger;
    let queue: L1MessageQueueWithGasPriceOracle;

    beforeEach(async () => {
      // fork network
      const provider = new ethers.JsonRpcProvider("https://rpc.ankr.com/eth");
      if (!forkBlock) {
        forkBlock = (await provider.getBlockNumber()) - 10;
      }
      await network.provider.request({
        method: "hardhat_reset",
        params: [
          {
            forking: {
              jsonRpcUrl: "https://rpc.ankr.com/eth",
              blockNumber: forkBlock,
            },
          },
        ],
      });
      await network.provider.request({
        method: "hardhat_impersonateAccount",
        params: ["0x1100000000000000000000000000000000000011"],
      });

      // mock eth balance
      deployer = await ethers.getSigner("0x1100000000000000000000000000000000000011");
      await mockETHBalance(ethers.parseEther("1000"));

      // mock owner of proxy admin
      proxyAdmin = await ethers.getContractAt("ProxyAdmin", "0xEB803eb3F501998126bf37bB823646Ed3D59d072", deployer);
      await ethers.provider.send("hardhat_setStorageAt", [
        await proxyAdmin.getAddress(),
        "0x0",
        ethers.AbiCoder.defaultAbiCoder().encode(["address"], [deployer.address]),
      ]);
      expect(await proxyAdmin.owner()).to.eq(deployer.address);

      router = await ethers.getContractAt("L1GatewayRouter", L1_ROUTER, deployer);
      messenger = await ethers.getContractAt("L1ScrollMessenger", L1_MESSENGER, deployer);
      queue = await ethers.getContractAt("L1MessageQueueWithGasPriceOracle", L1_MESSAGE_QUEUE, deployer);
    });

    const upgradeL1 = async (proxy: string, impl: string) => {
      await proxyAdmin.upgrade(proxy, impl);
      const L1ScrollMessenger = await ethers.getContractFactory("L1ScrollMessenger", deployer);
      const L1MessageQueueWithGasPriceOracle = await ethers.getContractFactory(
        "L1MessageQueueWithGasPriceOracle",
        deployer
      );
      const ScrollChain = await ethers.getContractFactory("ScrollChain", deployer);
      await proxyAdmin.upgrade(
        L1_MESSENGER,
        (await L1ScrollMessenger.deploy(L2_MESSENGER, SCROLL_CHAIN, L1_MESSAGE_QUEUE)).getAddress()
      );
      await proxyAdmin.upgrade(
        L1_MESSAGE_QUEUE,
        (
          await L1MessageQueueWithGasPriceOracle.deploy(
            L1_MESSENGER,
            SCROLL_CHAIN,
            "0x72CAcBcfDe2d1e19122F8A36a4d6676cd39d7A5d"
          )
        ).getAddress()
      );
      await queue.initializeV2();
      await proxyAdmin.upgrade(
        SCROLL_CHAIN,
        (await ScrollChain.deploy(534352, L1_MESSAGE_QUEUE, "0xA2Ab526e5C5491F10FC05A55F064BF9F7CEf32a0")).getAddress()
      );
    };

    it.skip("should succeed on L1ETHGateway", async () => {
      const L1_GATEWAY = "0x7F2b8C31F88B6006c382775eea88297Ec1e3E905";
      const L2_GATEWAY = "0x6EA73e05AdC79974B931123675ea8F78FfdacDF0";
      const L1ETHGateway = await ethers.getContractFactory("L1ETHGateway", deployer);
      const impl = await L1ETHGateway.deploy(L2_GATEWAY, L1_ROUTER, L1_MESSENGER);
      const gateway = await ethers.getContractAt("L1ETHGateway", L1_GATEWAY, deployer);
      const amountIn = ethers.parseEther("1");
      const fee = await queue.estimateCrossDomainMessageFee(1e6);

      // before upgrade
      await showGasUsage(
        await gateway["depositETH(uint256,uint256)"](amountIn, 1e6, { value: amountIn + fee }),
        "L1ETHGateway.depositETH before upgrade"
      );
      await showGasUsage(
        await router["depositETH(uint256,uint256)"](amountIn, 1e6, { value: amountIn + fee }),
        "L1GatewayRouter.depositETH before upgrade"
      );
      await showGasUsage(
        await messenger["sendMessage(address,uint256,bytes,uint256)"](deployer.address, amountIn, "0x", 1e6, {
          value: amountIn + fee,
        }),
        "L1ScrollMessenger.sendMessage before upgrade"
      );

      // do upgrade
      await upgradeL1(L1_GATEWAY, await impl.getAddress());

      // after upgrade
      await showGasUsage(
        await gateway["depositETH(uint256,uint256)"](amountIn, 1e6, { value: amountIn + fee }),
        "L1ETHGateway.depositETH after upgrade"
      );
      await showGasUsage(
        await router["depositETH(uint256,uint256)"](amountIn, 1e6, { value: amountIn + fee }),
        "L1GatewayRouter.depositETH after upgrade"
      );
      await showGasUsage(
        await messenger["sendMessage(address,uint256,bytes,uint256)"](deployer.address, amountIn, "0x", 1e6, {
          value: amountIn + fee,
        }),
        "L1ScrollMessenger.sendMessage after upgrade"
      );
    });

    it.skip("should succeed on L1WETHGateway", async () => {
      const L1_WETH = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2";
      const L2_WETH = "0x5300000000000000000000000000000000000004";
      const L1_GATEWAY = "0x7AC440cAe8EB6328de4fA621163a792c1EA9D4fE";
      const L2_GATEWAY = "0x7003E7B7186f0E6601203b99F7B8DECBfA391cf9";
      const L1WETHGateway = await ethers.getContractFactory("L1WETHGateway", deployer);
      const impl = await L1WETHGateway.deploy(L1_WETH, L2_WETH, L2_GATEWAY, L1_ROUTER, L1_MESSENGER);
      const gateway = await ethers.getContractAt("L1WETHGateway", L1_GATEWAY, deployer);
      const amountIn = ethers.parseEther("1");
      const fee = await queue.estimateCrossDomainMessageFee(1e6);
      const token = await ethers.getContractAt("MockERC20", L1_WETH, deployer);
      await mockERC20Balance(await token.getAddress(), amountIn * 10n, 3);
      await token.approve(L1_GATEWAY, MaxUint256);
      await token.approve(L1_ROUTER, MaxUint256);

      // before upgrade
      await showGasUsage(
        await gateway["depositERC20(address,uint256,uint256)"](L1_WETH, amountIn, 1e6, { value: fee }),
        "L1WETHGateway.depositERC20 WETH before upgrade"
      );
      await showGasUsage(
        await router["depositERC20(address,uint256,uint256)"](L1_WETH, amountIn, 1e6, { value: fee }),
        "L1GatewayRouter.depositERC20 WETH before upgrade"
      );

      // do upgrade
      await upgradeL1(L1_GATEWAY, await impl.getAddress());

      // after upgrade
      await showGasUsage(
        await gateway["depositERC20(address,uint256,uint256)"](L1_WETH, amountIn, 1e6, { value: fee }),
        "L1WETHGateway.depositERC20 WETH after upgrade"
      );
      await showGasUsage(
        await router["depositERC20(address,uint256,uint256)"](L1_WETH, amountIn, 1e6, { value: fee }),
        "L1GatewayRouter.depositERC20 WETH after upgrade"
      );
    });

    it.skip("should succeed on L1StandardERC20Gateway", async () => {
      const L1_USDT = "0xdAC17F958D2ee523a2206206994597C13D831ec7";
      const L1_GATEWAY = "0xD8A791fE2bE73eb6E6cF1eb0cb3F36adC9B3F8f9";
      const L2_GATEWAY = "0xE2b4795039517653c5Ae8C2A9BFdd783b48f447A";
      const L1StandardERC20Gateway = await ethers.getContractFactory("L1StandardERC20Gateway", deployer);
      const impl = await L1StandardERC20Gateway.deploy(
        L2_GATEWAY,
        L1_ROUTER,
        L1_MESSENGER,
        "0xC7d86908ccf644Db7C69437D5852CedBC1aD3f69",
        "0x66e5312EDeEAef6e80759A0F789e7914Fb401484"
      );
      const gateway = await ethers.getContractAt("L1StandardERC20Gateway", L1_GATEWAY, deployer);
      const amountIn = ethers.parseUnits("1", 6);
      const fee = await queue.estimateCrossDomainMessageFee(1e6);
      const token = await ethers.getContractAt("MockERC20", L1_USDT, deployer);
      await mockERC20Balance(await token.getAddress(), amountIn * 10n, 2);
      await token.approve(L1_GATEWAY, MaxUint256);
      await token.approve(L1_ROUTER, MaxUint256);

      // before upgrade
      await showGasUsage(
        await gateway["depositERC20(address,uint256,uint256)"](L1_USDT, amountIn, 1e6, { value: fee }),
        "L1StandardERC20Gateway.depositERC20 USDT before upgrade"
      );
      await showGasUsage(
        await router["depositERC20(address,uint256,uint256)"](L1_USDT, amountIn, 1e6, { value: fee }),
        "L1GatewayRouter.depositERC20 USDT before upgrade"
      );

      // do upgrade
      await upgradeL1(L1_GATEWAY, await impl.getAddress());

      // after upgrade
      await showGasUsage(
        await gateway["depositERC20(address,uint256,uint256)"](L1_USDT, amountIn, 1e6, { value: fee }),
        "L1StandardERC20Gateway.depositERC20 USDT after upgrade"
      );
      await showGasUsage(
        await router["depositERC20(address,uint256,uint256)"](L1_USDT, amountIn, 1e6, { value: fee }),
        "L1GatewayRouter.depositERC20 USDT after upgrade"
      );
    });

    it.skip("should succeed on L1CustomERC20Gateway", async () => {
      const L1_DAI = "0x6B175474E89094C44Da98b954EedeAC495271d0F";
      const L1_GATEWAY = "0x67260A8B73C5B77B55c1805218A42A7A6F98F515";
      const L2_GATEWAY = "0xaC78dff3A87b5b534e366A93E785a0ce8fA6Cc62";
      const L1CustomERC20Gateway = await ethers.getContractFactory("L1CustomERC20Gateway", deployer);
      const impl = await L1CustomERC20Gateway.deploy(L2_GATEWAY, L1_ROUTER, L1_MESSENGER);
      const gateway = await ethers.getContractAt("L1CustomERC20Gateway", L1_GATEWAY, deployer);
      const amountIn = ethers.parseUnits("1", 18);
      const fee = await queue.estimateCrossDomainMessageFee(1e6);
      const token = await ethers.getContractAt("MockERC20", L1_DAI, deployer);
      await mockERC20Balance(await token.getAddress(), amountIn * 10n, 2);
      await token.approve(L1_GATEWAY, MaxUint256);
      await token.approve(L1_ROUTER, MaxUint256);

      // before upgrade
      await showGasUsage(
        await gateway["depositERC20(address,uint256,uint256)"](L1_DAI, amountIn, 1e6, { value: fee }),
        "L1CustomERC20Gateway.depositERC20 DAI before upgrade"
      );
      await showGasUsage(
        await router["depositERC20(address,uint256,uint256)"](L1_DAI, amountIn, 1e6, { value: fee }),
        "L1GatewayRouter.depositERC20 DAI before upgrade"
      );

      // do upgrade
      await upgradeL1(L1_GATEWAY, await impl.getAddress());

      // after upgrade
      await showGasUsage(
        await gateway["depositERC20(address,uint256,uint256)"](L1_DAI, amountIn, 1e6, { value: fee }),
        "L1CustomERC20Gateway.depositERC20 DAI after upgrade"
      );
      await showGasUsage(
        await router["depositERC20(address,uint256,uint256)"](L1_DAI, amountIn, 1e6, { value: fee }),
        "L1GatewayRouter.depositERC20 DAI after upgrade"
      );
    });

    it.skip("should succeed on L1USDCGateway", async () => {
      const L1_USDC = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48";
      const L2_USDC = "0x06eFdBFf2a14a7c8E15944D1F4A48F9F95F663A4";
      const L1_GATEWAY = "0xf1AF3b23DE0A5Ca3CAb7261cb0061C0D779A5c7B";
      const L2_GATEWAY = "0x33B60d5Dd260d453cAC3782b0bDC01ce84672142";
      const L1USDCGateway = await ethers.getContractFactory("L1USDCGateway", deployer);
      const impl = await L1USDCGateway.deploy(L1_USDC, L2_USDC, L2_GATEWAY, L1_ROUTER, L1_MESSENGER);
      const gateway = await ethers.getContractAt("L1USDCGateway", L1_GATEWAY, deployer);
      const amountIn = ethers.parseUnits("1", 6);
      const fee = await queue.estimateCrossDomainMessageFee(1e6);
      const token = await ethers.getContractAt("MockERC20", L1_USDC, deployer);
      await mockERC20Balance(await token.getAddress(), amountIn * 10n, 9);
      await token.approve(L1_GATEWAY, MaxUint256);
      await token.approve(L1_ROUTER, MaxUint256);

      // before upgrade
      await showGasUsage(
        await gateway["depositERC20(address,uint256,uint256)"](L1_USDC, amountIn, 1e6, { value: fee }),
        "L1USDCGateway.depositERC20 USDC before upgrade"
      );
      await showGasUsage(
        await router["depositERC20(address,uint256,uint256)"](L1_USDC, amountIn, 1e6, { value: fee }),
        "L1GatewayRouter.depositERC20 USDC before upgrade"
      );

      // do upgrade
      await upgradeL1(L1_GATEWAY, await impl.getAddress());

      // after upgrade
      await showGasUsage(
        await gateway["depositERC20(address,uint256,uint256)"](L1_USDC, amountIn, 1e6, { value: fee }),
        "L1USDCGateway.depositERC20 USDC after upgrade"
      );
      await showGasUsage(
        await router["depositERC20(address,uint256,uint256)"](L1_USDC, amountIn, 1e6, { value: fee }),
        "L1GatewayRouter.depositERC20 USDC after upgrade"
      );
    });

    it.skip("should succeed on L1LidoGateway", async () => {
      const L1_WSTETH = "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0";
      const L2_WSTETH = "0xf610A9dfB7C89644979b4A0f27063E9e7d7Cda32";
      const L1_GATEWAY = "0x6625C6332c9F91F2D27c304E729B86db87A3f504";
      const L2_GATEWAY = "0x8aE8f22226B9d789A36AC81474e633f8bE2856c9";
      const L1LidoGateway = await ethers.getContractFactory("L1LidoGateway", deployer);
      const impl = await L1LidoGateway.deploy(L1_WSTETH, L2_WSTETH, L2_GATEWAY, L1_ROUTER, L1_MESSENGER);
      const gateway = await ethers.getContractAt("L1LidoGateway", L1_GATEWAY, deployer);
      const amountIn = ethers.parseUnits("1", 6);
      const fee = await queue.estimateCrossDomainMessageFee(1e6);
      const token = await ethers.getContractAt("MockERC20", L1_WSTETH, deployer);
      await mockERC20Balance(await token.getAddress(), amountIn * 10n, 0);
      await token.approve(L1_GATEWAY, MaxUint256);
      await token.approve(L1_ROUTER, MaxUint256);

      // before upgrade
      await showGasUsage(
        await gateway["depositERC20(address,uint256,uint256)"](L1_WSTETH, amountIn, 1e6, { value: fee }),
        "L1LidoGateway.depositERC20 wstETH before upgrade"
      );
      await showGasUsage(
        await router["depositERC20(address,uint256,uint256)"](L1_WSTETH, amountIn, 1e6, { value: fee }),
        "L1GatewayRouter.depositERC20 wstETH before upgrade"
      );

      // do upgrade
      await upgradeL1(L1_GATEWAY, await impl.getAddress());
      await gateway.initializeV2(deployer.address, deployer.address, deployer.address, deployer.address);

      // after upgrade
      await showGasUsage(
        await gateway["depositERC20(address,uint256,uint256)"](L1_WSTETH, amountIn, 1e6, { value: fee }),
        "L1LidoGateway.depositERC20 wstETH after upgrade"
      );
      await showGasUsage(
        await router["depositERC20(address,uint256,uint256)"](L1_WSTETH, amountIn, 1e6, { value: fee }),
        "L1GatewayRouter.depositERC20 wstETH after upgrade"
      );
    });
  });

  context("L2 upgrade", async () => {
    let forkBlock: number;
    let router: L2GatewayRouter;
    let messenger: L2ScrollMessenger;

    beforeEach(async () => {
      // fork network
      const provider = new ethers.JsonRpcProvider("https://rpc.scroll.io");
      if (!forkBlock) {
        forkBlock = (await provider.getBlockNumber()) - 31;
      }
      await network.provider.request({
        method: "hardhat_reset",
        params: [
          {
            forking: {
              jsonRpcUrl: "https://rpc.scroll.io",
              blockNumber: forkBlock,
            },
          },
        ],
      });
      await network.provider.request({
        method: "hardhat_impersonateAccount",
        params: ["0x1100000000000000000000000000000000000011"],
      });

      // mock eth balance
      deployer = await ethers.getSigner("0x1100000000000000000000000000000000000011");
      await mockETHBalance(ethers.parseEther("1000"));

      // mock owner of proxy admin
      proxyAdmin = await ethers.getContractAt("ProxyAdmin", "0xA76acF000C890b0DD7AEEf57627d9899F955d026", deployer);
      await ethers.provider.send("hardhat_setStorageAt", [
        await proxyAdmin.getAddress(),
        "0x0",
        ethers.AbiCoder.defaultAbiCoder().encode(["address"], [deployer.address]),
      ]);
      expect(await proxyAdmin.owner()).to.eq(deployer.address);

      router = await ethers.getContractAt("L2GatewayRouter", L2_ROUTER, deployer);
      messenger = await ethers.getContractAt("L2ScrollMessenger", L2_MESSENGER, deployer);
    });

    const upgradeL2 = async (proxy: string, impl: string) => {
      await proxyAdmin.upgrade(proxy, impl);
      const L2ScrollMessenger = await ethers.getContractFactory("L2ScrollMessenger", deployer);
      await proxyAdmin.upgrade(
        L2_MESSENGER,
        (await L2ScrollMessenger.deploy(L1_MESSENGER, L2_MESSAGE_QUEUE)).getAddress()
      );
    };

    it.skip("should succeed on L2ETHGateway", async () => {
      const L1_GATEWAY = "0x7F2b8C31F88B6006c382775eea88297Ec1e3E905";
      const L2_GATEWAY = "0x6EA73e05AdC79974B931123675ea8F78FfdacDF0";
      const L2ETHGateway = await ethers.getContractFactory("L2ETHGateway", deployer);
      const impl = await L2ETHGateway.deploy(L1_GATEWAY, L2_ROUTER, L2_MESSENGER);
      const gateway = await ethers.getContractAt("L2ETHGateway", L2_GATEWAY, deployer);
      const amountIn = ethers.parseEther("1");

      // before upgrade
      await showGasUsage(
        await gateway["withdrawETH(uint256,uint256)"](amountIn, 1e6, { value: amountIn }),
        "L2ETHGateway.withdrawETH before upgrade"
      );
      await showGasUsage(
        await router["withdrawETH(uint256,uint256)"](amountIn, 1e6, { value: amountIn }),
        "L2GatewayRouter.withdrawETH before upgrade"
      );
      await showGasUsage(
        await messenger["sendMessage(address,uint256,bytes,uint256)"](deployer.address, amountIn, "0x", 1e6, {
          value: amountIn,
        }),
        "L2ScrollMessenger.sendMessage before upgrade"
      );

      // do upgrade
      await upgradeL2(L2_GATEWAY, await impl.getAddress());

      // after upgrade
      await showGasUsage(
        await gateway["withdrawETH(uint256,uint256)"](amountIn, 1e6, { value: amountIn }),
        "L2ETHGateway.withdrawETH after upgrade"
      );
      await showGasUsage(
        await router["withdrawETH(uint256,uint256)"](amountIn, 1e6, { value: amountIn }),
        "L2GatewayRouter.withdrawETH after upgrade"
      );
      await showGasUsage(
        await messenger["sendMessage(address,uint256,bytes,uint256)"](deployer.address, amountIn, "0x", 1e6, {
          value: amountIn,
        }),
        "L2ScrollMessenger.sendMessage after upgrade"
      );
    });

    it.skip("should succeed on L2WETHGateway", async () => {
      const L1_WETH = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2";
      const L2_WETH = "0x5300000000000000000000000000000000000004";
      const L1_GATEWAY = "0x7AC440cAe8EB6328de4fA621163a792c1EA9D4fE";
      const L2_GATEWAY = "0x7003E7B7186f0E6601203b99F7B8DECBfA391cf9";
      const L2WETHGateway = await ethers.getContractFactory("L2WETHGateway", deployer);
      const impl = await L2WETHGateway.deploy(L2_WETH, L1_WETH, L1_GATEWAY, L2_ROUTER, L2_MESSENGER);
      const gateway = await ethers.getContractAt("L2WETHGateway", L2_GATEWAY, deployer);
      const amountIn = ethers.parseEther("1");
      const token = await ethers.getContractAt("MockERC20", L2_WETH, deployer);
      await mockERC20Balance(await token.getAddress(), amountIn * 10n, 0);
      await token.approve(L2_GATEWAY, MaxUint256);
      await token.approve(L2_ROUTER, MaxUint256);

      // before upgrade
      await showGasUsage(
        await gateway["withdrawERC20(address,uint256,uint256)"](L2_WETH, amountIn, 1e6),
        "L2WETHGateway.withdrawERC20 WETH before upgrade"
      );
      await showGasUsage(
        await router["withdrawERC20(address,uint256,uint256)"](L2_WETH, amountIn, 1e6),
        "L2GatewayRouter.withdrawERC20 WETH before upgrade"
      );

      // do upgrade
      await upgradeL2(L2_GATEWAY, await impl.getAddress());

      // after upgrade
      await showGasUsage(
        await gateway["withdrawERC20(address,uint256,uint256)"](L2_WETH, amountIn, 1e6),
        "L2WETHGateway.withdrawERC20 WETH after upgrade"
      );
      await showGasUsage(
        await router["withdrawERC20(address,uint256,uint256)"](L2_WETH, amountIn, 1e6),
        "L2GatewayRouter.withdrawERC20 WETH after upgrade"
      );
    });

    it.skip("should succeed on L2StandardERC20Gateway", async () => {
      const L2_USDT = "0xf55BEC9cafDbE8730f096Aa55dad6D22d44099Df";
      const L1_GATEWAY = "0xD8A791fE2bE73eb6E6cF1eb0cb3F36adC9B3F8f9";
      const L2_GATEWAY = "0xE2b4795039517653c5Ae8C2A9BFdd783b48f447A";
      const L2StandardERC20Gateway = await ethers.getContractFactory("L2StandardERC20Gateway", deployer);
      const impl = await L2StandardERC20Gateway.deploy(
        L1_GATEWAY,
        L2_ROUTER,
        L2_MESSENGER,
        "0x66e5312EDeEAef6e80759A0F789e7914Fb401484"
      );
      const gateway = await ethers.getContractAt("L2StandardERC20Gateway", L2_GATEWAY, deployer);
      const amountIn = ethers.parseUnits("1", 6);
      const token = await ethers.getContractAt("MockERC20", L2_USDT, deployer);
      await mockERC20Balance(await token.getAddress(), amountIn * 10n, 51);
      await token.approve(L2_GATEWAY, MaxUint256);
      await token.approve(L2_ROUTER, MaxUint256);

      // before upgrade
      await showGasUsage(
        await gateway["withdrawERC20(address,uint256,uint256)"](L2_USDT, amountIn, 1e6),
        "L2StandardERC20Gateway.withdrawERC20 USDT before upgrade"
      );
      await showGasUsage(
        await router["withdrawERC20(address,uint256,uint256)"](L2_USDT, amountIn, 1e6),
        "L2GatewayRouter.withdrawERC20 USDT before upgrade"
      );

      // do upgrade
      await upgradeL2(L2_GATEWAY, await impl.getAddress());

      // after upgrade
      await showGasUsage(
        await gateway["withdrawERC20(address,uint256,uint256)"](L2_USDT, amountIn, 1e6),
        "L2StandardERC20Gateway.withdrawERC20 USDT after upgrade"
      );
      await showGasUsage(
        await router["withdrawERC20(address,uint256,uint256)"](L2_USDT, amountIn, 1e6),
        "L2GatewayRouter.withdrawERC20 USDT after upgrade"
      );
    });

    it.skip("should succeed on L2CustomERC20Gateway", async () => {
      const L2_DAI = "0xcA77eB3fEFe3725Dc33bccB54eDEFc3D9f764f97";
      const L1_GATEWAY = "0x67260A8B73C5B77B55c1805218A42A7A6F98F515";
      const L2_GATEWAY = "0xaC78dff3A87b5b534e366A93E785a0ce8fA6Cc62";
      const L2CustomERC20Gateway = await ethers.getContractFactory("L2CustomERC20Gateway", deployer);
      const impl = await L2CustomERC20Gateway.deploy(L1_GATEWAY, L2_ROUTER, L2_MESSENGER);
      const gateway = await ethers.getContractAt("L2CustomERC20Gateway", L2_GATEWAY, deployer);
      const amountIn = ethers.parseUnits("1", 18);
      const token = await ethers.getContractAt("MockERC20", L2_DAI, deployer);
      await mockERC20Balance(await token.getAddress(), amountIn * 10n, 51);
      await token.approve(L1_GATEWAY, MaxUint256);
      await token.approve(L1_ROUTER, MaxUint256);

      // before upgrade
      await showGasUsage(
        await gateway["withdrawERC20(address,uint256,uint256)"](L2_DAI, amountIn, 1e6),
        "L2CustomERC20Gateway.withdrawERC20 DAI before upgrade"
      );
      await showGasUsage(
        await router["withdrawERC20(address,uint256,uint256)"](L2_DAI, amountIn, 1e6),
        "L2GatewayRouter.withdrawERC20 DAI before upgrade"
      );

      // do upgrade
      await upgradeL2(L2_GATEWAY, await impl.getAddress());

      // after upgrade
      await showGasUsage(
        await gateway["withdrawERC20(address,uint256,uint256)"](L2_DAI, amountIn, 1e6),
        "L2CustomERC20Gateway.withdrawERC20 DAI after upgrade"
      );
      await showGasUsage(
        await router["withdrawERC20(address,uint256,uint256)"](L2_DAI, amountIn, 1e6),
        "L2GatewayRouter.withdrawERC20 DAI after upgrade"
      );
    });

    it.skip("should succeed on L2USDCGateway", async () => {
      const L1_USDC = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48";
      const L2_USDC = "0x06eFdBFf2a14a7c8E15944D1F4A48F9F95F663A4";
      const L1_GATEWAY = "0xf1AF3b23DE0A5Ca3CAb7261cb0061C0D779A5c7B";
      const L2_GATEWAY = "0x33B60d5Dd260d453cAC3782b0bDC01ce84672142";
      const L2USDCGateway = await ethers.getContractFactory("L2USDCGateway", deployer);
      const impl = await L2USDCGateway.deploy(L1_USDC, L2_USDC, L1_GATEWAY, L2_ROUTER, L2_MESSENGER);
      const gateway = await ethers.getContractAt("L2USDCGateway", L2_GATEWAY, deployer);
      const amountIn = ethers.parseUnits("1", 6);
      const token = await ethers.getContractAt("MockERC20", L2_USDC, deployer);
      await mockERC20Balance(await token.getAddress(), amountIn * 10n, 9);
      await token.approve(L2_GATEWAY, MaxUint256);
      await token.approve(L2_ROUTER, MaxUint256);

      // before upgrade
      await showGasUsage(
        await gateway["withdrawERC20(address,uint256,uint256)"](L2_USDC, amountIn, 1e6),
        "L2USDCGateway.withdrawERC20 USDC before upgrade"
      );
      await showGasUsage(
        await router["withdrawERC20(address,uint256,uint256)"](L2_USDC, amountIn, 1e6),
        "L2GatewayRouter.withdrawERC20 USDC before upgrade"
      );

      // do upgrade
      await upgradeL2(L2_GATEWAY, await impl.getAddress());

      // after upgrade
      await showGasUsage(
        await gateway["withdrawERC20(address,uint256,uint256)"](L2_USDC, amountIn, 1e6),
        "L2USDCGateway.withdrawERC20 USDC after upgrade"
      );
      await showGasUsage(
        await router["withdrawERC20(address,uint256,uint256)"](L2_USDC, amountIn, 1e6),
        "L2GatewayRouter.withdrawERC20 USDC after upgrade"
      );
    });

    it.skip("should succeed on L2LidoGateway", async () => {
      const L1_WSTETH = "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0";
      const L2_WSTETH = "0xf610A9dfB7C89644979b4A0f27063E9e7d7Cda32";
      const L1_GATEWAY = "0x6625C6332c9F91F2D27c304E729B86db87A3f504";
      const L2_GATEWAY = "0x8aE8f22226B9d789A36AC81474e633f8bE2856c9";
      const L2LidoGateway = await ethers.getContractFactory("L2LidoGateway", deployer);
      const impl = await L2LidoGateway.deploy(L1_WSTETH, L2_WSTETH, L1_GATEWAY, L2_ROUTER, L2_MESSENGER);
      const gateway = await ethers.getContractAt("L2LidoGateway", L2_GATEWAY, deployer);
      const amountIn = ethers.parseUnits("1", 6);
      const token = await ethers.getContractAt("MockERC20", L2_WSTETH, deployer);
      await mockERC20Balance(await token.getAddress(), amountIn * 10n, 51);
      await token.approve(L2_GATEWAY, MaxUint256);
      await token.approve(L2_ROUTER, MaxUint256);

      // before upgrade
      await showGasUsage(
        await gateway["withdrawERC20(address,uint256,uint256)"](L2_WSTETH, amountIn, 1e6),
        "L2LidoGateway.withdrawERC20 wstETH before upgrade"
      );
      await showGasUsage(
        await router["withdrawERC20(address,uint256,uint256)"](L2_WSTETH, amountIn, 1e6),
        "L2GatewayRouter.withdrawERC20 wstETH before upgrade"
      );

      // do upgrade
      await upgradeL2(L2_GATEWAY, await impl.getAddress());
      await gateway.initializeV2(deployer.address, deployer.address, deployer.address, deployer.address);

      // after upgrade
      await showGasUsage(
        await gateway["withdrawERC20(address,uint256,uint256)"](L2_WSTETH, amountIn, 1e6),
        "L2LidoGateway.withdrawERC20 wstETH after upgrade"
      );
      await showGasUsage(
        await router["withdrawERC20(address,uint256,uint256)"](L2_WSTETH, amountIn, 1e6),
        "L2GatewayRouter.withdrawERC20 wstETH after upgrade"
      );
    });
  });
});
