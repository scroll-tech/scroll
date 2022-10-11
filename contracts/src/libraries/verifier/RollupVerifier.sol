// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.4.16 <0.9.0;

library RollupVerifier {
  function pairing(G1Point[] memory p1, G2Point[] memory p2) internal view returns (bool) {
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

  uint256 constant q_mod = 21888242871839275222246405745257275088548364400416034343698204186575808495617;

  function fr_invert(uint256 a) internal view returns (uint256) {
    return fr_pow(a, q_mod - 2);
  }

  function fr_pow(uint256 a, uint256 power) internal view returns (uint256) {
    uint256[6] memory input;
    uint256[1] memory result;
    bool ret;

    input[0] = 32;
    input[1] = 32;
    input[2] = 32;
    input[3] = a;
    input[4] = power;
    input[5] = q_mod;

    assembly {
      ret := staticcall(gas(), 0x05, input, 0xc0, result, 0x20)
    }
    require(ret);

    return result[0];
  }

  function fr_div(uint256 a, uint256 b) internal view returns (uint256) {
    require(b != 0);
    return mulmod(a, fr_invert(b), q_mod);
  }

  function fr_mul_add(
    uint256 a,
    uint256 b,
    uint256 c
  ) internal pure returns (uint256) {
    return addmod(mulmod(a, b, q_mod), c, q_mod);
  }

  function fr_mul_add_pm(
    uint256[78] memory m,
    uint256[] calldata proof,
    uint256 opcode,
    uint256 t
  ) internal pure returns (uint256) {
    for (uint256 i = 0; i < 32; i += 2) {
      uint256 a = opcode & 0xff;
      if (a != 0xff) {
        opcode >>= 8;
        uint256 b = opcode & 0xff;
        opcode >>= 8;
        t = addmod(mulmod(proof[a], m[b], q_mod), t, q_mod);
      } else {
        break;
      }
    }

    return t;
  }

  function fr_mul_add_mt(
    uint256[78] memory m,
    uint256 base,
    uint256 opcode,
    uint256 t
  ) internal pure returns (uint256) {
    for (uint256 i = 0; i < 32; i += 1) {
      uint256 a = opcode & 0xff;
      if (a != 0xff) {
        opcode >>= 8;
        t = addmod(mulmod(base, t, q_mod), m[a], q_mod);
      } else {
        break;
      }
    }

    return t;
  }

  function fr_reverse(uint256 input) internal pure returns (uint256 v) {
    v = input;

    // swap bytes
    v =
      ((v & 0xFF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00) >> 8) |
      ((v & 0x00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF) << 8);

    // swap 2-byte long pairs
    v =
      ((v & 0xFFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000) >> 16) |
      ((v & 0x0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF) << 16);

    // swap 4-byte long pairs
    v =
      ((v & 0xFFFFFFFF00000000FFFFFFFF00000000FFFFFFFF00000000FFFFFFFF00000000) >> 32) |
      ((v & 0x00000000FFFFFFFF00000000FFFFFFFF00000000FFFFFFFF00000000FFFFFFFF) << 32);

    // swap 8-byte long pairs
    v =
      ((v & 0xFFFFFFFFFFFFFFFF0000000000000000FFFFFFFFFFFFFFFF0000000000000000) >> 64) |
      ((v & 0x0000000000000000FFFFFFFFFFFFFFFF0000000000000000FFFFFFFFFFFFFFFF) << 64);

    // swap 16-byte long pairs
    v = (v >> 128) | (v << 128);
  }

  uint256 constant p_mod = 21888242871839275222246405745257275088696311157297823662689037894645226208583;

  struct G1Point {
    uint256 x;
    uint256 y;
  }

  struct G2Point {
    uint256[2] x;
    uint256[2] y;
  }

  function ecc_from(uint256 x, uint256 y) internal pure returns (G1Point memory r) {
    r.x = x;
    r.y = y;
  }

  function ecc_add(
    uint256 ax,
    uint256 ay,
    uint256 bx,
    uint256 by
  ) internal view returns (uint256, uint256) {
    bool ret = false;
    G1Point memory r;
    uint256[4] memory input_points;

    input_points[0] = ax;
    input_points[1] = ay;
    input_points[2] = bx;
    input_points[3] = by;

    assembly {
      ret := staticcall(gas(), 6, input_points, 0x80, r, 0x40)
    }
    require(ret);

    return (r.x, r.y);
  }

  function ecc_sub(
    uint256 ax,
    uint256 ay,
    uint256 bx,
    uint256 by
  ) internal view returns (uint256, uint256) {
    return ecc_add(ax, ay, bx, p_mod - by);
  }

  function ecc_mul(
    uint256 px,
    uint256 py,
    uint256 s
  ) internal view returns (uint256, uint256) {
    uint256[3] memory input;
    bool ret = false;
    G1Point memory r;

    input[0] = px;
    input[1] = py;
    input[2] = s;

    assembly {
      ret := staticcall(gas(), 7, input, 0x60, r, 0x40)
    }
    require(ret);

    return (r.x, r.y);
  }

  function _ecc_mul_add(uint256[5] memory input) internal view {
    bool ret = false;

    assembly {
      ret := staticcall(gas(), 7, input, 0x60, add(input, 0x20), 0x40)
    }
    require(ret);

    assembly {
      ret := staticcall(gas(), 6, add(input, 0x20), 0x80, add(input, 0x60), 0x40)
    }
    require(ret);
  }

  function ecc_mul_add(
    uint256 px,
    uint256 py,
    uint256 s,
    uint256 qx,
    uint256 qy
  ) internal view returns (uint256, uint256) {
    uint256[5] memory input;
    input[0] = px;
    input[1] = py;
    input[2] = s;
    input[3] = qx;
    input[4] = qy;

    _ecc_mul_add(input);

    return (input[3], input[4]);
  }

  function ecc_mul_add_pm(
    uint256[78] memory m,
    uint256[] calldata proof,
    uint256 opcode,
    uint256 t0,
    uint256 t1
  ) internal view returns (uint256, uint256) {
    uint256[5] memory input;
    input[3] = t0;
    input[4] = t1;
    for (uint256 i = 0; i < 32; i += 2) {
      uint256 a = opcode & 0xff;
      if (a != 0xff) {
        opcode >>= 8;
        uint256 b = opcode & 0xff;
        opcode >>= 8;
        input[0] = proof[a];
        input[1] = proof[a + 1];
        input[2] = m[b];
        _ecc_mul_add(input);
      } else {
        break;
      }
    }

    return (input[3], input[4]);
  }

  function update_hash_scalar(
    uint256 v,
    uint256[144] memory absorbing,
    uint256 pos
  ) internal pure {
    absorbing[pos++] = 0x02;
    absorbing[pos++] = v;
  }

  function update_hash_point(
    uint256 x,
    uint256 y,
    uint256[144] memory absorbing,
    uint256 pos
  ) internal pure {
    absorbing[pos++] = 0x01;
    absorbing[pos++] = x;
    absorbing[pos++] = y;
  }

  function to_scalar(bytes32 r) private pure returns (uint256 v) {
    uint256 tmp = uint256(r);
    tmp = fr_reverse(tmp);
    v = tmp % 0x30644e72e131a029b85045b68181585d2833e84879b9709143e1f593f0000001;
  }

  function hash(uint256[144] memory absorbing, uint256 length) private view returns (bytes32[1] memory v) {
    bool success;
    assembly {
      success := staticcall(sub(gas(), 2000), 2, absorbing, length, v, 32)
      switch success
      case 0 {
        invalid()
      }
    }
    assert(success);
  }

  function squeeze_challenge(uint256[144] memory absorbing, uint32 length) internal view returns (uint256 v) {
    absorbing[length] = 0;
    bytes32 res = hash(absorbing, length * 32 + 1)[0];
    v = to_scalar(res);
    absorbing[0] = uint256(res);
    length = 1;
  }

  function get_verify_circuit_g2_s() internal pure returns (G2Point memory s) {
    s.x[0] = uint256(19996377281670978687180986182441301914718493784645870391946826878753710639456);
    s.x[1] = uint256(4287478848095488335912479212753150961411468232106701703291869721868407715111);
    s.y[0] = uint256(6995741485533723263267942814565501722132921805029874890336635619836737653877);
    s.y[1] = uint256(11126659726611658836425410744462014686753643655648740844565393330984713428953);
  }

  function get_verify_circuit_g2_n() internal pure returns (G2Point memory n) {
    n.x[0] = uint256(11559732032986387107991004021392285783925812861821192530917403151452391805634);
    n.x[1] = uint256(10857046999023057135944570762232829481370756359578518086990519993285655852781);
    n.y[0] = uint256(17805874995975841540914202342111839520379459829704422454583296818431106115052);
    n.y[1] = uint256(13392588948715843804641432497768002650278120570034223513918757245338268106653);
  }

  function get_target_circuit_g2_s() internal pure returns (G2Point memory s) {
    s.x[0] = uint256(19996377281670978687180986182441301914718493784645870391946826878753710639456);
    s.x[1] = uint256(4287478848095488335912479212753150961411468232106701703291869721868407715111);
    s.y[0] = uint256(6995741485533723263267942814565501722132921805029874890336635619836737653877);
    s.y[1] = uint256(11126659726611658836425410744462014686753643655648740844565393330984713428953);
  }

  function get_target_circuit_g2_n() internal pure returns (G2Point memory n) {
    n.x[0] = uint256(11559732032986387107991004021392285783925812861821192530917403151452391805634);
    n.x[1] = uint256(10857046999023057135944570762232829481370756359578518086990519993285655852781);
    n.y[0] = uint256(17805874995975841540914202342111839520379459829704422454583296818431106115052);
    n.y[1] = uint256(13392588948715843804641432497768002650278120570034223513918757245338268106653);
  }

  function get_wx_wg(uint256[] calldata proof, uint256[4] memory instances)
    internal
    view
    returns (
      uint256,
      uint256,
      uint256,
      uint256
    )
  {
    uint256[78] memory m;
    uint256[144] memory absorbing;
    uint256 t0 = 0;
    uint256 t1 = 0;

    (t0, t1) = (
      ecc_mul(
        13911018583007884881416842514661274050567796652031922980888952067142200734890,
        6304656948134906299141761906515211516376236447819044970320185642735642777036,
        instances[0]
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        10634526547038245645834822324032425487434811507756950001533785848774317018670,
        11025818855933089539342999945076144168100709119485154428833847826982360951459,
        instances[1],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        13485936455723319058155687139769502499697405985650416391707184524158646623799,
        16234009237501684544798205490615498675425737095147152991328466405207467143566,
        instances[2],
        t0,
        t1
      )
    );
    (m[0], m[1]) = (
      ecc_mul_add(
        21550585789286941025166870525096478397065943995678337623823808437877187678077,
        4447338868884713453743453617617291019986465683944733951178865127876671635659,
        instances[3],
        t0,
        t1
      )
    );
    update_hash_scalar(7565563496810572832679683861627381535096739771067228659745730142637512143527, absorbing, 0);
    update_hash_point(m[0], m[1], absorbing, 2);
    for (t0 = 0; t0 <= 4; t0++) {
      update_hash_point(proof[0 + t0 * 2], proof[1 + t0 * 2], absorbing, 5 + t0 * 3);
    }
    m[2] = (squeeze_challenge(absorbing, 20));
    for (t0 = 0; t0 <= 13; t0++) {
      update_hash_point(proof[10 + t0 * 2], proof[11 + t0 * 2], absorbing, 1 + t0 * 3);
    }
    m[3] = (squeeze_challenge(absorbing, 43));
    m[4] = (squeeze_challenge(absorbing, 1));
    for (t0 = 0; t0 <= 9; t0++) {
      update_hash_point(proof[38 + t0 * 2], proof[39 + t0 * 2], absorbing, 1 + t0 * 3);
    }
    m[5] = (squeeze_challenge(absorbing, 31));
    for (t0 = 0; t0 <= 3; t0++) {
      update_hash_point(proof[58 + t0 * 2], proof[59 + t0 * 2], absorbing, 1 + t0 * 3);
    }
    m[6] = (squeeze_challenge(absorbing, 13));
    for (t0 = 0; t0 <= 70; t0++) {
      update_hash_scalar(proof[66 + t0 * 1], absorbing, 1 + t0 * 2);
    }
    m[7] = (squeeze_challenge(absorbing, 143));
    m[8] = (squeeze_challenge(absorbing, 1));
    for (t0 = 0; t0 <= 3; t0++) {
      update_hash_point(proof[137 + t0 * 2], proof[138 + t0 * 2], absorbing, 1 + t0 * 3);
    }
    m[9] = (mulmod(m[6], 6143038923529407703646399695489445107254060255791852207908457597807435305312, q_mod));
    m[10] = (mulmod(m[6], 7358966525675286471217089135633860168646304224547606326237275077574224349359, q_mod));
    m[11] = (mulmod(m[6], 11377606117859914088982205826922132024839443553408109299929510653283289974216, q_mod));
    m[12] = (fr_pow(m[6], 33554432));
    m[13] = (addmod(m[12], q_mod - 1, q_mod));
    m[14] = (mulmod(21888242219518804655518433051623070663413851959604507555939307129453691614729, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 1, q_mod));
    m[14] = (fr_div(m[14], t0));
    m[15] = (mulmod(3814514741328848551622746860665626251343731549210296844380905280010844577811, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 11377606117859914088982205826922132024839443553408109299929510653283289974216, q_mod));
    m[15] = (fr_div(m[15], t0));
    m[16] = (mulmod(14167635312934689395373925807699824183296350635557349457928542208657273886961, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 17329448237240114492580865744088056414251735686965494637158808787419781175510, q_mod));
    m[16] = (fr_div(m[16], t0));
    m[17] = (mulmod(12609034248192017902501772617940356704925468750503023243291639149763830461639, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 16569469942529664681363945218228869388192121720036659574609237682362097667612, q_mod));
    m[17] = (fr_div(m[17], t0));
    m[18] = (mulmod(12805242257443675784492534138904933930037912868081131057088370227525924812579, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 9741553891420464328295280489650144566903017206473301385034033384879943874347, q_mod));
    m[18] = (fr_div(m[18], t0));
    m[19] = (mulmod(6559137297042406441428413756926584610543422337862324541665337888392460442551, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 5723528081196465413808013109680264505774289533922470433187916976440924869204, q_mod));
    m[19] = (fr_div(m[19], t0));
    m[20] = (mulmod(14811589476322888753142612645486192973009181596950146578897598212834285850868, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 7358966525675286471217089135633860168646304224547606326237275077574224349359, q_mod));
    m[20] = (fr_div(m[20], t0));
    t0 = (addmod(m[15], m[16], q_mod));
    t0 = (addmod(t0, m[17], q_mod));
    t0 = (addmod(t0, m[18], q_mod));
    m[15] = (addmod(t0, m[19], q_mod));
    t0 = (fr_mul_add(proof[74], proof[72], proof[73]));
    t0 = (fr_mul_add(proof[75], proof[67], t0));
    t0 = (fr_mul_add(proof[76], proof[68], t0));
    t0 = (fr_mul_add(proof[77], proof[69], t0));
    t0 = (fr_mul_add(proof[78], proof[70], t0));
    m[16] = (fr_mul_add(proof[79], proof[71], t0));
    t0 = (mulmod(proof[67], proof[68], q_mod));
    m[16] = (fr_mul_add(proof[80], t0, m[16]));
    t0 = (mulmod(proof[69], proof[70], q_mod));
    m[16] = (fr_mul_add(proof[81], t0, m[16]));
    t0 = (addmod(1, q_mod - proof[97], q_mod));
    m[17] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[100], proof[100], q_mod));
    t0 = (addmod(t0, q_mod - proof[100], q_mod));
    m[18] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(proof[100], q_mod - proof[99], q_mod));
    m[19] = (mulmod(t0, m[14], q_mod));
    m[21] = (mulmod(m[3], m[6], q_mod));
    t0 = (addmod(m[20], m[15], q_mod));
    m[15] = (addmod(1, q_mod - t0, q_mod));
    m[22] = (addmod(proof[67], m[4], q_mod));
    t0 = (fr_mul_add(proof[91], m[3], m[22]));
    m[23] = (mulmod(t0, proof[98], q_mod));
    t0 = (addmod(m[22], m[21], q_mod));
    m[22] = (mulmod(t0, proof[97], q_mod));
    m[24] = (mulmod(4131629893567559867359510883348571134090853742863529169391034518566172092834, m[21], q_mod));
    m[25] = (addmod(proof[68], m[4], q_mod));
    t0 = (fr_mul_add(proof[92], m[3], m[25]));
    m[23] = (mulmod(t0, m[23], q_mod));
    t0 = (addmod(m[25], m[24], q_mod));
    m[22] = (mulmod(t0, m[22], q_mod));
    m[24] = (mulmod(4131629893567559867359510883348571134090853742863529169391034518566172092834, m[24], q_mod));
    m[25] = (addmod(proof[69], m[4], q_mod));
    t0 = (fr_mul_add(proof[93], m[3], m[25]));
    m[23] = (mulmod(t0, m[23], q_mod));
    t0 = (addmod(m[25], m[24], q_mod));
    m[22] = (mulmod(t0, m[22], q_mod));
    m[24] = (mulmod(4131629893567559867359510883348571134090853742863529169391034518566172092834, m[24], q_mod));
    t0 = (addmod(m[23], q_mod - m[22], q_mod));
    m[22] = (mulmod(t0, m[15], q_mod));
    m[21] = (mulmod(m[21], 11166246659983828508719468090013646171463329086121580628794302409516816350802, q_mod));
    m[23] = (addmod(proof[70], m[4], q_mod));
    t0 = (fr_mul_add(proof[94], m[3], m[23]));
    m[24] = (mulmod(t0, proof[101], q_mod));
    t0 = (addmod(m[23], m[21], q_mod));
    m[23] = (mulmod(t0, proof[100], q_mod));
    m[21] = (mulmod(4131629893567559867359510883348571134090853742863529169391034518566172092834, m[21], q_mod));
    m[25] = (addmod(proof[71], m[4], q_mod));
    t0 = (fr_mul_add(proof[95], m[3], m[25]));
    m[24] = (mulmod(t0, m[24], q_mod));
    t0 = (addmod(m[25], m[21], q_mod));
    m[23] = (mulmod(t0, m[23], q_mod));
    m[21] = (mulmod(4131629893567559867359510883348571134090853742863529169391034518566172092834, m[21], q_mod));
    m[25] = (addmod(proof[66], m[4], q_mod));
    t0 = (fr_mul_add(proof[96], m[3], m[25]));
    m[24] = (mulmod(t0, m[24], q_mod));
    t0 = (addmod(m[25], m[21], q_mod));
    m[23] = (mulmod(t0, m[23], q_mod));
    m[21] = (mulmod(4131629893567559867359510883348571134090853742863529169391034518566172092834, m[21], q_mod));
    t0 = (addmod(m[24], q_mod - m[23], q_mod));
    m[21] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[104], m[3], q_mod));
    m[23] = (mulmod(proof[103], t0, q_mod));
    t0 = (addmod(proof[106], m[4], q_mod));
    m[23] = (mulmod(m[23], t0, q_mod));
    m[24] = (mulmod(proof[67], proof[82], q_mod));
    m[2] = (mulmod(0, m[2], q_mod));
    m[24] = (addmod(m[2], m[24], q_mod));
    m[25] = (addmod(m[2], proof[83], q_mod));
    m[26] = (addmod(proof[104], q_mod - proof[106], q_mod));
    t0 = (addmod(1, q_mod - proof[102], q_mod));
    m[27] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[102], proof[102], q_mod));
    t0 = (addmod(t0, q_mod - proof[102], q_mod));
    m[28] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[24], m[3], q_mod));
    m[24] = (mulmod(proof[102], t0, q_mod));
    m[25] = (addmod(m[25], m[4], q_mod));
    t0 = (mulmod(m[24], m[25], q_mod));
    t0 = (addmod(m[23], q_mod - t0, q_mod));
    m[23] = (mulmod(t0, m[15], q_mod));
    m[24] = (mulmod(m[14], m[26], q_mod));
    t0 = (addmod(proof[104], q_mod - proof[105], q_mod));
    t0 = (mulmod(m[26], t0, q_mod));
    m[26] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[109], m[3], q_mod));
    m[29] = (mulmod(proof[108], t0, q_mod));
    t0 = (addmod(proof[111], m[4], q_mod));
    m[29] = (mulmod(m[29], t0, q_mod));
    m[30] = (fr_mul_add(proof[82], proof[68], m[2]));
    m[31] = (addmod(proof[109], q_mod - proof[111], q_mod));
    t0 = (addmod(1, q_mod - proof[107], q_mod));
    m[32] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[107], proof[107], q_mod));
    t0 = (addmod(t0, q_mod - proof[107], q_mod));
    m[33] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[30], m[3], q_mod));
    t0 = (mulmod(proof[107], t0, q_mod));
    t0 = (mulmod(t0, m[25], q_mod));
    t0 = (addmod(m[29], q_mod - t0, q_mod));
    m[29] = (mulmod(t0, m[15], q_mod));
    m[30] = (mulmod(m[14], m[31], q_mod));
    t0 = (addmod(proof[109], q_mod - proof[110], q_mod));
    t0 = (mulmod(m[31], t0, q_mod));
    m[31] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[114], m[3], q_mod));
    m[34] = (mulmod(proof[113], t0, q_mod));
    t0 = (addmod(proof[116], m[4], q_mod));
    m[34] = (mulmod(m[34], t0, q_mod));
    m[35] = (fr_mul_add(proof[82], proof[69], m[2]));
    m[36] = (addmod(proof[114], q_mod - proof[116], q_mod));
    t0 = (addmod(1, q_mod - proof[112], q_mod));
    m[37] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[112], proof[112], q_mod));
    t0 = (addmod(t0, q_mod - proof[112], q_mod));
    m[38] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[35], m[3], q_mod));
    t0 = (mulmod(proof[112], t0, q_mod));
    t0 = (mulmod(t0, m[25], q_mod));
    t0 = (addmod(m[34], q_mod - t0, q_mod));
    m[34] = (mulmod(t0, m[15], q_mod));
    m[35] = (mulmod(m[14], m[36], q_mod));
    t0 = (addmod(proof[114], q_mod - proof[115], q_mod));
    t0 = (mulmod(m[36], t0, q_mod));
    m[36] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[119], m[3], q_mod));
    m[39] = (mulmod(proof[118], t0, q_mod));
    t0 = (addmod(proof[121], m[4], q_mod));
    m[39] = (mulmod(m[39], t0, q_mod));
    m[40] = (fr_mul_add(proof[82], proof[70], m[2]));
    m[41] = (addmod(proof[119], q_mod - proof[121], q_mod));
    t0 = (addmod(1, q_mod - proof[117], q_mod));
    m[42] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[117], proof[117], q_mod));
    t0 = (addmod(t0, q_mod - proof[117], q_mod));
    m[43] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[40], m[3], q_mod));
    t0 = (mulmod(proof[117], t0, q_mod));
    t0 = (mulmod(t0, m[25], q_mod));
    t0 = (addmod(m[39], q_mod - t0, q_mod));
    m[25] = (mulmod(t0, m[15], q_mod));
    m[39] = (mulmod(m[14], m[41], q_mod));
    t0 = (addmod(proof[119], q_mod - proof[120], q_mod));
    t0 = (mulmod(m[41], t0, q_mod));
    m[40] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[124], m[3], q_mod));
    m[41] = (mulmod(proof[123], t0, q_mod));
    t0 = (addmod(proof[126], m[4], q_mod));
    m[41] = (mulmod(m[41], t0, q_mod));
    m[44] = (fr_mul_add(proof[84], proof[67], m[2]));
    m[45] = (addmod(m[2], proof[85], q_mod));
    m[46] = (addmod(proof[124], q_mod - proof[126], q_mod));
    t0 = (addmod(1, q_mod - proof[122], q_mod));
    m[47] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[122], proof[122], q_mod));
    t0 = (addmod(t0, q_mod - proof[122], q_mod));
    m[48] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[44], m[3], q_mod));
    m[44] = (mulmod(proof[122], t0, q_mod));
    t0 = (addmod(m[45], m[4], q_mod));
    t0 = (mulmod(m[44], t0, q_mod));
    t0 = (addmod(m[41], q_mod - t0, q_mod));
    m[41] = (mulmod(t0, m[15], q_mod));
    m[44] = (mulmod(m[14], m[46], q_mod));
    t0 = (addmod(proof[124], q_mod - proof[125], q_mod));
    t0 = (mulmod(m[46], t0, q_mod));
    m[45] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[129], m[3], q_mod));
    m[46] = (mulmod(proof[128], t0, q_mod));
    t0 = (addmod(proof[131], m[4], q_mod));
    m[46] = (mulmod(m[46], t0, q_mod));
    m[49] = (fr_mul_add(proof[86], proof[67], m[2]));
    m[50] = (addmod(m[2], proof[87], q_mod));
    m[51] = (addmod(proof[129], q_mod - proof[131], q_mod));
    t0 = (addmod(1, q_mod - proof[127], q_mod));
    m[52] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[127], proof[127], q_mod));
    t0 = (addmod(t0, q_mod - proof[127], q_mod));
    m[53] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[49], m[3], q_mod));
    m[49] = (mulmod(proof[127], t0, q_mod));
    t0 = (addmod(m[50], m[4], q_mod));
    t0 = (mulmod(m[49], t0, q_mod));
    t0 = (addmod(m[46], q_mod - t0, q_mod));
    m[46] = (mulmod(t0, m[15], q_mod));
    m[49] = (mulmod(m[14], m[51], q_mod));
    t0 = (addmod(proof[129], q_mod - proof[130], q_mod));
    t0 = (mulmod(m[51], t0, q_mod));
    m[50] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[134], m[3], q_mod));
    m[51] = (mulmod(proof[133], t0, q_mod));
    t0 = (addmod(proof[136], m[4], q_mod));
    m[51] = (mulmod(m[51], t0, q_mod));
    m[54] = (fr_mul_add(proof[88], proof[67], m[2]));
    m[2] = (addmod(m[2], proof[89], q_mod));
    m[55] = (addmod(proof[134], q_mod - proof[136], q_mod));
    t0 = (addmod(1, q_mod - proof[132], q_mod));
    m[56] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[132], proof[132], q_mod));
    t0 = (addmod(t0, q_mod - proof[132], q_mod));
    m[20] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[54], m[3], q_mod));
    m[3] = (mulmod(proof[132], t0, q_mod));
    t0 = (addmod(m[2], m[4], q_mod));
    t0 = (mulmod(m[3], t0, q_mod));
    t0 = (addmod(m[51], q_mod - t0, q_mod));
    m[2] = (mulmod(t0, m[15], q_mod));
    m[3] = (mulmod(m[14], m[55], q_mod));
    t0 = (addmod(proof[134], q_mod - proof[135], q_mod));
    t0 = (mulmod(m[55], t0, q_mod));
    m[4] = (mulmod(t0, m[15], q_mod));
    t0 = (fr_mul_add(m[5], 0, m[16]));
    t0 = (fr_mul_add_mt(m, m[5], 24064768791442479290152634096194013545513974547709823832001394403118888981009, t0));
    t0 = (fr_mul_add_mt(m, m[5], 4704208815882882920750, t0));
    m[2] = (fr_div(t0, m[13]));
    m[3] = (mulmod(m[8], m[8], q_mod));
    m[4] = (mulmod(m[3], m[8], q_mod));
    (t0, t1) = (ecc_mul(proof[137], proof[138], m[4]));
    (t0, t1) = (ecc_mul_add_pm(m, proof, 281470825202571, t0, t1));
    (m[14], m[15]) = (ecc_add(t0, t1, proof[143], proof[144]));
    m[5] = (mulmod(m[4], m[10], q_mod));
    m[10] = (mulmod(m[4], proof[99], q_mod));
    m[11] = (mulmod(m[3], m[11], q_mod));
    m[13] = (mulmod(m[3], m[7], q_mod));
    m[16] = (mulmod(m[13], m[7], q_mod));
    m[17] = (mulmod(m[16], m[7], q_mod));
    m[18] = (mulmod(m[17], m[7], q_mod));
    m[19] = (mulmod(m[18], m[7], q_mod));
    m[20] = (mulmod(m[19], m[7], q_mod));
    t0 = (mulmod(m[20], proof[105], q_mod));
    t0 = (fr_mul_add_pm(m, proof, 5192218722096118505335019273393006, t0));
    m[10] = (addmod(m[10], t0, q_mod));
    m[6] = (mulmod(m[8], m[6], q_mod));
    m[21] = (mulmod(m[8], m[7], q_mod));
    for (t0 = 0; t0 < 52; t0++) {
      m[22 + t0 * 1] = (mulmod(m[21 + t0 * 1], m[7 + t0 * 0], q_mod));
    }
    t0 = (mulmod(m[73], proof[66], q_mod));
    t0 = (fr_mul_add_pm(m, proof, 25987190009742107077980742527956132804769685504365379353571332812354881865795, t0));
    t0 = (fr_mul_add_pm(m, proof, 18679399068738585913008893864493214572484549614980916660536066406366626396277, t0));
    t0 = (fr_mul_add_pm(m, proof, 11472319920207072041878598272885343947088038914199705598762544978176638855245, t0));
    t0 = (fr_mul_add_pm(m, proof, 281471073851486, t0));
    m[74] = (fr_mul_add(proof[96], m[22], t0));
    m[75] = (mulmod(m[21], m[12], q_mod));
    m[76] = (mulmod(m[75], m[12], q_mod));
    m[12] = (mulmod(m[76], m[12], q_mod));
    t0 = (fr_mul_add(m[21], m[2], m[74]));
    t0 = (fr_mul_add(proof[90], m[8], t0));
    m[2] = (addmod(m[10], t0, q_mod));
    m[4] = (addmod(m[4], m[67], q_mod));
    m[10] = (addmod(m[20], m[64], q_mod));
    m[19] = (addmod(m[19], m[61], q_mod));
    m[18] = (addmod(m[18], m[58], q_mod));
    m[17] = (addmod(m[17], m[55], q_mod));
    m[16] = (addmod(m[16], m[52], q_mod));
    m[13] = (addmod(m[13], m[49], q_mod));
    m[3] = (addmod(m[3], m[46], q_mod));
    m[20] = (mulmod(m[7], m[7], q_mod));
    m[46] = (mulmod(m[20], m[7], q_mod));
    for (t0 = 0; t0 < 6; t0++) {
      m[49 + t0 * 3] = (mulmod(m[46 + t0 * 3], m[7 + t0 * 0], q_mod));
    }
    t0 = (mulmod(m[64], proof[72], q_mod));
    t0 = (fr_mul_add_pm(m, proof, 22300414885789078225200772312192282479902050, t0));
    m[67] = (addmod(t0, proof[133], q_mod));
    m[64] = (addmod(m[68], m[64], q_mod));
    m[2] = (addmod(m[2], m[67], q_mod));
    m[4] = (addmod(m[4], m[61], q_mod));
    m[58] = (addmod(m[66], m[58], q_mod));
    m[55] = (addmod(m[65], m[55], q_mod));
    m[52] = (addmod(m[62], m[52], q_mod));
    m[49] = (addmod(m[59], m[49], q_mod));
    m[46] = (addmod(m[56], m[46], q_mod));
    m[20] = (addmod(m[53], m[20], q_mod));
    m[7] = (addmod(m[50], m[7], q_mod));
    m[47] = (addmod(m[47], 1, q_mod));
    (t0, t1) = (ecc_mul(proof[137], proof[138], m[5]));
    (t0, t1) = (ecc_mul_add_pm(m, proof, 95779547201103344574663521248920622570100289727824934, t0, t1));
    (t0, t1) = (ecc_mul_add(m[0], m[1], m[73], t0, t1));
    (t0, t1) = (
      ecc_mul_add_pm(m, proof, 23117566384181460736372107411586488455996274321045495459183463611775605426176, t0, t1)
    );
    (t0, t1) = (ecc_mul_add_pm(m, proof, 1208910625647296115640116, t0, t1));
    (t0, t1) = (
      ecc_mul_add(
        18203201369910127748653093239046925262331867792564567575715419312489770354152,
        21337935618380961062706628489144973405767465584115959095575086935926375008565,
        m[44],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        7424704028332535427089305319864133204532066896526891781118451245849784254708,
        12678856732599950219016748766794420664612259488496142493506929751242408175780,
        m[43],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        8957037383966114205039201379598315116392474748202370204432548294176569739025,
        28893144485358453797177540052763531794017266671779456104655986575591563425,
        m[42],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        8899458845706710365757662322486820909933020909173771476551503677327456268940,
        17943661811108313529459365208510090779520246001781766573073385652501929352756,
        m[41],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        2066192237212045571380353294172299821813238583585695797659665519337931185322,
        12893117415479244053731985851205411826087268368524437394295109896310630419016,
        m[40],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        7029209694864206103748719578587258594999467058459124354420673099152700042635,
        155042903642804194607913895998475761748212512551291074467541114278976537732,
        m[39],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        16259069680733604636667370958538524295394410112802664620441902480921241179420,
        17488623510549326881754440343703364765315186391411575518778842897050730190490,
        m[38],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        8407488098623013246100134722886116864122098390579548782136305885068409559706,
        3568146295252833243435443545345500897014052457217198721664547400431876704581,
        m[37],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        5695240006165323166776258492529211703695708080346745066944671822978474788477,
        5906437993123332765602165777880337958638812398082372651201793656017332416828,
        m[36],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        2659006490238079124981436484030257425933934727839646251920092277478167608717,
        21267095543134844017717273781957151356162397753509908685868267465378266613009,
        m[35],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        11667150339256836494926506499230187360957884531183800528342644917396989453992,
        15540782144062394272475578831064080588044323224200171932910650185556553066875,
        m[34],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        14538287369116104122244775799647649410451760052847570378748695199010853240168,
        8755608829971274804476073327578326530208497176627947686849099256174562639267,
        m[33],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        8808629196631084710334110767449499515582902470045288549019060600095073238105,
        13294364470509711632739201553507258372326885785844949555702886281377427438475,
        m[32],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        13530039227429344427307885259315348094603239544740319258739863478267732941156,
        14620961799645572759159810469728918487803767644700931469827291205450509619585,
        m[31],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        20143075587083355112417414887372164250381042430441089145485481665404780784123,
        9674175910548207533970570126063643897609459066877075659644076646142886425503,
        m[30],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        20838181470940778746497458037822874891443259982457936197338585360188045646865,
        17604436498939349000552743603444692514421198196632934037915131564076907882457,
        m[29],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        8808629196631084710334110767449499515582902470045288549019060600095073238105,
        13294364470509711632739201553507258372326885785844949555702886281377427438475,
        m[28],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        4485596020921606218295723396096228276271826489358088483611583353683289026870,
        13510458114075088326282033836278698875863675653560040772231774870357268688709,
        m[27],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        5689866494008618407240588637047214252297874578255941138955533598036931418426,
        2300693805333588771389246453785873951508203893413051563103782308268989878392,
        m[26],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        5369038269427160378147433138732024697166237728341087293257688719583044616678,
        15700448579924136666314696630042469274031007615486805958631969804767251063409,
        m[25],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        11978866022148046334703072073665622533545779572475689419419225265186628184748,
        6003507861920008241570845663435940331649107374272819554259170920205785257391,
        m[24],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        19541682318825983281360568185450727788672304379755672087471546806768410813080,
        7228748902536238479110940789248141601208539488548995028410294630493235254571,
        m[23],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        9286666528678535158794564481311446553441466915226232276501961953188461631089,
        10206803073576976981612889266580882628230194403040886323606748430787220964730,
        m[22],
        t0,
        t1
      )
    );
    (t0, t1) = (ecc_mul_add_pm(m, proof, 79226992401923871795060804672, t0, t1));
    (m[0], m[1]) = (ecc_mul_add(proof[143], proof[144], m[9], t0, t1));
    (t0, t1) = (ecc_mul(1, 2, m[2]));
    (m[0], m[1]) = (ecc_sub(m[0], m[1], t0, t1));
    return (m[14], m[15], m[0], m[1]);
  }

  function verify(uint256[] calldata proof, uint256[] calldata target_circuit_final_pair) public view {
    uint256[4] memory instances;
    instances[0] = target_circuit_final_pair[0] & ((1 << 136) - 1);
    instances[1] = (target_circuit_final_pair[0] >> 136) + ((target_circuit_final_pair[1] & 1) << 136);
    instances[2] = target_circuit_final_pair[2] & ((1 << 136) - 1);
    instances[3] = (target_circuit_final_pair[2] >> 136) + ((target_circuit_final_pair[3] & 1) << 136);

    uint256 x0 = 0;
    uint256 x1 = 0;
    uint256 y0 = 0;
    uint256 y1 = 0;

    G1Point[] memory g1_points = new G1Point[](2);
    G2Point[] memory g2_points = new G2Point[](2);
    bool checked = false;

    (x0, y0, x1, y1) = get_wx_wg(proof, instances);
    g1_points[0].x = x0;
    g1_points[0].y = y0;
    g1_points[1].x = x1;
    g1_points[1].y = y1;
    g2_points[0] = get_verify_circuit_g2_s();
    g2_points[1] = get_verify_circuit_g2_n();

    checked = pairing(g1_points, g2_points);
    require(checked, "verified failed");

    g1_points[0].x = target_circuit_final_pair[0];
    g1_points[0].y = target_circuit_final_pair[1];
    g1_points[1].x = target_circuit_final_pair[2];
    g1_points[1].y = target_circuit_final_pair[3];
    g2_points[0] = get_target_circuit_g2_s();
    g2_points[1] = get_target_circuit_g2_n();

    checked = pairing(g1_points, g2_points);
    require(checked, "verified failed");
  }
}
