## Prover

A runnable zkvm prover which can communicate with coordinator, receiving proving task and generate proof

## Testing

+ Get the url of the endpoint of coordinator and a rpc endpoint response to the corresponding chain

+ Build a `config.json` file with previous knowledge from the template in current directory

+ Call `make test_run`