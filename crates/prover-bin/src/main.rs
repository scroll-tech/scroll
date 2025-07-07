mod prover;
mod types;
mod zk_circuits_handler;

use clap::{ArgAction, Parser, Subcommand};
use prover::{LocalProver, LocalProverConfig};
use scroll_proving_sdk::{
    prover::{types::ProofType, ProverBuilder},
    utils::{get_version, init_tracing},
};
use std::{fs::File, io::BufReader, path::Path};

#[derive(Parser, Debug)]
#[command(disable_version_flag = true)]
struct Args {
    /// Path of config file
    #[arg(long = "config", default_value = "conf/config.json")]
    config_file: String,

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
    /// Dump vk of this prover
    Dump {
        /// path to save the verifier's asset
        asset_path: String,
    },
    Handle {
        /// path to save the verifier's asset
        task_path: String,
    },
}

#[derive(Debug, serde::Deserialize)]
struct HandleSet {
    chunks: Vec<String>,
    batches: Vec<String>,
    bundles: Vec<String>,
}

#[tokio::main]
async fn main() -> eyre::Result<()> {
    init_tracing();

    let args = Args::parse();

    if args.version {
        println!("version is {}", get_version());
        std::process::exit(0);
    }

    let cfg = LocalProverConfig::from_file(args.config_file)?;
    let default_fork_name = cfg.circuits.keys().next().unwrap().clone();
    let sdk_config = cfg.sdk_config.clone();
    let local_prover = LocalProver::new(cfg.clone());

    match args.command {
        Some(Commands::Dump { asset_path }) => {
            let fork_name = args.fork_name.unwrap_or(default_fork_name);
            println!("dump assets for {fork_name} into {asset_path}");
            local_prover.dump_verifier_assets(&fork_name, asset_path.as_ref())?;
        }
        Some(Commands::Handle { task_path }) => {
            let file = File::open(Path::new(&task_path))?;
            let reader = BufReader::new(file);
            let handle_set: HandleSet = serde_json::from_reader(reader)?;

            let prover = ProverBuilder::new(sdk_config, local_prover)
                .build()
                .await
                .map_err(|e| eyre::eyre!("build prover fail: {e}"))?;

            let prover = std::sync::Arc::new(prover);
            println!("Handling task set 1: chunks ...");
            assert!(
                prover
                    .clone()
                    .one_shot(&handle_set.chunks, ProofType::Chunk)
                    .await
            );
            println!("Done! Handling task set 2: batches ...");
            assert!(
                prover
                    .clone()
                    .one_shot(&handle_set.batches, ProofType::Batch)
                    .await
            );
            println!("Done! Handling task set 3: bundles ...");
            assert!(
                prover
                    .clone()
                    .one_shot(&handle_set.bundles, ProofType::Bundle)
                    .await
            );
            println!("All done!");
        }
        None => {
            let prover = ProverBuilder::new(sdk_config, local_prover)
                .build()
                .await
                .map_err(|e| eyre::eyre!("build prover fail: {e}"))?;

            prover.run().await;
        }
    }

    Ok(())
}
