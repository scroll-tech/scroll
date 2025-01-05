#![feature(lazy_cell)]
#![feature(core_intrinsics)]

mod config;
mod coordinator_client;
mod geth_client;
mod key_signer;
mod types;
mod utils;
mod version;
mod zk_circuits_handler;
mod prover;

use clap::Parser;
use scroll_proving_sdk::{config::Config, prover::ProverBuilder, utils::init_tracing};
use prover::LocalProver;

#[derive(Parser, Debug)]
#[clap(disable_version_flag = true)]
struct Args {
    /// Path of config file
    #[arg(long = "config", default_value = "config.json")]
    config_file: String,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    init_tracing();

    let args = Args::parse();
    let cfg: Config = Config::from_file(args.config_file)?;
    let local_prover = LocalProver::new(cfg.prover.local.clone().unwrap());
    let prover = ProverBuilder::new(cfg)
        .with_proving_service(Box::new(local_prover))
        .build()
        .await?;

    prover.run().await;

    Ok(())
}
