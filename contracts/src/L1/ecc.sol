// SPDX-License-Identifier: GPL-3.0

pragma solidity >=0.7.0 <0.9.0;

contract Ecc {
    /* ECC Functions */
    // https://etherscan.io/address/0x41bf00f080ed41fa86201eac56b8afb170d9e36d#code
    function ecAdd(uint256[2] memory p0, uint256[2] memory p1) public view
        returns (uint256[2] memory retP)
    {
        uint256[4] memory i = [p0[0], p0[1], p1[0], p1[1]];

        assembly {
            // call ecadd precompile
            // inputs are: x1, y1, x2, y2
            if iszero(staticcall(not(0), 0x06, i, 0x80, retP, 0x40)) {
                revert(0, 0)
            }
        }
    }

    // https://etherscan.io/address/0x41bf00f080ed41fa86201eac56b8afb170d9e36d#code
    function ecMul(uint256[2] memory p, uint256 s) public view
        returns (uint256[2] memory retP)
    {
        // With a public key (x, y), this computes p = scalar * (x, y).
        uint256[3] memory i = [p[0], p[1], s];

        assembly {
            // call ecmul precompile
            // inputs are: x, y, scalar
            if iszero(staticcall(not(0), 0x07, i, 0x60, retP, 0x40)) {
                revert(0, 0)
            }
        }
    }

    // scroll-tech/scroll/contracts/src/libraries/verifier/RollupVerifier.sol
    struct G1Point {
        uint256 x;
        uint256 y;
    }
    struct G2Point {
        uint256[2] x;
        uint256[2] y;
    }
    function ecPairing(G1Point[] memory p1, G2Point[] memory p2) internal view returns (bool) {
        uint256 length = p1.length * 6;
        uint256[] memory input = new uint256[](length);
        uint256[1] memory result;
        bool ret;

        require(p1.length == p2.length);

        for (uint256 i = 0; i < p1.length; i++) {
            input[0 + i * 6] = p1[i].x;
            input[1 + i * 6] = p1[i].y;
            input[2 + i * 6] = p2[i].x[0];
            input[3 + i * 6] = p2[i].x[1];
            input[4 + i * 6] = p2[i].y[0];
            input[5 + i * 6] = p2[i].y[1];
        }

        assembly {
            ret := staticcall(gas(), 8, add(input, 0x20), mul(length, 0x20), result, 0x20)
        }
        require(ret);
        return result[0] != 0;
    }

    /* Bench */
    function ecAdds(uint256 n) public
    {
        uint256[2] memory p0;
        p0[0] = 1;
        p0[1] = 2;
        uint256[2] memory p1;
        p1[0] = 1;
        p1[1] = 2;

        for (uint i = 0; i < n; i++) {
            ecAdd(p0, p1);
        }
    }

    function ecMuls(uint256 n) public
    {
        uint256[2] memory p0;
        p0[0] = 1;
        p0[1] = 2;

        for (uint i = 0; i < n; i++) {
            ecMul(p0, 3);
        }
    }

    function ecPairings(uint256 n) public
    {
        G1Point[] memory g1_points = new G1Point[](2);
        G2Point[] memory g2_points = new G2Point[](2);
        g1_points[0].x = 0x0000000000000000000000000000000000000000000000000000000000000001;
        g1_points[0].y = 0x0000000000000000000000000000000000000000000000000000000000000002;
        g2_points[0].x[1] = 0x1800deef121f1e76426a00665e5c4479674322d4f75edadd46debd5cd992f6ed;
        g2_points[0].x[0] = 0x198e9393920d483a7260bfb731fb5d25f1aa493335a9e71297e485b7aef312c2;
        g2_points[0].y[1] = 0x12c85ea5db8c6deb4aab71808dcb408fe3d1e7690c43d37b4ce6cc0166fa7daa;
        g2_points[0].y[0] = 0x090689d0585ff075ec9e99ad690c3395bc4b313370b38ef355acdadcd122975b;
        g1_points[1].x = 0x1aa125a22bd902874034e67868aed40267e5575d5919677987e3bc6dd42a32fe;
        g1_points[1].y = 0x1bacc186725464068956d9a191455c2d6f6db282d83645c610510d8d4efbaee0;
        g2_points[1].x[1] = 0x1b7734c80605f71f1e2de61e998ce5854ff2abebb76537c3d67e50d71422a852;
        g2_points[1].x[0] = 0x10d5a1e34b2388a5ebe266033a5e0e63c89084203784da0c6bd9b052a78a2cac;
        g2_points[1].y[1] = 0x275739c5c2cdbc72e37c689e2ab441ea76c1d284b9c46ae8f5c42ead937819e1;
        g2_points[1].y[0] = 0x018de34c5b7c3d3d75428bbe050f1449ea3d9961d563291f307a1874f7332e65;

        for (uint i = 0; i < n; i++) {
            ecPairing(g1_points, g2_points);
            // bool checked = false;
            // checked = ecPairing(g1_points, g2_points);
            // require(checked);
        }
    }

    // https://github.com/OpenZeppelin/openzeppelin-contracts/blob/8a0b7bed82d6b8053872c3fd40703efd58f5699d/test/utils/cryptography/ECDSA.test.js#L230
    function ecRecovers(uint256 n) public
    {
        bytes32 hash = 0xb94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9;
        bytes32 r = 0xe742ff452d41413616a5bf43fe15dd88294e983d3d36206c2712f39083d638bd;
        uint8 v = 0x1b;
        bytes32 s = 0xe0a0fc89be718fbc1033e1d30d78be1c68081562ed2e97af876f286f3453231d;

        for (uint i = 0; i < n; i++) {
            ecrecover(hash, v, r, s);
        }
    }
}
