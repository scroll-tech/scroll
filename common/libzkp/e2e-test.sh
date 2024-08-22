set -xeu
set -o pipefail

export CHAIN_ID=534352
export RUST_BACKTRACE=full
export RUST_LOG=debug
export RUST_MIN_STACK=100000000
export PROVER_OUTPUT_DIR=test_zkp_test
export SCROLL_PROVER_ASSETS_DIR=/assets/test_assets
export DARWIN_V2_TEST_DIR=/assets
#export LD_LIBRARY_PATH=/:/usr/local/cuda/lib64

mkdir -p $PROVER_OUTPUT_DIR

REPO=$(realpath ../..)

function build_test_bins() {
    cd $REPO/prover
    make tests_binary
    cd $REPO/coordinator
    make libzkp
    go test -tags="gpu ffi" -timeout 0 -c ./internal/logic/verifier
    cd $REPO/common/libzkp
}

build_test_bins
rm -rf $PROVER_OUTPUT_DIR/*
#rm -rf prover.log verifier.log
$REPO/prover/prover.test --exact zk_circuits_handler::darwin_v2::tests::test_circuits 2>&1 | tee prover.log
$REPO/coordinator/verifier.test -test.v 2>&1 | tee verifier.log
