mod dumper;
mod prover;
mod types;
mod zk_circuits_handler;

use clap::{ArgAction, Parser, Subcommand, ValueEnum};
use prover::{LocalProver, LocalProverConfig};
use scroll_proving_sdk::{
    prover::{types::ProofType, ProverBuilder},
    utils::{init_tracing, VERSION},
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

#[derive(Clone, Debug, PartialEq, Eq, ValueEnum)]
enum TaskType {
    Chunk,
    Batch,
    Bundle,
}

impl From<TaskType> for ProofType {
    fn from(value: TaskType) -> Self {
        match value {
            TaskType::Chunk => ProofType::Chunk,
            TaskType::Batch => ProofType::Batch,
            TaskType::Bundle => ProofType::Bundle,
        }
    }
}

#[derive(Subcommand, Debug)]
enum Commands {
    Handle {
        /// path to save the verifier's asset
        task_path: String,
    },
    Dump {
        #[arg(long = "json", default_value = "false")]
        json_mode: bool,
        task_type: TaskType,
        task_id: String,
    },
}

#[derive(Debug, serde::Deserialize)]
struct HandleSet {
    chunks: Vec<String>,
    batches: Vec<String>,
    bundles: Vec<String>,
}

#[tokio::main(flavor = "current_thread")]
async fn main() -> eyre::Result<()> {
    init_tracing();

    let args = Args::parse();

    if args.version {
        println!("version is {}", VERSION);
        std::process::exit(0);
    }

    let cfg = LocalProverConfig::from_file(args.config_file)?;
    let sdk_config = cfg.sdk_config.clone();
    let local_prover = LocalProver::new(cfg.clone());

    // Pre-flight check: verify all circuit asset URLs are reachable before any proving.
    // This catches misconfigured base_url (e.g., extra "releases/" prefix) early,
    // rather than failing mid-proving with an opaque HTTP 403.
    for (fork_name, circuit) in &cfg.circuits {
        circuit.location_data.preflight_check().await.map_err(|e| {
            eyre::eyre!("Pre-flight check failed for fork '{}': {}\n  Asset URL: {}", fork_name, e, circuit.location_data.base_url)
        })?;
    }

    match args.command {
        Some(Commands::Dump {
            json_mode,
            task_type,
            task_id,
        }) => {
            let prover = ProverBuilder::new(
                sdk_config,
                dumper::Dumper {
                    json_mode,
                    ..Default::default()
                },
            )
            .build()
            .await
            .map_err(|e| eyre::eyre!("build prover fail: {e}"))?;

            std::sync::Arc::new(prover)
                .one_shot(&[task_id], task_type.into())
                .await;
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
