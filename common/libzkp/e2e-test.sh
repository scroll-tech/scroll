set -xeu
export CHAIN_ID=534352
export RUST_BACKTRACE=full
export RUST_LOG=debug
export RUST_MIN_STACK=100000000
export PROVER_OUTPUT_DIR=test_zkp_test
#export LD_LIBRARY_PATH=/:/usr/local/cuda/lib64

REPO="../.."

function build_test_bins() {
    cd $REPO
    cd prover
    make libzkp
    go test -tags="gpu ffi" -timeout 0 -c core/prover_test.go
    cd ..
    cd coordinator
    go test -tags="gpu ffi" -timeout 0 -c ./internal/logic/verifier
    cd ..
    cd common/libzkp
}

$REPO/prover/core.test -test.v
$REPO/coordinator/verifier.test -test.v
