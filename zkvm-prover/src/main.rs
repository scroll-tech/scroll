mod prover;
mod types;
mod zk_circuits_handler;

use clap::{ArgAction, Parser};
use prover::{LocalProver, LocalProverConfig};
use scroll_proving_sdk::{
    prover::ProverBuilder,
    utils::{get_version, init_tracing},
};

#[derive(Parser, Debug)]
#[command(disable_version_flag = true)]
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
        println!("version is {}", get_version());
        std::process::exit(0);
    }

    let cfg = LocalProverConfig::from_file(args.config_file)?;
    let sdk_config = cfg.sdk_config.clone();
    let local_prover = LocalProver::new(cfg);
    let prover = ProverBuilder::new(sdk_config, local_prover).build().await?;

    prover.run().await;

    Ok(())
}
