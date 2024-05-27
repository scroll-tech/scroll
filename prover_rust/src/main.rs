mod config;
mod coordinator_client;
mod geth_client;
mod key_signer;
mod prover;
mod task_cache;
mod types;
mod utils_log;
mod version;
mod zk_circuits_handler;
mod task_processor;

use anyhow::Result;
use config::Config;
use prover::Prover;
use std::rc::Rc;
use task_cache::{TaskCache, ClearCacheCoordinatorListener};
use task_processor::TaskProcessor;


fn main() -> Result<(), Box<dyn std::error::Error>> {
    utils_log::log_init();

    let file_name = "config.json";
    let config: Config = Config::from_file(file_name.to_string())?;

    println!("{:?}", config);

    let task_cache = Rc::new(TaskCache::new(&config.db_path)?);

    let coordinator_listener = Box::new(ClearCacheCoordinatorListener {
        task_cache: task_cache.clone(),
    });

    let prover = Prover::new(&config, coordinator_listener)?;

    let task_processer = TaskProcessor::new(&prover, task_cache);

    task_processer.start();

    Ok(())
}
