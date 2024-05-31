set -xeu
set -o pipefail

export CHAIN_ID=534352
export RUST_BACKTRACE=full
export RUST_LOG=debug
export RUST_MIN_STACK=100000000
export PROVER_OUTPUT_DIR=test_zkp_test
#export LD_LIBRARY_PATH=/:/usr/local/cuda/lib64

mkdir -p $PROVER_OUTPUT_DIR

REPO=$(realpath ../..)

function build_test_bins() {
    cd impl
    cargo build --release
    ln -f -s $(realpath target/release/libzkp.so) $REPO/prover/core/lib
    ln -f -s $(realpath target/release/libzkp.so) $REPO/coordinator/internal/logic/verifier/lib
    cd $REPO/prover
    go test -tags="gpu ffi" -timeout 0 -c core/prover_test.go
    cd $REPO/coordinator
    go test -tags="gpu ffi" -timeout 0 -c ./internal/logic/verifier
    cd $REPO/common/libzkp
}

function build_test_bins_old() {
    cd $REPO
    cd prover
    make libzkp
    go test -tags="gpu ffi" -timeout 0 -c core/prover_test.go
    cd ..
    cd coordinator
    make libzkp
    go test -tags="gpu ffi" -timeout 0 -c ./internal/logic/verifier
    cd ..
    cd common/libzkp
}

build_test_bins
#rm -rf test_zkp_test/*
#rm -rf prover.log verifier.log
#$REPO/prover/core.test -test.v 2>&1 | tee prover.log
$REPO/coordinator/verifier.test -test.v 2>&1 | tee verifier.log
