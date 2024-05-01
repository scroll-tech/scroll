// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {console} from "forge-std/console.sol";
import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {L1Blocks} from "../L2/predeploys/L1Blocks.sol";
import {IL1Blocks} from "../L2/predeploys/IL1Blocks.sol";

contract L1BlocksTest is DSTestPlus {
    event UpdateTotalLimit(uint256 oldTotalLimit, uint256 newTotalLimit);

    uint256 private constant MIN_BASE_FEE_PER_BLOB_GAS = 1;

    uint256 private constant BLOB_BASE_FEE_UPDATE_FRACTION = 3338477;

    L1Blocks private b;

    function setUp() public {
        b = new L1Blocks();
    }

    function testSetL1BlockHeaderCancun() external {
        bytes[] memory headers = new bytes[](3);
        bytes32[] memory hashes = new bytes32[](3);
        bytes32[] memory roots = new bytes32[](3);
        uint256[] memory timestamps = new uint256[](3);
        uint256[] memory baseFees = new uint256[](3);
        uint256[] memory blobBaseFees = new uint256[](3);
        bytes32[] memory parentBeaconRoots = new bytes32[](3);
        headers[
            0
        ] = hex"f9025fa0db672c41cfd47c84ddb478ffde5a09b76964f77dceca0e62bdf719c965d73e7fa01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d4934794dae56d85ff707b3d19427f23d8b03b7b76da1006a0bd33ab68095087d81beb810b3f5d0b16050b3f798ae3978e440bab048dd78992a020cbdd2cd6113eb72dade6be7ec18fe6a7167a8a0af912a2171af909a8cda9f6a059c3691e83e0ddeafeedd07f5e30850cc6c963e85327aa1201fbe1f731ff3dbcb901000021000000000001080801008000320000000060000000000005000080c200040000100000006002440409000000010402011000890200000a201000042800440442148c0100208408004009012000200000040801644808800000600029068004012001020000002510000000020900c8122010020284000080021006000101000000401810621c0040000000001010000004800100404808000640255000002201000010002000000040c0000000000400a004000c0000884000304e00202400100402000000004204000004041008005600001000001000000003015030120012280000022020910040429204408020009000000010120000400000000400808401286d1b8401c9c380832830cd8465f1b05799d883010d0e846765746888676f312e32312e37856c696e7578a02617b147c1b3cf43a28d08d508ab2d8860c6cf40da58837676cfaaf4f9e07f62880000000000000000850e6ca77a30a06c119891018ed3a43d04197a8eb94d85f287304b8ace21b08e9f68cbd2f8de618080a0b35bb80bc5f4e3d8f19b62f6274add24dca334db242546c3024403027aaf6412";
        hashes[0] = bytes32(0xf8e2f40d98fe5862bc947c8c83d34799c50fb344d7445d020a8a946d891b62ee);
        roots[0] = bytes32(0xbd33ab68095087d81beb810b3f5d0b16050b3f798ae3978e440bab048dd78992);
        timestamps[0] = 1710338135;
        baseFees[0] = 61952457264;
        blobBaseFees[0] = _exp(MIN_BASE_FEE_PER_BLOB_GAS, 0, BLOB_BASE_FEE_UPDATE_FRACTION);
        parentBeaconRoots[0] = bytes32(0xb35bb80bc5f4e3d8f19b62f6274add24dca334db242546c3024403027aaf6412);
        headers[
            1
        ] = hex"f90255a0f8e2f40d98fe5862bc947c8c83d34799c50fb344d7445d020a8a946d891b62eea01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347948186b214a917fb4922eb984fb80cfafa30ee8810a0e4ca273952459efe6fddc9b1ed4d3fce53a85a66103985e1af9e8a120115afe4a0fb16519bfe18e4d8380778053ace3010a4357c1bdb1c9dfd8feda8078be23fc4a08a1616cfb1d886e9c3183cba4b19e16b80d4a38f4eb4f996a196c143a4092766b90100142b546af340118b1004096cd426d6a50059e2a04672400240850043c408202c141ce30299c053a076385b69443203545203ac298c2221c842c6919b34ae78a91c0755e824512b0a280be24d910cb1ed6009c05b00f658544422544c976475909fa6e2248aa30c009245f0400b085d601d7708730004240a1e7966ba45081062080997d14740708b214617d1014200261d83088349819e28662203e386700400aac239f268a160b2011148e69211078086229c050560030aa9a5049b80c8e055afd12593054a854569011302900e08780c663283770b42543409630683f46251de58a9290112401c2c0c762044211241c72150a2081f8063020882d00e02e580808401286d1c8401c9c38083eab9738465f1b0638f6c6f6b696275696c6465722e78797aa071e00db543e3e4b91eb392d81890499926275490e7f592f76f7428e2997765a6880000000000000000850cf01fc701a0804fd42384c45b31d07052e161d33a2e3d47d1b64dc72cc32d76c3905ff35d328080a0a471c7622a976313a61e01b01212dcea6acd71f351618734928dcabe4aba62fe";
        hashes[1] = bytes32(0x92d191c33229bf530e0a74913d604dc7c7f8d6dfd5d73f90e76f7a2e5d19c263);
        roots[1] = bytes32(0xe4ca273952459efe6fddc9b1ed4d3fce53a85a66103985e1af9e8a120115afe4);
        timestamps[1] = 1710338147;
        baseFees[1] = 55568221953;
        blobBaseFees[1] = _exp(MIN_BASE_FEE_PER_BLOB_GAS, 0, BLOB_BASE_FEE_UPDATE_FRACTION);
        parentBeaconRoots[1] = bytes32(0xa471c7622a976313a61e01b01212dcea6acd71f351618734928dcabe4aba62fe);
        headers[
            2
        ] = hex"f90254a092d191c33229bf530e0a74913d604dc7c7f8d6dfd5d73f90e76f7a2e5d19c263a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d493479488c6c46ebf353a52bdbab708c23d0c81daa8134aa08261a2a412930bbeade873726535a993723369615e2abad295da9a5398082292a02605fd9b61c93f7f28b1fc11c8672df88bca49d65bab63b6c1af9fe82b7f76aca0e549cd4fb39961fd40eb76924713dc631e2431681aadd617b208f23f0bc0f208b901001065c50c7d041102110c2820fc84213025021620005494934803116424000555000440cb001042650012d3008f12a100072188319d3f214801a6c048006a09110801c08818480a8c1830488c8a0600600880800d80550a91011013c4c1304182102272008224142821004c0008003a90d0220a11001b0c2d320801da062a2002240783462681b028026915290a08902008180a01010080186402a040053102002e810882041364612898008c308904c10510ac020c0851022801d6da00242558ab8c0402424890bb07000001001616000802431230820310260159061222274042d2a00005b0a2124384010058246e0400021083059c0052422a2a314c0b2c0d808401286d1d8401c9c380836d30ee8465f1b06f8b6a6574626c64722e78797aa057dc6b69f4e021c07813699c031bf61baad4f468040398aa8cf891d6da85d730880000000000000000850cfab14a38a017ce08097a888dc1ad4161a9dffc38bca83d22a0da5b358d13ec2945b59660d58302000080a0e8d226954b651474710dc4c1b32331d9a3074a3e61972c56b361c0df4c3594c7";
        hashes[2] = bytes32(0xa2917e0758c98640d868182838c93bb12f0d07b6b17efe6b62d9df42c7643791);
        roots[2] = bytes32(0x8261a2a412930bbeade873726535a993723369615e2abad295da9a5398082292);
        timestamps[2] = 1710338159;
        baseFees[2] = 55745530424;
        blobBaseFees[2] = _exp(MIN_BASE_FEE_PER_BLOB_GAS, 0, BLOB_BASE_FEE_UPDATE_FRACTION);
        parentBeaconRoots[2] = bytes32(0xe8d226954b651474710dc4c1b32331d9a3074a3e61972c56b361c0df4c3594c7);
        hevm.startPrank(address(0xffffFFFfFFffffffffffffffFfFFFfffFFFfFFfE));
        for (uint256 i = 0; i < 3; i++) {
            b.setL1BlockHeader(headers[i]);
            assertEq(b.latestBlockNumber(), i + 19426587);
            assertEq(b.latestBlockHash(), hashes[i]);
            assertEq(b.latestStateRoot(), roots[i]);
            assertEq(b.latestBlockTimestamp(), timestamps[i]);
            assertEq(b.latestBaseFee(), baseFees[i]);
            assertEq(b.latestBlobBaseFee(), blobBaseFees[i]);
            assertEq(b.latestParentBeaconRoot(), parentBeaconRoots[i]);
            for (uint256 j = 0; j < i; j++) {
                assertEq(b.getBlockHash(j + 19426587), hashes[j]);
                assertEq(b.getStateRoot(j + 19426587), roots[j]);
                assertEq(b.getBlockTimestamp(j + 19426587), timestamps[j]);
                assertEq(b.getBaseFee(j + 19426587), baseFees[j]);
                assertEq(b.getBlobBaseFee(j + 19426587), blobBaseFees[j]);
                assertEq(b.getParentBeaconRoot(j + 19426587), parentBeaconRoots[j]);
            }
        }
        hevm.stopPrank();
    }

    /// @dev Approximates factor * e ** (numerator / denominator) using Taylor expansion:
    /// based on `fake_exponential` in https://eips.ethereum.org/EIPS/eip-4844
    function _exp(
        uint256 factor,
        uint256 numerator,
        uint256 denominator
    ) private pure returns (uint256) {
        uint256 output;
        uint256 numerator_accum = factor * denominator;
        for (uint256 i = 1; numerator_accum > 0; i++) {
            output += numerator_accum;
            numerator_accum = (numerator_accum * numerator) / (denominator * i);
        }
        return output / denominator;
    }
}
