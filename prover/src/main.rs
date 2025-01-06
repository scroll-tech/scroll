#![feature(lazy_cell)]
#![feature(core_intrinsics)]

mod config;
mod prover;
mod types;
mod utils;
mod version;
mod zk_circuits_handler;

use clap::{ArgAction, Parser};
use prover::LocalProver;
use scroll_proving_sdk::{config::Config, prover::ProverBuilder, utils::init_tracing};

#[derive(Parser, Debug)]
#[clap(disable_version_flag = true)]
struct Args {
    /// Path of config file
    #[arg(long = "config", default_value = "conf/config.json")]
    config_file: String,

    /// Version of this prover
    #[arg(short, long, action = ArgAction::SetTrue)]
    version: bool,

    /// Path of log file
    #[arg(long = "log.file")]
    log_file: Option<String>,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    init_tracing();

    let args = Args::parse();

    if args.version {
        println!("version is {}", version::get_version());
        std::process::exit(0);
    }

    utils::log_init(args.log_file);

    let cfg: Config = Config::from_file(args.config_file)?;
    let local_prover = LocalProver::new(
        cfg.prover
            .local
            .clone()
            .ok_or_else(|| anyhow::anyhow!("Missing local prover configuration"))?,
    );
    let prover = ProverBuilder::new(cfg)
        .with_proving_service(Box::new(local_prover))
        .build()
        .await?;

    prover.run().await;

    Ok(())
}
