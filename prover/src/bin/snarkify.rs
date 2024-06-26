use std::io;

use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use snarkify_sdk::prover::ProofHandler;
// #[path="../prover_core.rs"]
// mod prover_core;
//
// use prover_core::Prover;

struct MyProofHandler;

#[derive(Deserialize)]
struct MyInput {
    public_input: String,
}

#[derive(Serialize)]
struct MyOutput {
    proof: String,
}

#[async_trait]
impl ProofHandler for MyProofHandler {
    type Input = MyInput;
    type Output = MyOutput;
    type Error = ();

    async fn prove(data: Self::Input) -> Result<Self::Output, Self::Error> {
        Ok(MyOutput {
            proof: data.public_input.chars().rev().collect(),
        })
    }
}

fn main() -> Result<(), io::Error> {
    snarkify_sdk::run::<MyProofHandler>()
}