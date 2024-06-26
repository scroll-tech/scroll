use std::io;
use std::rc::Rc;

use async_trait::async_trait;
use snarkify_sdk::prover::ProofHandler;
use prover_runner::{
    prover_core::Prover,
    types::{Task, ProofDetail},
    config::{Config, AssetsDirEnvConfig},
    version,
};

struct MyProofHandler;


#[async_trait]
impl ProofHandler for MyProofHandler {
    type Input = Task;
    type Output = ProofDetail;
    type Error = String;

    async fn prove(data: Self::Input) -> Result<Self::Output, Self::Error> {
        let config: Config = Config::from_file("config.json".to_string()).map_err(|e| e.to_string())?;

        if let Err(e) = AssetsDirEnvConfig::init() {
            log::error!("AssetsDirEnvConfig init failed: {:#}", e);
            std::process::exit(-2);
        }

        let prover = Prover::new(&config).map_err(|e| e.to_string())?;

        log::info!(
            "prover start successfully. name: {}, type: {:?}, publickey: {}, version: {}",
            config.prover_name,
            config.proof_type,
            prover.get_public_key(),
            version::get_version(),
        );

        prover.prove_task(&data).map_err(|e| e.to_string())
    }
}

fn main() -> Result<(), io::Error> {
    snarkify_sdk::run::<MyProofHandler>()
}