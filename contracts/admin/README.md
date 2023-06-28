# admin cli

WIP

provides commands to generate calldata to then paste into `cast sign` or similar tools. No cast sign raw tx exists, and want to give users ability to
chose what method they sign with, so prefer not signing the tx in this cli tool.

example (hypothetical) usage:

- npm link
- admin-cli approveHash --network testnet --domain L1 --targetAddress 0x0 --targetCalldata 0x0

{
to: 0x1234,
data: 0x1234,
functionSig: "approveHash(bytes32)"
}

Flow:

- first, approve desired transaction (schedules transaction in Timelock) in SAFE with approveHash()
- second, someone collects all the signers and sends executeTransaction()
- third, someone calls execute() on the Timelock. this actually sends the transaction throught the forwarder and executes the call
