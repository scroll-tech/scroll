#![feature(lazy_cell)]
#![feature(core_intrinsics)]

mod config;
mod prover;
mod types;
mod utils;
mod zk_circuits_handler;

use clap::{ArgAction, Parser};
use prover::{LocalProver, LocalProverConfig};
use scroll_proving_sdk::{
    prover::ProverBuilder,
    utils::{get_version, init_tracing},
};
use tokio::runtime;
use utils::get_prover_type;

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

fn main() -> anyhow::Result<()> {
    let rt = runtime::Builder::new_multi_thread()
        .thread_stack_size(16 * 1024 * 1024) // Set stack size to 16MB
        .enable_all()
        .build()
        .expect("Failed to create Tokio runtime");

    rt.block_on(async {
        init_tracing();

        let args = Args::parse();

        if args.version {
            println!("version is {}", get_version());
            std::process::exit(0);
        }

        let cfg = LocalProverConfig::from_file(args.config_file)?;
        let sdk_config = cfg.sdk_config.clone();
        let mut prover_types = vec![];
        sdk_config
            .prover
            .circuit_types
            .iter()
            .for_each(|circuit_type| {
                if let Some(pt) = get_prover_type(*circuit_type) {
                    if !prover_types.contains(&pt) {
                        prover_types.push(pt);
                    }
                }
            });
        let local_prover = LocalProver::new(cfg, prover_types);
        let prover = ProverBuilder::new(sdk_config)
            .with_proving_service(Box::new(local_prover))
            .build()
            .await?;

        prover.run().await;

        Ok(())
    })
}
