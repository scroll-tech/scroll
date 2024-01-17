module.exports = {
  skipFiles: [
    'mocks',
    'test',
    'L2/predeploys/L1BlockContainer.sol',
    'libraries/verifier/ZkTrieVerifier.sol',
    'libraries/verifier/PatriciaMerkleTrieVerifier.sol'
  ],
  istanbulReporter: ["lcov", "json"]
};
