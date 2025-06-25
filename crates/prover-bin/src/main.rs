mod prover;
mod types;
mod zk_circuits_handler;

use clap::{ArgAction, Parser, Subcommand};
use prover::{LocalProver, LocalProverConfig};
use scroll_proving_sdk::{
    prover::{types::ProofType, ProverBuilder},
    utils::{get_version, init_tracing},
};
use std::{fs::File, path::Path};

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
        /// File to save the vks
        file_name: String,
    },
}

fn dump_vk(file: &Path, prover: &LocalProver, fork_name: &str) -> eyre::Result<()> {
    let f = File::create(file)?;

    #[derive(Debug, serde::Serialize)]
    struct VKDump {
        pub chunk_vk: String,
        pub batch_vk: String,
        pub bundle_vk: String,
    }

    let handler = prover.new_handler(fork_name);

    let dump = VKDump {
        chunk_vk: handler.get_vk(ProofType::Chunk),
        batch_vk: handler.get_vk(ProofType::Batch),
        bundle_vk: handler.get_vk(ProofType::Bundle),
    };
    serde_json::to_writer(f, &dump)?;

    Ok(())
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
        Some(Commands::Dump { file_name }) => {
            let fork_name = args.fork_name.unwrap_or(default_fork_name);
            println!("dump vk for {fork_name}");
            dump_vk(Path::new(&file_name), &local_prover, &fork_name)?;
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
