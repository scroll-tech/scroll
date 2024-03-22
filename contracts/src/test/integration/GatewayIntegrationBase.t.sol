// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {Test} from "forge-std/Test.sol";
import {Vm} from "forge-std/Vm.sol";

import {ProxyAdmin} from "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import {ITransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import {Domain} from "./Domain.t.sol";

import {IL2ScrollMessenger} from "../../L2/IL2ScrollMessenger.sol";
import {AddressAliasHelper} from "../../libraries/common/AddressAliasHelper.sol";

abstract contract GatewayIntegrationBase is Test {
    bytes32 private constant SENT_MESSAGE_TOPIC =
        keccak256("SentMessage(address,address,uint256,uint256,uint256,bytes)");

    address internal constant L1_SCROLL_MESSENGER = 0x6774Bcbd5ceCeF1336b5300fb5186a12DDD8b367;

    address internal constant L1_SCROLL_CHAIN = 0xa13BAF47339d63B743e7Da8741db5456DAc1E556;

    address internal constant L1_MESSAGE_QUEUE = 0x0d7E906BD9cAFa154b048cFa766Cc1E54E39AF9B;

    address internal constant L1_GATEWAY_ROUTER = 0xF8B1378579659D8F7EE5f3C929c2f3E332E41Fd6;

    address internal constant L2_SCROLL_MESSENGER = 0x781e90f1c8Fc4611c9b7497C3B47F99Ef6969CbC;

    address internal constant L2_MESSAGE_QUEUE = 0x5300000000000000000000000000000000000000;

    address internal constant L2_GATEWAY_ROUTER = 0x4C0926FF5252A435FD19e10ED15e5a249Ba19d79;

    Domain internal mainnet;

    Domain internal scroll;

    uint256 internal lastFromMainnetLogIndex;

    uint256 internal lastFromScrollLogIndex;

    receive() external payable {}

    // solhint-disable-next-line func-name-mixedcase
    function __GatewayIntegrationBase_setUp() internal {
        setChain("scroll", ChainData("Scroll Chain", 534352, "https://rpc.scroll.io"));
        setChain("mainnet", ChainData("Mainnet", 1, "https://rpc.ankr.com/eth"));

        mainnet = new Domain(getChain("mainnet"));
        scroll = new Domain(getChain("scroll"));
    }

    function relayFromMainnet() internal {
        scroll.selectFork();

        address malias = AddressAliasHelper.applyL1ToL2Alias(L1_SCROLL_MESSENGER);

        // Read all L1 -> L2 messages and relay them under Scroll fork
        Vm.Log[] memory allLogs = vm.getRecordedLogs();
        for (; lastFromMainnetLogIndex < allLogs.length; lastFromMainnetLogIndex++) {
            Vm.Log memory _log = allLogs[lastFromMainnetLogIndex];
            if (_log.topics[0] == SENT_MESSAGE_TOPIC && _log.emitter == address(L1_SCROLL_MESSENGER)) {
                address sender = address(uint160(uint256(_log.topics[1])));
                address target = address(uint160(uint256(_log.topics[2])));
                (uint256 value, uint256 nonce, uint256 gasLimit, bytes memory message) = abi.decode(
                    _log.data,
                    (uint256, uint256, uint256, bytes)
                );
                vm.prank(malias);
                IL2ScrollMessenger(L2_SCROLL_MESSENGER).relayMessage{gas: gasLimit}(
                    sender,
                    target,
                    value,
                    nonce,
                    message
                );
            }
        }
    }

    function relayFromScroll() internal {
        mainnet.selectFork();

        // Read all L2 -> L1 messages and relay them under Primary fork
        // Note: We bypass the L1 messenger relay here because it's easier to not have to generate valid state roots / merkle proofs
        Vm.Log[] memory allLogs = vm.getRecordedLogs();
        for (; lastFromScrollLogIndex < allLogs.length; lastFromScrollLogIndex++) {
            Vm.Log memory _log = allLogs[lastFromScrollLogIndex];
            if (_log.topics[0] == SENT_MESSAGE_TOPIC && _log.emitter == address(L2_SCROLL_MESSENGER)) {
                address sender = address(uint160(uint256(_log.topics[1])));
                address target = address(uint160(uint256(_log.topics[2])));
                (uint256 value, , , bytes memory message) = abi.decode(_log.data, (uint256, uint256, uint256, bytes));
                // Set xDomainMessageSender
                vm.store(address(L1_SCROLL_MESSENGER), bytes32(uint256(201)), bytes32(uint256(uint160(sender))));
                vm.startPrank(address(L1_SCROLL_MESSENGER));
                (bool success, bytes memory response) = target.call{value: value}(message);
                vm.stopPrank();
                vm.store(address(L1_SCROLL_MESSENGER), bytes32(uint256(201)), bytes32(uint256(1)));
                if (!success) {
                    assembly {
                        revert(add(response, 32), mload(response))
                    }
                }
            }
        }
    }

    function upgrade(
        bool isMainnet,
        address proxy,
        address implementation
    ) internal {
        address admin;
        address owner;
        if (isMainnet) {
            mainnet.selectFork();
            admin = 0xEB803eb3F501998126bf37bB823646Ed3D59d072;
            owner = 0x798576400F7D662961BA15C6b3F3d813447a26a6;
        } else {
            scroll.selectFork();
            admin = 0xA76acF000C890b0DD7AEEf57627d9899F955d026;
            owner = 0x13D24a7Ff6F5ec5ff0e9C40Fc3B8C9c01c65437B;
        }

        vm.startPrank(owner);
        ProxyAdmin(admin).upgrade(ITransparentUpgradeableProxy(proxy), implementation);
        vm.stopPrank();
    }
}
