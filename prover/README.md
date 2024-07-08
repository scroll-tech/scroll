## How-to run Snarkify prover locally
1. `make snarkify`
2. update `config.json` with right coordinator and l2geth endpoints
3. download parameters and assets to the referred directory in `config.json`, 
following step3 & 4 in https://www.notion.so/scrollzkp/How-to-run-a-batch-prover-locally-43fee225911f4375a61a232cdea1e546
4. setting up envrionment variable
```shell
export LD_LIBRARY_PATH=$(pwd)/lib:$LD_LIBRARY_PATH
export SCROLL_PROVER_ASSETS_DIR=assets,assets
export RUST_MIN_STACK=100000000
export CHAIN_ID=534351 #update here, 534351 for sepolia, and 534352 for mainnet
```
5. run `./target/release/snarkify`
6. In a different shell, run `run_batch.sh` to submit a sample job to the prover to generate proof
7. 
## How-to run Snarkify prover docker
under the scroll/prover directory, run
1. `make snarkify`
2. `docker build -t scroll-prover-gpu:latest .`
3. `docker run -v /home/ubuntu/scroll/volume:/snarkify-data  -p 8080:8080 scroll-prover-gpu`
4. In a different shell, run `run_batch.sh` to submit a sample job to the prover to generate proof
