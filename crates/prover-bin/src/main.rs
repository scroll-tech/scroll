#[macro_use]
extern crate tracing;

mod prover;
mod types;
mod zk_circuits_handler;

use crate::prover::ProverKind;
use clap::{ArgAction, Parser, Subcommand};
use scroll_proving_sdk::{
    prover::{ProverBuilder, types::ProofType},
    utils::{VERSION, init_tracing},
};
use std::{
    fs::File,
    io::BufReader,
    path::{Path, PathBuf},
};

#[derive(Parser, Debug)]
#[command(disable_version_flag = true)]
struct Args {
    /// Prover kind
    #[arg(long = "prover.kind", value_enum, default_value_t = ProverKind::Local)]
    prover_kind: ProverKind,

    /// Path of config file
    #[arg(long = "config", default_value = "conf/config.json")]
    config_file: PathBuf,

    #[arg(long = "forkname")]
    fork_name: Option<String>,

    /// Version of this prover
    #[arg(short, long, action = ArgAction::SetTrue)]
    version: bool,

    /// Path of log file
    #[arg(long = "log.file")]
    log_file: Option<String>,

    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(Subcommand, Debug)]
enum Commands {
    Handle {
        /// path to save the verifier's asset
        task_path: String,
    },
}

#[derive(Debug, serde::Deserialize)]
struct HandleSet {
    #[serde(default)]
    chunks: Vec<String>,
    #[serde(default)]
    batches: Vec<String>,
    #[serde(default)]
    bundles: Vec<String>,
}

#[tokio::main]
async fn main() -> eyre::Result<()> {
    init_tracing();

    let args = Args::parse();

    if args.version {
        println!("version is {VERSION}");
        std::process::exit(0);
    }
    info!(version = %VERSION, "Starting prover");

    let (sdk_config, prover) = args.prover_kind.create_from_file(&args.config_file)?;
    info!(prover = ?prover, "Loaded prover");

    match args.command {
        Some(Commands::Handle { task_path }) => {
            let file = File::open(Path::new(&task_path))?;
            let reader = BufReader::new(file);
            let handle_set: HandleSet = serde_json::from_reader(reader)?;

            let prover = ProverBuilder::new(sdk_config, prover)
                .build()
                .await
                .map_err(|e| eyre::eyre!("build prover fail: {e}"))?;

            let prover = std::sync::Arc::new(prover);
            info!("Handling task set 1: chunks ...");
            assert!(
                prover
                    .clone()
                    .one_shot(&handle_set.chunks, ProofType::Chunk)
                    .await
            );
            info!("Done! Handling task set 2: batches ...");
            assert!(
                prover
                    .clone()
                    .one_shot(&handle_set.batches, ProofType::Batch)
                    .await
            );
            info!("Done! Handling task set 3: bundles ...");
            assert!(
                prover
                    .clone()
                    .one_shot(&handle_set.bundles, ProofType::Bundle)
                    .await
            );
            info!("All done!");
        }
        None => {
            let prover = ProverBuilder::new(sdk_config, prover)
                .build()
                .await
                .map_err(|e| eyre::eyre!("build prover fail: {e}"))?;

            prover.run().await;
        }
    }

    Ok(())
}
