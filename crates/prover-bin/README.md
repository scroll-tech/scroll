## Prover

A runnable zkvm prover which can communicate with coordinator, receving proving task and generate proof

## Testing

+ Get the url of the endpoint of coordinator and a rpc endpoint response to the cooresponding chain

+ Build a `config.json` file with previous knowledge from the template in current directory

+ Call `make test_run`