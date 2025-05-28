# Permissionless Batches
Permissionless batches aka enforced batches is a feature that provides guarantee to users that they can exit Scroll even if the operator is down or censoring. 
It allows anyone to take over and submit a batch (permissionless batch submission) together with a proof after a certain time period has passed without a batch being finalized on L1. 

Once permissionless batch mode is activated, the operator can no longer submit batches in a permissioned way. Only the security council can deactivate permissionless batch mode and reinstate the operator as the only batch submitter.
There are two types of situations to consider:
- `Permissionless batch mode is activated:` This means that finalization halted for some time. Now anyone can submit batches utilizing the [batch production toolkit](#batch-production-toolkit).
- `Permissionless batch mode is deactivated:` This means that the security council has decided to reinstate the operator as the only batch submitter. The operator needs to [recover](#operator-recovery) the sequencer and relayer to resume batch submission and the valid L2 chain.


## Batch production toolkit
The batch production toolkit is a set of tools that allow anyone to submit a batch in permissionless mode. It consists of three main components:
1. l2geth state recovery from L1
2. l2geth block production
3. production, proving and submission of batch with `docker-compose.yml`

### Prerequisites
- Unix-like OS, 32GB RAM
- Docker
- [l2geth](https://github.com/scroll-tech/go-ethereum/) or [Docker image](https://hub.docker.com/r/scrolltech/l2geth) of corresponding version [TODO link list with versions](#batch-production-toolkit).
- access to an Ethereum L1 RPC node (beacon node and execution client)
- ability to run a prover or access to a proving service (e.g. Sindri)
- L1 account with funds to pay for the batch submission

### 1. l2geth state recovery from L1
Once permissionless mode is activated there's no blocks being produced and propagated on L2. The first step is to recover the latest state of the L2 chain from L1. This is done by running l2geth in recovery mode. 
More information about l2geth recovery (aka L1 follower mode) can be found [here TODO: put correct link once released](https://github.com/scroll-tech/scroll-documentation/pull/374).

Running l2geth in recovery mode requires following configuration:
- `--scroll` or `--scroll-sepolia` - enables Scroll Mainnet or Sepolia mode
- `--da.blob.beaconnode` - L1 RPC beacon node
- `--l1.endpoint` - L1 RPC execution client
- `--da.sync=true` - enables syncing with L1
- `--da.recovery` - enables recovery mode
- `--da.recovery.initiall1block` - initial L1 block (commit tx of initial batch)
- `--da.recovery.initialbatch` - batch where to start recovery from. Can be found on [Scrollscan Explorer](https://scrollscan.com/batches).
- `--da.recovery.l2endblock` - until which L2 block recovery should run (optional)

```bash
./build/bin/geth --scroll<-sepolia> \
--datadir "tmp/datadir" \
--gcmode archive \
--http --http.addr "0.0.0.0" --http.port 8545 --http.api "eth,net,web3,debug,scroll" --http.vhosts "*" \
--da.blob.beaconnode "<L1 RPC beacon node>" \
--l1.endpoint "<L1 RPC execution client>" \
--da.sync=true --da.recovery --da.recovery.initiall1block "<initial L1 block (commit tx of initial batch)>" --da.recovery.initialbatch "<batch where to start recovery from>" --da.recovery.l2endblock "<until which L2 block recovery should run (optional)>" \
--verbosity 3
```

### 2. l2geth block production
After the state is recovered, the next step is to produce blocks on L2. This is done by running l2geth in block production mode.
As a prerequisite, the state recovery must be completed and the latest state of the L2 chain must be available.

You also need to generate a keystore e.g. with [Clef](https://geth.ethereum.org/docs/fundamentals/account-management) to be able to sign blocks. 
This key is not used for any funds, but required for block production to work. Once you generated blocks you can safely discard it.

Running l2geth in block production mode requires following configuration:
- `--scroll` or `--scroll-sepolia` - enables Scroll Mainnet or Sepolia mode
- `--da.blob.beaconnode` - L1 RPC beacon node
- `--l1.endpoint` - L1 RPC execution client
- `--da.sync=true` - enables syncing with L1
- `--da.recovery` - enables recovery mode
- `--da.recovery.produceblocks` - enables block production
- `--miner.etherbase '0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee' --mine` - enables mining. the address is not used, but required for mining to work
- `---miner.gaslimit 1 --miner.gasprice 1 --miner.maxaccountsnum 100 --rpc.gascap 0 --gpo.ignoreprice 1` - gas limits for block production

```bash
./build/bin/geth --scroll<-sepolia> \
--datadir "tmp/datadir" \
--gcmode archive \
--http --http.addr "0.0.0.0" --http.port 8545 --http.api "eth,net,web3,debug,scroll" --http.vhosts "*" \
--da.blob.beaconnode "<L1 RPC beacon node>" \
--l1.endpoint "<L1 RPC execution client>" \
--da.sync=true --da.recovery --da.recovery.produceblocks \
--miner.gaslimit 1 --miner.gasprice 1 --miner.maxaccountsnum 100 --rpc.gascap 0 --gpo.ignoreprice 1 \
--miner.etherbase '0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee' --mine \
--ccc \
--verbosity 3
```

### 3. production, proving and submission of batch with `docker-compose.yml`
After the blocks are produced, the next step is to produce a batch, prove it and submit it to L1. This is done by running the `docker-compose.yml` in the `permissionless-batches` folder.


#### Producing a batch
To produce a batch you need to run the `batch-production-submission` profile in `docker-compose.yml`.

1. Fill `conf/genesis.json` with the latest genesis state from the L2 chain. The genesis for the current fork can be found here: [TODO link list with versions](#batch-production-toolkit)
2. Make sure that `l2geth` with your locally produced blocks is running and reachable from the Docker network (e.g. `http://host.docker.internal:8545`)
3. Fill in required fields in `conf/relayer/config.json`


Run with `make batch_production_submission`.
This will produce chunks, a batch and bundle which will be proven in the next step.
`Success! You're ready to generate proofs!` indicates that everything is working correctly and the batch is ready to be proven.

#### Proving a batch
To prove the chunk, batch and bundle you just generated you need to run the `local-prover` or `cloud-prover` profile in `docker-compose.yml`.

Local Proving:

1. Hardware spec for local prover: CPU: 36+ core, 128G memory GPU: 24G memory (eg. Rtx 3090/3090Ti/4090/A10/L4)
2. Make sure `verifier` `low_version_circuit` and `high_version_circuit` in `conf/coordinator/config.json` are correct for the latest fork: [TODO link list with versions](#batch-production-toolkit)
2. Set the `SCROLL_ZKVM_VERSION` environment variable on `Makefile` to the correct version. [TODO link list with versions](#batch-production-toolkit)
4. Fill in the required fields in `conf/proving-service/local-prover/config.json`

Run with `make local_prover`.

Cloud Proving(not supported yet):

1. Make sure `verifier` `low_version_circuit` and `high_version_circuit` in `conf/coordinator/config.json` are correct for the latest fork: [TODO link list with versions](#batch-production-toolkit)
2. Set the `SCROLL_ZKVM_VERSION` environment variable on `Makefile` to the correct version. [TODO link list with versions](#batch-production-toolkit)
3. Fill in the required fields in `conf/proving-service/cloud-prover/config.json`. It is recommended to use Sindri. You'll need to obtain credits and an API key from their [website](https://sindri.app/).

Run with `make cloud_prover`.

This will prove chunks, the batch and bundle.
Run `make check_proving_status`
`Success! You're ready to submit permissionless batch and proof!` indicates that everything is working correctly and the batch is ready to be submit.


#### Batch submission
To submit the batch you need to run the `batch-production-submission` profile in `docker-compose.yml`.

1. Fill in required fields in `conf/relayer/config.json` for the sender config.

Run with `make batch_production_submission`.
This will submit the batch to L1 and finalize it. The transaction will be retried in case of failure.

**Troubleshooting**
- in case the submission fails it will print the calldata for the transaction in an error message. You can use this with `cast call --trace --rpc-url "$SCROLL_L1_DEPLOYMENT_RPC" "$L1_SCROLL_CHAIN_PROXY_ADDR" <calldata>` to see what went wrong.
  - `0x4df567b9: ErrorNotInEnforcedBatchMode`: permissionless batch mode is not activated, you can't submit a batch
  - `0xa5d305cc: ErrorBatchIsEmpty`: no blob was provided. This is usually returned if you do the `cast call`, permissionless mode is activated but you didn't provide a blob in the transaction.

## Operator recovery
Operator recovery needs to be run by the rollup operator to resume normal rollup operation after permissionless batch mode is deactivated. It consists of two main components:
1. l2geth recovery
2. Relayer recovery

These steps are required to resume permissioned batch submission and the valid L2 chain. They will restore the entire history of the batches submitted during permissionless mode. 

### Prerequisites
- l2geth with the latest state of the L2 chain (before permissionless mode was activated)
- signer key for the sequencer according to Clique consensus
- relayer and coordinator are set up, running and up-to-date with the latest state of the L2 chain (before permissionless mode was activated)

### l2geth recovery
Running l2geth in recovery mode requires following configuration:
- `--scroll` or `--scroll-sepolia` - enables Scroll Mainnet or Sepolia mode
- `--da.blob.beaconnode` - L1 RPC beacon node
- `--l1.endpoint` - L1 RPC execution client
- `--da.sync=true` - enables syncing with L1
- `--da.recovery` - enables recovery mode
- `--da.recovery.signblocks` - enables signing blocks with the sequencer and configured key
- `--da.recovery.initiall1block` - initial L1 block (commit tx of initial batch)
- `--da.recovery.initialbatch` - batch where to start recovery from. Can be found on [Scrollscan Explorer](https://scrollscan.com/batches).
- `--da.recovery.l2endblock` - until which L2 block recovery should run (optional)

```bash
./build/bin/geth --scroll<-sepolia> \
--datadir "tmp/datadir" \
--gcmode archive \
--http --http.addr "0.0.0.0" --http.port 8545 --http.api "eth,net,web3,debug,scroll" --http.vhosts "*" \
--da.blob.beaconnode "<L1 RPC beacon node>" \
--l1.endpoint "<L1 RPC execution client>" \
--da.sync=true --da.recovery --da.recovery.signblocks --da.recovery.initiall1block "<initial L1 block (commit tx of initial batch)>" --da.recovery.initialbatch "<batch where to start recovery from>" --da.recovery.l2endblock "<until which L2 block recovery should run (optional)>" \
--verbosity 3
```

After the recovery is finished, start the sequencer in normal operation and continue issuing L2 blocks as normal. This will resume the L2 chain, allow the relayer (after running recovery) to create new batches and allow other L2 follower nodes to sync up the valid and signed L2 chain.

### Relayer recovery
Start the relayer with the following additional top-level configuration:
```
  "recovery_config": {
    "enable": true
  }
```

This will make the relayer recover all the chunks, batches and bundles that were submitted during permissionless mode. These batches are marked automatically as proven and finalized.
Once this process is finished, start the relayer normally without the recovery config to resume normal operation.
```
  "recovery_config": {
    "enable": false
  }
```