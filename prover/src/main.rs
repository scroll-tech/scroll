#![feature(lazy_cell)]

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

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args = Args::parse();

    if args.version {
        println!("version is {}", version::get_version());
        std::process::exit(0);
    }

    let config: Config = Config::from_file(args.config_file)?;

    let _guard = config.sentry.as_ref().and_then(|conf| {
        if conf.enabled {
            Some(sentry::init((
                conf.dsn.clone(),
                sentry::ClientOptions {
                    release: Some(version::get_version_cow()),
                    ..Default::default()
                },
            )))
        } else {
            None
        }
    });

    _guard.iter().for_each(|_| {
        sentry::configure_scope(|scope| {
            scope.set_tag("prover_type", config.prover_type);
            scope.set_tag("partner_name", config.partner_name());
        });

        sentry::capture_message("test message on start", sentry::Level::Info);
    });

    utils::log_init(args.log_file, _guard.is_some());

    if let Err(e) = AssetsDirEnvConfig::init() {
        log::error!("AssetsDirEnvConfig init failed: {:#}", e);
        std::process::exit(-2);
    }

    let task_cache = Rc::new(TaskCache::new(&config.db_path)?);

    let coordinator_listener = Box::new(ClearCacheCoordinatorListener {
        task_cache: task_cache.clone(),
    });

    let prover = Prover::new(&config, coordinator_listener)?;

    _guard.iter().for_each(|_| {
        sentry::configure_scope(|scope| {
            let public_key = sentry::protocol::Value::from(prover.get_public_key());
            scope.set_extra("public_key", public_key);
        });
    });

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
