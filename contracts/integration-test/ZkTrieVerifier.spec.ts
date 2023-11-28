/* eslint-disable node/no-unpublished-import */
/* eslint-disable node/no-missing-import */
import { expect } from "chai";
import { concat } from "ethers/lib/utils";
import { ethers } from "hardhat";
import { MockZkTrieVerifier } from "../typechain";

import { generateABI, createCode } from "../scripts/poseidon";

const chars = "0123456789abcdef";

interface ITestConfig {
  block: number;
  desc: string;
  account: string;
  storage: string;
  expectedRoot: string;
  expectedValue: string;
  accountProof: string[];
  storageProof: string[];
}

const testcases: Array<ITestConfig> = [
  {
    // curl -H "content-type: application/json" -X POST --data '{"id":0,"jsonrpc":"2.0","method":"eth_getProof","params":["0x5300000000000000000000000000000000000004", ["0x8391082587ea494a8beba02cc40273f27e5477a967cd400736ac46950da0b378"], "0x1111ad"]}' https://rpc.scroll.io
    block: 1118637,
    desc: "WETH.balance[0xa7994f02237aed2c116a702a8f5322a1fb325b31]",
    account: "0x5300000000000000000000000000000000000004",
    storage: "0x8391082587ea494a8beba02cc40273f27e5477a967cd400736ac46950da0b378",
    expectedRoot: "0x1334a21a74914182745c1f5142e70b487262096784ae7669186657462c01b103",
    expectedValue: "0x00000000000000000000000000000000000000000000000000006239b5a2c000",
    accountProof: [
      "0x0907d980105678a2007eb5683d850f36a9caafe6e7fd3279987d7a94a13a360d3a1478f9a4c1f8c755227ee3544929bb0d7cfa2d999a48493d048ff0250bb002ab",
      "0x092b59a024f142555555c767842c4fcc3996686c57699791fcb10013f69ffd9b2507360087cb303767fd43f2650960621246a8d205d086e03d9c1626e4aaa5b143",
      "0x091f876342916ac1d5a14ef40cfc5644452170b16d1b045877f303cd52322ba1e00ba09f36443c2a63fbd7ff8feeb2c84e99fde6db08fd8e4c67ad061c482ff276",
      "0x09277b3069a4b944a45df222366aae727ec64efaf0a8ecb000645d0eea3a3fa93609b925158cc04f610f8c616369094683ca7a86239f49e97852aa286d148a3913",
      "0x092fb789200a7324067934da8be91c48f86c4e6f35fed6d1ce8ae4d7051f480bc0074019222c788b139b6919dfbc9d0b51f274e0ed3ea03553b8db30392ac05ce4",
      "0x092f79da8f9f2c3a3a3813580ff18d4619b95f54026b2f16ccbcca684d5e25e1f52912fa319d9a7ba537a52cc6571844b4d1aa99b8a78cea6f686a6279ade5dcae",
      "0x09249d249bcf92a369bd7715ec63a4b29d706a5dbb304efd678a2e5d7982e7fa9b202e3225c1031d83ab62d78516a4cbdbf2b22842c57182e7cb0dbb4303ac38c5",
      "0x0904837ebb85ceccab225d4d826fe57edca4b00862199b91082f65dfffa7669b90039c710273b02e60c2e74eb8b243721e852e0e56fa51668b6362fd920f817cb7",
      "0x090a36f6aabc3768a05dd8f93667a0eb2e5b63d94b5ce27132fb38d13c56d49cb4249c2013daee90184ae285226271f150f6a8f74f2c85dbd0721c5f583e620b10",
      "0x091b82f139a06af573e871fdd5f5ac18f17c568ffe1c9e271505b371ad7f0603e716b187804a49d2456a0baa7c2317c14d9aa7e58ad64df38bc6c1c7b86b072333",
      "0x0929668e59dfc2e2aef10194f5d287d8396e1a897d68f106bdb12b9541c0bab71d2bf910dea11e3209b3feff88d630af46006e402e935bc84c559694d88c117733",
      "0x0914231c92f09f56628c10603dc2d2120d9d11b27fa23753a14171127c3a1ee3dd0d6b9cbd11d031fe6e1b650023edc58aa580fa4f4aa1b30bf82e0e4c7a308bb9",
      "0x0914c1dd24c520d96aac93b7ef3062526067f1b15a080c482abf449d3c2cde781b195eb63b5e328572090319310914d81b2ca8350b6e15dc9d13e878f8c28c9d52",
      "0x0927cb93e3d9c144a5a3653c5cf2ed5940d64f461dd588cd192516ae7d855e9408166e85986d4c9836cd6cd822174ba9db9c7a043d73e86b5b2cfc0a2e082894c3",
      "0x090858bf8a0119626fe9339bd92116a070ba1a66423b0f7d3f4666b6851fdea01400f7f51eb22df168c41162d7f18f9d97155d87da523b05a1dde54e7a30a98c31",
      "0x0902776c1f5f93a95baea2e209ddb4a5e49dd1112a7f7d755a45addffe4a233dad0d8cc62b957d9b254fdc8199c720fcf8d5c65d14899911e991b4530710aca75e",
      "0x091d7fde5c78c88bbf6082a20a185cde96a203ea0d29c829c1ab9322fc3ca0ae3100ef7cba868cac216d365a0232ad6227ab1ef3290166bc6c19b719b79dbc17fc",
      "0x091690160269c53c6b74337a00d02cb40a88ea5eba06e1942088b619baee83279e12d96d62dda9c4b5897d58fea40b5825d87a5526dec37361ec7c93a3256ea76d",
      "0x091bccb091cde3f8ca7cfda1df379c9bfa412908c41037ae4ec0a20ce984e2c9a51d02c109d2e6e25dc60f10b1bc3b3f97ca1ce1aa025ce4f3146de3979403b99e",
      "0x0927083540af95e57acba69671a4a596f721432549b8760941f4251e0dd7a013a917cee0f60d333cf88e40ae8710fb1fd6e3920346a376b3ba6686a4b2020a043e",
      "0x082170b57b8f05f6990eec62e74cdb303741f6c464a85d68582c19c51e53f490000a5029a62ddc14c9c07c549db300bd308b6367454966c94b8526f4ceed5693b2",
      "0x0827a0b16ef333dcfe00610d19dc468b9e856f544c9b5e9b046357e0a38aedaeb90000000000000000000000000000000000000000000000000000000000000000",
      "0x06126f891e8753e67c5cbfa2a67e9d71942eab3a88cde86e97a4af94ea0dde497821fb69ccdb00e6eaeaf7fc1e73630f39f846970b72ac801e396da0033fb0c247",
      "0x0420e9fb498ff9c35246d527da24aa1710d2cc9b055ecf9a95a8a2a11d3d836cdf050800000000000000000000000000000000000000000000000016ef00000000000000000000000000000000000000000000000000000000000000600058d1a5ce14104d0dedcaecaab39b6e22c2608e40af67a71908e6e97bbf4a43c59c4537140c25a9e8c4073351c26b9831c1e5af153b9be4713a4af9edfdf32b58077b735e120f14136a7980da529d9e8d3a71433fc9dc5aa8c01e3a4eb60cb3a4f9cf9ca5c8e0be205300000000000000000000000000000000000004000000000000000000000000",
      "0x5448495320495320534f4d45204d4147494320425954455320464f5220534d54206d3172525867503278704449",
    ],
    storageProof: [
      "0x09240ea2601c34d792a0a5a8a84d8e501cfdfdf2c10ef13ea560acac58661882dd1b3644d1d4f3e32fc78498a7ebeffac8c6a494ac6f36923ef1be476444c0d564",
      "0x0912af3ac8f8ea443e6d89d071fccaa2b3c8462220c1c2921234f613b41594f08f2a170e61f5f436b536c155b438044cf0d0f24b94b4c034ad22b3eae824998243",
      "0x0916011d547d7a54929c3515078f4f672c6b390ccdd4119f0776376910bc5a38da1a059ed9c504fadcc9f77e8a402175743bee1f5be27b7002b0f6c5b51070452c",
      "0x09017285edc268d979eb410b46627e541afda16cdb3577ce04c15dc14cc6609c60143f0c01e71e99b2efbe3d8e62a2c812889aa9fd88dd4b0ed8eadcf1ec9b096a",
      "0x0922901e65200b007ad8e1b972e90403b336e459e0cf9b9d68732da345b1b0f6872c9e3f3edacbd857b26d0a66a80aa56c6ebaa9849e9ea5a2b17fd59cabe138e4",
      "0x091b77a00164a72880eec6c18fc043fa99f922e20bbee156e1ebfd3a358bee6bbb24d97cfaa234befe197a567476cade91b7d97a1017b8d5286dae4dddadffe1cd",
      "0x09216f1c4d67a9a428885bb8d978ad369d2d69d4dcc1692c3a0c3ea05da7d6f0ac2d6dda722e76eb513c67718e7be0478851758be5547322473a53b5b2b67faf95",
      "0x091f56c6f18ceb7077125df1ed17a42a85956090594125c1b182161de20f8af6aa2e36977412f9ea2ad2c0951153969eca8408317558ff1b6b4ad731726235f606",
      "0x092ca197dda6c519d80296f4fcda2933df9608ec684ad000133259024041d070812d29b058a998cf7ffc647b2739041725d77889f58953799c6aba6d9e5b981fc8",
      "0x091c25a87d321a09ad2a149d1a7eaa77727c7feffb4c39caf44f8edd4377f7bd0c16d1091494d3c90d301c1cb4596692798e78e4cc3d53c3a08e2641de43f9da18",
      "0x092166058c98245eb85b08da1c569df11f86b00cc44212a9a8ee0d60556d05a8030942c68b535651e11af38264ecc89e5f79b66c3d9ce87233ad65d4894a3d1c3d",
      "0x0908c3b13b7400630170baec7448c7ec99fa9100cad373e189e42aca121e2c8f450f9e40d92d98bb0b1286a18581591fddfa8637fc941c1630237293d69e5cb98f",
      "0x091362d251bbd8b255d63cd91bcfc257b8fb3ea608ce652784e3db11b22ca86c0122a0068fa1f1d54f313bed9fd9209212af3f366e4ff28092bf42c4abebffe10a",
      "0x081d67961bb431a9da78eb976fabd641e20fbf4b7e32eb3faac7dfb5abb50f1faf1438d77000c1cf96c9d61347e1351eb0200260ebe523e69f6e9f334ec86e6b58",
      "0x0819324d2488778bdef23319a6832001ee85f578cc920670c81f3645f898a46ec62e00385c4416ca4ccbab237b13396e5e25e5da12101021c6a6f9ecfe7c7fed19",
      "0x041421380c36ea8ef65a9bdb0202b06d1e03f52857cdfea3795463653eaa3dd7d80101000000000000000000000000000000000000000000000000000000006239b5a2c000208391082587ea494a8beba02cc40273f27e5477a967cd400736ac46950da0b378",
      "0x5448495320495320534f4d45204d4147494320425954455320464f5220534d54206d3172525867503278704449",
    ],
  },
  {
    // curl -H "content-type: application/json" -X POST --data '{"id":0,"jsonrpc":"2.0","method":"eth_getProof","params":["0x5300000000000000000000000000000000000004", ["0x0000000000000000000000000000000000000000000000000000000000000002"], "0x1111ad"]}' https://rpc.scroll.io
    block: 1118637,
    desc: "WETH.totalSupply",
    account: "0x5300000000000000000000000000000000000004",
    storage: "0x0000000000000000000000000000000000000000000000000000000000000002",
    expectedRoot: "0x1334a21a74914182745c1f5142e70b487262096784ae7669186657462c01b103",
    expectedValue: "0x0000000000000000000000000000000000000000000000600058d1a5ce14104d",
    accountProof: [
      "0x0907d980105678a2007eb5683d850f36a9caafe6e7fd3279987d7a94a13a360d3a1478f9a4c1f8c755227ee3544929bb0d7cfa2d999a48493d048ff0250bb002ab",
      "0x092b59a024f142555555c767842c4fcc3996686c57699791fcb10013f69ffd9b2507360087cb303767fd43f2650960621246a8d205d086e03d9c1626e4aaa5b143",
      "0x091f876342916ac1d5a14ef40cfc5644452170b16d1b045877f303cd52322ba1e00ba09f36443c2a63fbd7ff8feeb2c84e99fde6db08fd8e4c67ad061c482ff276",
      "0x09277b3069a4b944a45df222366aae727ec64efaf0a8ecb000645d0eea3a3fa93609b925158cc04f610f8c616369094683ca7a86239f49e97852aa286d148a3913",
      "0x092fb789200a7324067934da8be91c48f86c4e6f35fed6d1ce8ae4d7051f480bc0074019222c788b139b6919dfbc9d0b51f274e0ed3ea03553b8db30392ac05ce4",
      "0x092f79da8f9f2c3a3a3813580ff18d4619b95f54026b2f16ccbcca684d5e25e1f52912fa319d9a7ba537a52cc6571844b4d1aa99b8a78cea6f686a6279ade5dcae",
      "0x09249d249bcf92a369bd7715ec63a4b29d706a5dbb304efd678a2e5d7982e7fa9b202e3225c1031d83ab62d78516a4cbdbf2b22842c57182e7cb0dbb4303ac38c5",
      "0x0904837ebb85ceccab225d4d826fe57edca4b00862199b91082f65dfffa7669b90039c710273b02e60c2e74eb8b243721e852e0e56fa51668b6362fd920f817cb7",
      "0x090a36f6aabc3768a05dd8f93667a0eb2e5b63d94b5ce27132fb38d13c56d49cb4249c2013daee90184ae285226271f150f6a8f74f2c85dbd0721c5f583e620b10",
      "0x091b82f139a06af573e871fdd5f5ac18f17c568ffe1c9e271505b371ad7f0603e716b187804a49d2456a0baa7c2317c14d9aa7e58ad64df38bc6c1c7b86b072333",
      "0x0929668e59dfc2e2aef10194f5d287d8396e1a897d68f106bdb12b9541c0bab71d2bf910dea11e3209b3feff88d630af46006e402e935bc84c559694d88c117733",
      "0x0914231c92f09f56628c10603dc2d2120d9d11b27fa23753a14171127c3a1ee3dd0d6b9cbd11d031fe6e1b650023edc58aa580fa4f4aa1b30bf82e0e4c7a308bb9",
      "0x0914c1dd24c520d96aac93b7ef3062526067f1b15a080c482abf449d3c2cde781b195eb63b5e328572090319310914d81b2ca8350b6e15dc9d13e878f8c28c9d52",
      "0x0927cb93e3d9c144a5a3653c5cf2ed5940d64f461dd588cd192516ae7d855e9408166e85986d4c9836cd6cd822174ba9db9c7a043d73e86b5b2cfc0a2e082894c3",
      "0x090858bf8a0119626fe9339bd92116a070ba1a66423b0f7d3f4666b6851fdea01400f7f51eb22df168c41162d7f18f9d97155d87da523b05a1dde54e7a30a98c31",
      "0x0902776c1f5f93a95baea2e209ddb4a5e49dd1112a7f7d755a45addffe4a233dad0d8cc62b957d9b254fdc8199c720fcf8d5c65d14899911e991b4530710aca75e",
      "0x091d7fde5c78c88bbf6082a20a185cde96a203ea0d29c829c1ab9322fc3ca0ae3100ef7cba868cac216d365a0232ad6227ab1ef3290166bc6c19b719b79dbc17fc",
      "0x091690160269c53c6b74337a00d02cb40a88ea5eba06e1942088b619baee83279e12d96d62dda9c4b5897d58fea40b5825d87a5526dec37361ec7c93a3256ea76d",
      "0x091bccb091cde3f8ca7cfda1df379c9bfa412908c41037ae4ec0a20ce984e2c9a51d02c109d2e6e25dc60f10b1bc3b3f97ca1ce1aa025ce4f3146de3979403b99e",
      "0x0927083540af95e57acba69671a4a596f721432549b8760941f4251e0dd7a013a917cee0f60d333cf88e40ae8710fb1fd6e3920346a376b3ba6686a4b2020a043e",
      "0x082170b57b8f05f6990eec62e74cdb303741f6c464a85d68582c19c51e53f490000a5029a62ddc14c9c07c549db300bd308b6367454966c94b8526f4ceed5693b2",
      "0x0827a0b16ef333dcfe00610d19dc468b9e856f544c9b5e9b046357e0a38aedaeb90000000000000000000000000000000000000000000000000000000000000000",
      "0x06126f891e8753e67c5cbfa2a67e9d71942eab3a88cde86e97a4af94ea0dde497821fb69ccdb00e6eaeaf7fc1e73630f39f846970b72ac801e396da0033fb0c247",
      "0x0420e9fb498ff9c35246d527da24aa1710d2cc9b055ecf9a95a8a2a11d3d836cdf050800000000000000000000000000000000000000000000000016ef00000000000000000000000000000000000000000000000000000000000000600058d1a5ce14104d0dedcaecaab39b6e22c2608e40af67a71908e6e97bbf4a43c59c4537140c25a9e8c4073351c26b9831c1e5af153b9be4713a4af9edfdf32b58077b735e120f14136a7980da529d9e8d3a71433fc9dc5aa8c01e3a4eb60cb3a4f9cf9ca5c8e0be205300000000000000000000000000000000000004000000000000000000000000",
      "0x5448495320495320534f4d45204d4147494320425954455320464f5220534d54206d3172525867503278704449",
    ],
    storageProof: [
      "0x09240ea2601c34d792a0a5a8a84d8e501cfdfdf2c10ef13ea560acac58661882dd1b3644d1d4f3e32fc78498a7ebeffac8c6a494ac6f36923ef1be476444c0d564",
      "0x0912af3ac8f8ea443e6d89d071fccaa2b3c8462220c1c2921234f613b41594f08f2a170e61f5f436b536c155b438044cf0d0f24b94b4c034ad22b3eae824998243",
      "0x0916011d547d7a54929c3515078f4f672c6b390ccdd4119f0776376910bc5a38da1a059ed9c504fadcc9f77e8a402175743bee1f5be27b7002b0f6c5b51070452c",
      "0x092293af71b7b9315c32d08f06e291b85e3b3dbba786dd31952369f666281aa21125ab35feae70aaca9349f6af48f7dcf2dee0324e4eae03e929963e7728b633a3",
      "0x090607033a4b976c1e4683298d66b88a95ed45033ff43dea0670d84a8c42d35bf12562869385c0e70f561f18be4b78e7276b837f140a45ab12ffef1ba4ad5faecb",
      "0x090abc5f713c2f58583114bb5081d00cbd01789d8efbd95e471b151c71c475142f0f52ad30f8a63288eb9dd12aca2a670de08c03f8384f55d730c943e1c472625b",
      "0x0905156e8704d6195f6ae562aed2072f4e32422c6dfd4840ca354b9c4d2de5ce760fca52b1e0689ad374bae9fbea262a929f919695149a083fe6bacb806dc02fca",
      "0x0917078d4c193a3fdbfe8ce3a235a0e1df89e626b5e91636097e299883fc2447892ad46eefbb27909544fe02c05e29760315749f6ce21c17c52158f5f5616c2dad",
      "0x0917d02e5da8bdb969149c9327b247a6aaa479bcda4a03665da5103c10e616d2f40ccabdacdd25b34235d26e50e7af5d8d312a2cafdcadd41cc589a71a322f254c",
      "0x090c62f5c476c1def8ed8a8c25ae54581690b39dfab4b0f3f78b93df96f626714328ea922a76a058087563bb5370664e9a1cebe3062f2d904bf5e3a018219d6563",
      "0x091e481971f770e587b1f62f1da9ac4687abc5b2a23097fc38332e15ab957ca0ab0ec0a95c15313887e0d2f166c100deaf17f2ce50767680e6e5b2e3068801c0cd",
      "0x0911799e186f1bd299dfa08c07404b9d28e2b179fb6ad523f1846872537b6db85f198b573ac1397048258de38b391fcc5e0c86a0f81f4ca607785fb37041ab8b4d",
      "0x092053a028cf3bfcdabcb58985efc39f078cb0bcae4439528a0b6fe4b24bbdbd2c019a04a54e9e96077f3c2c39c1602a83387018b6357ea4c28e96764865d1c8f3",
      "0x07303fad3e4628ccae4de1adb41996c9f38b22445b6525ff163b4c68cbde275b1a06111cae9b4d17b730d94f589e20c6ae2cb59bf0b40ad05bf58703ee6d46eac4",
      "0x0606bc3fca1f1b3c877aa01a765c18db8b0d7f0bc50bd99f21223055bf1595c84d04fdc0fd416d8402fde743d908d032a20af6f2e65cdc6cc289f72c04f1c2476f",
      "0x04020953ad52de135367a1ba2629636216ed5174cce5629d11b5d97fe733f07dcc010100000000000000000000000000000000000000000000000000600058d1a5ce14104d200000000000000000000000000000000000000000000000000000000000000002",
      "0x5448495320495320534f4d45204d4147494320425954455320464f5220534d54206d3172525867503278704449",
    ],
  },
  {
    // curl -H "content-type: application/json" -X POST --data '{"id":0,"jsonrpc":"2.0","method":"eth_getProof","params":["0x5300000000000000000000000000000000000004", ["0x0000000000000000000000000000000000000000000000000000000000002222"], "0x1111ad"]}' https://rpc.scroll.io
    block: 1118637,
    desc: "random empty storage in WETH",
    account: "0x5300000000000000000000000000000000000004",
    storage: "0x0000000000000000000000000000000000000000000000000000000000002222",
    expectedRoot: "0x1334a21a74914182745c1f5142e70b487262096784ae7669186657462c01b103",
    expectedValue: "0x0000000000000000000000000000000000000000000000000000000000000000",
    accountProof: [
      "0x0907d980105678a2007eb5683d850f36a9caafe6e7fd3279987d7a94a13a360d3a1478f9a4c1f8c755227ee3544929bb0d7cfa2d999a48493d048ff0250bb002ab",
      "0x092b59a024f142555555c767842c4fcc3996686c57699791fcb10013f69ffd9b2507360087cb303767fd43f2650960621246a8d205d086e03d9c1626e4aaa5b143",
      "0x091f876342916ac1d5a14ef40cfc5644452170b16d1b045877f303cd52322ba1e00ba09f36443c2a63fbd7ff8feeb2c84e99fde6db08fd8e4c67ad061c482ff276",
      "0x09277b3069a4b944a45df222366aae727ec64efaf0a8ecb000645d0eea3a3fa93609b925158cc04f610f8c616369094683ca7a86239f49e97852aa286d148a3913",
      "0x092fb789200a7324067934da8be91c48f86c4e6f35fed6d1ce8ae4d7051f480bc0074019222c788b139b6919dfbc9d0b51f274e0ed3ea03553b8db30392ac05ce4",
      "0x092f79da8f9f2c3a3a3813580ff18d4619b95f54026b2f16ccbcca684d5e25e1f52912fa319d9a7ba537a52cc6571844b4d1aa99b8a78cea6f686a6279ade5dcae",
      "0x09249d249bcf92a369bd7715ec63a4b29d706a5dbb304efd678a2e5d7982e7fa9b202e3225c1031d83ab62d78516a4cbdbf2b22842c57182e7cb0dbb4303ac38c5",
      "0x0904837ebb85ceccab225d4d826fe57edca4b00862199b91082f65dfffa7669b90039c710273b02e60c2e74eb8b243721e852e0e56fa51668b6362fd920f817cb7",
      "0x090a36f6aabc3768a05dd8f93667a0eb2e5b63d94b5ce27132fb38d13c56d49cb4249c2013daee90184ae285226271f150f6a8f74f2c85dbd0721c5f583e620b10",
      "0x091b82f139a06af573e871fdd5f5ac18f17c568ffe1c9e271505b371ad7f0603e716b187804a49d2456a0baa7c2317c14d9aa7e58ad64df38bc6c1c7b86b072333",
      "0x0929668e59dfc2e2aef10194f5d287d8396e1a897d68f106bdb12b9541c0bab71d2bf910dea11e3209b3feff88d630af46006e402e935bc84c559694d88c117733",
      "0x0914231c92f09f56628c10603dc2d2120d9d11b27fa23753a14171127c3a1ee3dd0d6b9cbd11d031fe6e1b650023edc58aa580fa4f4aa1b30bf82e0e4c7a308bb9",
      "0x0914c1dd24c520d96aac93b7ef3062526067f1b15a080c482abf449d3c2cde781b195eb63b5e328572090319310914d81b2ca8350b6e15dc9d13e878f8c28c9d52",
      "0x0927cb93e3d9c144a5a3653c5cf2ed5940d64f461dd588cd192516ae7d855e9408166e85986d4c9836cd6cd822174ba9db9c7a043d73e86b5b2cfc0a2e082894c3",
      "0x090858bf8a0119626fe9339bd92116a070ba1a66423b0f7d3f4666b6851fdea01400f7f51eb22df168c41162d7f18f9d97155d87da523b05a1dde54e7a30a98c31",
      "0x0902776c1f5f93a95baea2e209ddb4a5e49dd1112a7f7d755a45addffe4a233dad0d8cc62b957d9b254fdc8199c720fcf8d5c65d14899911e991b4530710aca75e",
      "0x091d7fde5c78c88bbf6082a20a185cde96a203ea0d29c829c1ab9322fc3ca0ae3100ef7cba868cac216d365a0232ad6227ab1ef3290166bc6c19b719b79dbc17fc",
      "0x091690160269c53c6b74337a00d02cb40a88ea5eba06e1942088b619baee83279e12d96d62dda9c4b5897d58fea40b5825d87a5526dec37361ec7c93a3256ea76d",
      "0x091bccb091cde3f8ca7cfda1df379c9bfa412908c41037ae4ec0a20ce984e2c9a51d02c109d2e6e25dc60f10b1bc3b3f97ca1ce1aa025ce4f3146de3979403b99e",
      "0x0927083540af95e57acba69671a4a596f721432549b8760941f4251e0dd7a013a917cee0f60d333cf88e40ae8710fb1fd6e3920346a376b3ba6686a4b2020a043e",
      "0x082170b57b8f05f6990eec62e74cdb303741f6c464a85d68582c19c51e53f490000a5029a62ddc14c9c07c549db300bd308b6367454966c94b8526f4ceed5693b2",
      "0x0827a0b16ef333dcfe00610d19dc468b9e856f544c9b5e9b046357e0a38aedaeb90000000000000000000000000000000000000000000000000000000000000000",
      "0x06126f891e8753e67c5cbfa2a67e9d71942eab3a88cde86e97a4af94ea0dde497821fb69ccdb00e6eaeaf7fc1e73630f39f846970b72ac801e396da0033fb0c247",
      "0x0420e9fb498ff9c35246d527da24aa1710d2cc9b055ecf9a95a8a2a11d3d836cdf050800000000000000000000000000000000000000000000000016ef00000000000000000000000000000000000000000000000000000000000000600058d1a5ce14104d0dedcaecaab39b6e22c2608e40af67a71908e6e97bbf4a43c59c4537140c25a9e8c4073351c26b9831c1e5af153b9be4713a4af9edfdf32b58077b735e120f14136a7980da529d9e8d3a71433fc9dc5aa8c01e3a4eb60cb3a4f9cf9ca5c8e0be205300000000000000000000000000000000000004000000000000000000000000",
      "0x5448495320495320534f4d45204d4147494320425954455320464f5220534d54206d3172525867503278704449",
    ],
    storageProof: [
      "0x09240ea2601c34d792a0a5a8a84d8e501cfdfdf2c10ef13ea560acac58661882dd1b3644d1d4f3e32fc78498a7ebeffac8c6a494ac6f36923ef1be476444c0d564",
      "0x092fa31ba6c9b8f291512a582ab446daf7aa3787e68f9628d08ec0db329027d9001af83d361b481ed4b943d988cb0191c350b8efc85cfceba74afb60783488d441",
      "0x092c2ec2d967208cb5088400d826b52113d606435be011b6c9f721f293fb12242515681c9016eb1c222dcdbeeeb9fd3a504caba892f4c1832741a2b17a7305598a",
      "0x090c7fe825c29bf5df80c7101ff8a372ba4f7b2ac37c16a3bbda38cc1e38e682460499b7e5d21d3784f496e747140f465eb1a39a019d2be8baf13a5e39f359a4ed",
      "0x092bb11ebbc7cd1e565b86498aecab16842ab3fa852c7943cfbc49ee4bc593b2f308a78e1bc555e07d36d5c812af57c18f67199197a52ff74bc4e32ca6b7fadf32",
      "0x092fd1e042080801034c6d6c79d462016c74b97dfbb1272cf606e638911a08f21c02434541eeed6d66002c69042f9354211e40518316a2d98cc0da0f19fb1ea013",
      "0x09024bd491ec707bc3e8bea6b2754f37b1e85903061aefabd945537eef2f4d38b4136b925b004d29603c5e6195e073322d27f0c6ea3fa1ea5c5b248ff60dda594c",
      "0x09269e1f468bd9bbde77a13562645a80a77d26d801781ca95d385bd59ee1b0890b03694bf9043190620265bf0bc3baa4d82cc82302ae0bbf33cfa48b0ec9d5ab25",
      "0x0924d8bf62b2a725684847208dc021d5aee9f3c8f14c14786bc9f93232dfd3e068120bb7d022bbb159b4b84bb9e36cd2fcd89d761e265c1b88c8bdb9745a51cb22",
      "0x092680f932920fd86de0b417cfdbeb2836a470213097ed5abb1a2b4deba8437f6825fd0ec614b97e6cfa4d50b08ad1e0fd8a5cd72db3a468128d1045d6a54e5e6e",
      "0x0909e630914cee4db538057a0218a72288b88b2603aee0f805254b865a03de87c92ce46c1aa77ee8c42bb60c4175826f4dbb89d6282c01ff3de654c961599e66c3",
      "0x091a17302d53ad1b7a4472d111fd27b35720d49ce27259b5e42f46339dddf235e82b973c29f44cf69b589f724d7d2fa54bf38b37bde3fc66c0d965a8c10df80caa",
      "0x0916572156ae22ae2b0bc84ff41d16668be7163da26db2b13b86c218e0516c97a4131b584b7192464dde26060f66f678b03c8db8f64f1cd7a1f98a22a90cce5850",
      "0x092c6ee2ca598c123445bbbd403ca3ab8a95ce2443f941ebdcf7bb035e2a3e38e22e8d5b222a1019b126f0ecf277c7fed881413e879cd4dc5df66634b6e9fb688d",
      "0x0700000000000000000000000000000000000000000000000000000000000000002822301c27c0bd26a8f361545a09d509a2feed981accf780de30244f0300321d",
      "0x05",
      "0x5448495320495320534f4d45204d4147494320425954455320464f5220534d54206d3172525867503278704449",
    ],
  },
  {
    // curl -H "content-type: application/json" -X POST --data '{"id":0,"jsonrpc":"2.0","method":"eth_getProof","params":["0x5300000000000000000000000000000000000044", ["0x0000000000000000000000000000000000000000000000000000000000000000"], "0x1111ad"]}' https://rpc.scroll.io
    block: 1154766,
    desc: "random empty storage in some contract",
    account: "0x226D078166C78e00ce5E97d8f18CDc408512bb0F",
    storage: "0x0000000000000000000000000000000000000000000000000000000000000001",
    expectedRoot: "0x1e5cf13822e052084c315e944ca84f1ef375583e85e1508055123a182e415fab",
    expectedValue: "0x0000000000000000000000000000000000000000000000000000000000000000",
    accountProof: [
      "0x09062c633f6d7c7a157025fef8ab1c313a7caadda3a64b23664741f9de3b0478fe27571cf9b45d5f4deddf5f0b5354a613998fdcbe9249bb7cde92fd45513c5a99",
      "0x0920d6877efe14060018278754e91682430401880981fec1cd1b63610bed0c1e332a63aca7a8898b01983e2c53a7257310318da444fd6c8b705e488943205301a8",
      "0x090f6dadd53bbc0f5fa4fa03961aff0bf252ae335e11c1836253b6bc214d66759010b10d80991219a66f1eb7e07169b4cec4fa74b04edbdc08c3f238dfdf1d2fac",
      "0x0921ea10af71b5f3587ff9d42178a151427cbcde37b8bee6575463bf6b83110cca0520d5f97b44e7015453ec16d9c28980d2cec3df5c860eb8a455f49dcfa339be",
      "0x092d19cf96a7c129aac6f72f780703a9ef3233fc5124d592baee751a3550dd692a02c962b87efbba5aeea4856c3df29c1ea540e1fbc7a74529d5dc793fe8e490d8",
      "0x0922e20a087e600560007189ccc1a159e4fffeb1876a6de3772b7f450793a1c6620ada74791f3ecd25a650701578ef9661c64e75d836c681503e96228974a53903",
      "0x0924839671b636ebb56cb9a2860a3edf2a2875774e84dfcf8546135189f808d724260ac8be541ff088a9a1d2468c4c6e2faa793009be553a3cbca003649ee511db",
      "0x090cd8140d844f62e44ffe820c1b2b0d4aa8f0518c15ff61759d93d805cb017cb628d5b46a4c4ec0a10eb00155808890925050f7af2279b512c25005d963283262",
      "0x0913c0698673b0be011485eba05c61ac41bf14fc960ce5dbb6f5a021809eabbb0e18adaf85a3724e1a644268b845f5014b39e574928b9a01bfcd25d6fe1cf03e8f",
      "0x0912c2e7da4b091c52e0012e5c13baf07d9d9daed10a558262d2e700a7c823300e054dce1849561bbeede4368a3be06f5a2bae06bdb1bc2bcefdba84634fd1991c",
      "0x090b3e9c665497a0f9c1d3f1448c6d9144a287eb0accf86fea6f443f51986df7130392814f078a19643081787478ec3a010e2757a574877a194136c529813cf7ae",
      "0x09249a0e273abe79a0b99a55516e19213191b7f77ef34f8815edc4e1ede8711f7920615adbac1983d844c8a6ed50922562432c13d030069d8b3e92611b4fe39531",
      "0x09199575893e55d92fafb3b067130b9b6b5a46e7f6fb2d0af412d12591632dfe961adffb9dd1e7490095aac94bc1fcaeb591f4ba907fe2b882c9f6d8f7ab3a1809",
      "0x09259308e9398f029ebbe31a4b353f474622b4c96995b7365c3b13c392fcc3e7001be60286a497a3886aa9cff3ad6a5dc71504078eb7a44c43530b7b33eef4743f",
      "0x090709a21aaf18a1eaea3b925ab36f47a82095aa3e9ddbc4f01463005c4b64f6af0554d854637fcbfd9b1a4c2474de343950569e4f855d66f2ee14fcfb19ee17f5",
      "0x092d7319be75a70b8ea5f0acc6ab4a96971ec546f72b18bdc3e905ad6ea8a288f70626499aee389335559b1dd3cc8b6711f9fde0c517236190cba24fa87993877a",
      "0x09081b165a51e3081fc2e3e27d6fdb81134b65284851798de62899db3065a8c1fc040c8dce92508a510c2c34fc2949910dd41247c9f247cd216c03d9bb9d2881b4",
      "0x092a27c5be32e1ab6e85d1ac094bc1509d92285f45c63fca6dba9b14d485a94af326d44c1ff85666a4790182ddd7e51cbbe06af81d62082e6d79faec29a4501369",
      "0x091a46df6ffd6b439ffcd1b57e9548f5c4db26ade9e984efc8a91a01ab22134d3c1617b504ac2015793c5dac16d379b5ca6cb70c14243491bb68535ee686a3a553",
      "0x08180e90f9f9a4fd8065a5849539793bd9e9340b69770eff1716a733241e454c341641f913f1c32e2c652b876f902e5c2c8d51c482411ec44dae969bdc50264c42",
      "0x06273c162ecb059cd86ec0a01033dd61c39f59ee0a13eb41a28c0b2d49a45f6f94081be344adea9f54587a832b9efef6fc9ec010d86ec5fb2b53b5ff8dbabc4924",
      "0x040b792f5b15327fc37390341af919c991641846d380397e4c73cbb1298921a546050800000000000000000000000000000000000000000000000000fb0000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000be74cc05824041ef286fd08582cdfacec7784a35af72f937acf64ade5073da10889249d61c3649abf8749bf686a73f708d67726fada3e071b03d4541da9156b20226d078166c78e00ce5e97d8f18cdc408512bb0f000000000000000000000000",
      "0x5448495320495320534f4d45204d4147494320425954455320464f5220534d54206d3172525867503278704449",
    ],
    storageProof: [
      "0x05",
      "0x5448495320495320534f4d45204d4147494320425954455320464f5220534d54206d3172525867503278704449",
    ],
  },
];

describe("ZkTrieVerifier", async () => {
  let verifier: MockZkTrieVerifier;

  beforeEach(async () => {
    const [deployer] = await ethers.getSigners();

    const PoseidonHashWithDomainFactory = new ethers.ContractFactory(generateABI(2), createCode(2), deployer);

    const poseidon = await PoseidonHashWithDomainFactory.deploy();
    await poseidon.deployed();

    const MockZkTrieVerifier = await ethers.getContractFactory("MockZkTrieVerifier", deployer);
    verifier = await MockZkTrieVerifier.deploy(poseidon.address);
    await verifier.deployed();
  });

  const shouldRevert = async (test: ITestConfig, reason: string, extra?: string) => {
    const proof = concat([
      `0x${test.accountProof.length.toString(16).padStart(2, "0")}`,
      ...test.accountProof,
      `0x${test.storageProof.length.toString(16).padStart(2, "0")}`,
      ...test.storageProof,
      extra || "0x",
    ]);
    await expect(verifier.verifyZkTrieProof(test.account, test.storage, proof)).to.revertedWith(reason);
  };

  for (const test of testcases) {
    it(`should succeed for block[${test.block}] desc[${test.desc}] account[${test.account}] storage[${test.storage}]`, async () => {
      const proof = concat([
        `0x${test.accountProof.length.toString(16).padStart(2, "0")}`,
        ...test.accountProof,
        `0x${test.storageProof.length.toString(16).padStart(2, "0")}`,
        ...test.storageProof,
      ]);
      const [root, value, gasUsed] = await verifier.verifyZkTrieProof(test.account, test.storage, proof);
      expect(test.expectedRoot).to.eq(root);
      expect(test.expectedValue).to.eq(value);
      console.log("gas usage:", gasUsed.toString());
    });
  }

  it("should revert, when InvalidBranchNodeType", async () => {
    const test = testcases[0];
    for (const i of [0, 1, test.accountProof.length - 3]) {
      const correct = test.accountProof[i];
      const prefix = correct.slice(0, 4);
      for (let b = 0; b < 16; ++b) {
        if (b >= 6 && b < 10) continue;
        test.accountProof[i] = test.accountProof[i].replace(prefix, "0x" + chars[b >> 4] + chars[b % 16]);
        await shouldRevert(test, "InvalidBranchNodeType");
        test.accountProof[i] = correct;
      }
    }

    for (const i of [0, 1, test.storageProof.length - 3]) {
      const correct = test.storageProof[i];
      const prefix = correct.slice(0, 4);
      for (let b = 0; b < 16; ++b) {
        if (b >= 6 && b < 10) continue;
        test.storageProof[i] = test.storageProof[i].replace(prefix, "0x" + chars[b >> 4] + chars[b % 16]);
        await shouldRevert(test, "InvalidBranchNodeType");
        test.storageProof[i] = correct;
      }
    }
  });

  it("should revert, when BranchHashMismatch", async () => {
    const test = testcases[0];
    for (const i of [1, 2, test.accountProof.length - 3]) {
      const correct = test.accountProof[i];
      for (const p of [40, 98]) {
        const v = correct[p];
        for (let b = 0; b < 3; ++b) {
          if (v === chars[b]) continue;
          test.accountProof[i] = correct.slice(0, p) + chars[b] + correct.slice(p + 1);
          await shouldRevert(test, "BranchHashMismatch");
          test.accountProof[i] = correct;
        }
      }
    }

    for (const i of [1, 2, test.storageProof.length - 3]) {
      const correct = test.storageProof[i];
      for (const p of [40, 98]) {
        const v = correct[p];
        for (let b = 0; b < 3; ++b) {
          if (v === chars[b]) continue;
          test.storageProof[i] = correct.slice(0, p) + chars[b] + correct.slice(p + 1);
          await shouldRevert(test, "BranchHashMismatch");
          test.storageProof[i] = correct;
        }
      }
    }
  });

  it("should revert, when InvalidAccountLeafNodeType", async () => {
    const test = testcases[0];
    const index = test.accountProof.length - 2;
    const correct = test.accountProof[index];
    const prefix = correct.slice(0, 4);
    for (let b = 0; b < 20; ++b) {
      if (b === 4 || b === 5) continue;
      test.accountProof[index] = test.accountProof[index].replace(prefix, "0x" + chars[b >> 4] + chars[b % 16]);
      await shouldRevert(test, "InvalidAccountLeafNodeType");
      test.accountProof[index] = correct;
    }
  });

  it("should revert, when AccountKeyMismatch", async () => {
    const test = testcases[0];
    const index = test.accountProof.length - 2;
    const correct = test.accountProof[index];
    for (const p of [4, 10]) {
      const v = correct[p];
      for (let b = 0; b < 3; ++b) {
        if (v === chars[b]) continue;
        test.accountProof[index] = correct.slice(0, p) + chars[b] + correct.slice(p + 1);
        await shouldRevert(test, "AccountKeyMismatch");
        test.accountProof[index] = correct;
      }
    }
  });

  it("should revert, when InvalidAccountCompressedFlag", async () => {
    const test = testcases[0];
    const index = test.accountProof.length - 2;
    const correct = test.accountProof[index];
    for (const replaced of ["01080000", "05010000"]) {
      test.accountProof[index] = test.accountProof[index].replace("05080000", replaced);
      await shouldRevert(test, "InvalidAccountCompressedFlag");
      test.accountProof[index] = correct;
    }
  });

  it("should revert, when InvalidAccountLeafNodeHash", async () => {
    const test = testcases[0];
    const index = test.accountProof.length - 2;
    const correct = test.accountProof[index];
    for (const p of [80, 112, 144, 176, 208]) {
      const v = correct[p];
      for (let b = 0; b < 3; ++b) {
        if (v === chars[b]) continue;
        test.accountProof[index] = correct.slice(0, p) + chars[b] + correct.slice(p + 1);
        await shouldRevert(test, "InvalidAccountLeafNodeHash");
        test.accountProof[index] = correct;
      }
    }
  });

  it("should revert, when InvalidAccountKeyPreimageLength", async () => {
    const test = testcases[0];
    const index = test.accountProof.length - 2;
    const correct = test.accountProof[index];
    for (const p of [396, 397]) {
      const v = correct[p];
      for (let b = 0; b < 3; ++b) {
        if (v === chars[b]) continue;
        test.accountProof[index] = correct.slice(0, p) + chars[b] + correct.slice(p + 1);
        await shouldRevert(test, "InvalidAccountKeyPreimageLength");
        test.accountProof[index] = correct;
      }
    }
  });

  it("should revert, when InvalidAccountKeyPreimage", async () => {
    const test = testcases[0];
    const index = test.accountProof.length - 2;
    const correct = test.accountProof[index];
    for (const p of [398, 438]) {
      const v = correct[p];
      for (let b = 0; b < 3; ++b) {
        if (v === chars[b]) continue;
        test.accountProof[index] = correct.slice(0, p) + chars[b] + correct.slice(p + 1);
        await shouldRevert(test, "InvalidAccountKeyPreimage");
        test.accountProof[index] = correct;
      }
    }
  });

  it("should revert, when InvalidProofMagicBytes", async () => {
    const test = testcases[0];
    let index = test.accountProof.length - 1;
    let correct = test.accountProof[index];
    for (const p of [2, 32, 91]) {
      const v = correct[p];
      for (let b = 0; b < 3; ++b) {
        if (v === chars[b]) continue;
        test.accountProof[index] = correct.slice(0, p) + chars[b] + correct.slice(p + 1);
        await shouldRevert(test, "InvalidProofMagicBytes");
        test.accountProof[index] = correct;
      }
    }

    index = test.storageProof.length - 1;
    correct = test.storageProof[index];
    for (const p of [2, 32, 91]) {
      const v = correct[p];
      for (let b = 0; b < 3; ++b) {
        if (v === chars[b]) continue;
        test.storageProof[index] = correct.slice(0, p) + chars[b] + correct.slice(p + 1);
        await shouldRevert(test, "InvalidProofMagicBytes");
        test.storageProof[index] = correct;
      }
    }
  });

  it("should revert, when InvalidAccountLeafNodeHash", async () => {
    const test = testcases[0];
    const correct = test.storageProof.slice();
    test.storageProof = [
      "0x05",
      "0x5448495320495320534f4d45204d4147494320425954455320464f5220534d54206d3172525867503278704449",
    ];
    await shouldRevert(test, "InvalidAccountLeafNodeHash");
    test.storageProof = correct;
  });

  it("should revert, when InvalidStorageLeafNodeType", async () => {
    const test = testcases[0];
    const index = test.storageProof.length - 2;
    const correct = test.storageProof[index];
    const prefix = correct.slice(0, 4);
    for (let b = 0; b < 20; ++b) {
      if (b === 4 || b === 5) continue;
      test.storageProof[index] = test.storageProof[index].replace(prefix, "0x" + chars[b >> 4] + chars[b % 16]);
      await shouldRevert(test, "InvalidStorageLeafNodeType");
      test.storageProof[index] = correct;
    }
  });

  it("should revert, when StorageKeyMismatch", async () => {
    const test = testcases[0];
    const index = test.storageProof.length - 2;
    const correct = test.storageProof[index];
    for (const p of [4, 10]) {
      const v = correct[p];
      for (let b = 0; b < 3; ++b) {
        if (v === chars[b]) continue;
        test.storageProof[index] = correct.slice(0, p) + chars[b] + correct.slice(p + 1);
        await shouldRevert(test, "StorageKeyMismatch");
        test.storageProof[index] = correct;
      }
    }
  });

  it("should revert, when InvalidStorageCompressedFlag", async () => {
    const test = testcases[0];
    const index = test.storageProof.length - 2;
    const correct = test.storageProof[index];
    for (const replaced of ["00010000", "01000000"]) {
      test.storageProof[index] = test.storageProof[index].replace("01010000", replaced);
      await shouldRevert(test, "InvalidStorageCompressedFlag");
      test.storageProof[index] = correct;
    }
  });

  it("should revert, when InvalidStorageLeafNodeHash", async () => {
    const test = testcases[0];
    const index = test.storageProof.length - 2;
    const correct = test.storageProof[index];
    for (const p of [100, 132]) {
      const v = correct[p];
      for (let b = 0; b < 3; ++b) {
        if (v === chars[b]) continue;
        test.storageProof[index] = correct.slice(0, p) + chars[b] + correct.slice(p + 1);
        await shouldRevert(test, "InvalidStorageLeafNodeHash");
        test.storageProof[index] = correct;
      }
    }
  });

  it("should revert, when InvalidStorageKeyPreimageLength", async () => {
    const test = testcases[0];
    const index = test.storageProof.length - 2;
    const correct = test.storageProof[index];
    for (const p of [140, 141]) {
      const v = correct[p];
      for (let b = 0; b < 3; ++b) {
        if (v === chars[b]) continue;
        test.storageProof[index] = correct.slice(0, p) + chars[b] + correct.slice(p + 1);
        await shouldRevert(test, "InvalidStorageKeyPreimageLength");
        test.storageProof[index] = correct;
      }
    }
  });

  it("should revert, when InvalidStorageKeyPreimage", async () => {
    const test = testcases[0];
    const index = test.storageProof.length - 2;
    const correct = test.storageProof[index];
    for (const p of [142, 205]) {
      const v = correct[p];
      for (let b = 0; b < 3; ++b) {
        if (v === chars[b]) continue;
        test.storageProof[index] = correct.slice(0, p) + chars[b] + correct.slice(p + 1);
        await shouldRevert(test, "InvalidStorageKeyPreimage");
        test.storageProof[index] = correct;
      }
    }
  });

  it("should revert, when InvalidStorageEmptyLeafNodeHash", async () => {
    const test = testcases[0];
    const index = test.storageProof.length - 2;
    const correct = test.storageProof[index];
    test.storageProof[index] = "0x05";
    await shouldRevert(test, "InvalidStorageEmptyLeafNodeHash");
    test.storageProof[index] = correct;
  });

  it("should revert, when ProofLengthMismatch", async () => {
    const test = testcases[0];
    await shouldRevert(test, "ProofLengthMismatch", "0x0000");
  });
});
