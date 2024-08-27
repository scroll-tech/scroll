#![feature(lazy_cell)]
#![feature(core_intrinsics)]

mod config;
mod coordinator_client;
mod geth_client;
mod key_signer;
mod prover;
mod task_cache;
mod task_processor;
mod types;
mod utils;
mod version;
mod zk_circuits_handler;

use anyhow::Result;
use clap::{ArgAction, Parser};
use config::{AssetsDirEnvConfig, Config};
use prover::Prover;
use std::rc::Rc;
use task_cache::{ClearCacheCoordinatorListener, TaskCache};
use task_processor::TaskProcessor;

/// Simple program to greet a person
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

fn start() -> Result<()> {
    let args = Args::parse();

    if args.version {
        println!("version is {}", version::get_version());
        std::process::exit(0);
    }

    utils::log_init(args.log_file);

    let config: Config = Config::from_file(args.config_file)?;

    if let Err(e) = AssetsDirEnvConfig::init() {
        log::error!("AssetsDirEnvConfig init failed: {:#}", e);
        std::process::exit(-2);
    }

    let task_cache = Rc::new(TaskCache::new(&config.db_path)?);

    let coordinator_listener = Box::new(ClearCacheCoordinatorListener {
        task_cache: task_cache.clone(),
    });

    let prover = Prover::new(&config, coordinator_listener)?;

    log::info!(
        "prover start successfully. name: {}, type: {:?}, publickey: {}, version: {}",
        config.prover_name,
        config.prover_type,
        prover.get_public_key(),
        version::get_version(),
    );

    let task_processor = TaskProcessor::new(&prover, task_cache);

    task_processor.start();

    Ok(())
}

fn main() {
    let result = start();
    if let Err(e) = result {
        log::error!("main exit with error {:#}", e)
    }
}
