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

use anyhow::{Context, Result};
use config::Config;
use coordinator_client::listener::Listener;
use log;
use prover::Prover;
use core::time;
use std::rc::Rc;
use task_cache::TaskCache;
use types::TaskWrapper;

struct ClearCacheCoordinatorListener {
    pub task_cache: Rc<TaskCache>,
}

impl Listener for ClearCacheCoordinatorListener {
    fn on_proof_submitted(&self, req: &coordinator_client::types::SubmitProofRequest) {
        let result = self.task_cache.delete_task(req.task_id.clone());
        if let Err(e) = result {
            log::error!("delete task from embed db failed, {}", e.to_string());
        } else {
            log::info!(
                "delete task from embed db successfully, task_id: {}",
                &req.task_id
            );
        }
    }
}

struct TaskProcessor<'a> {
    prover: &'a Prover<'a>,
    task_cache: Rc<TaskCache>,
}

impl<'a> TaskProcessor<'a> {
    pub fn new(prover: &'a Prover, task_cache: Rc<TaskCache>) -> Self {
        TaskProcessor { prover, task_cache }
    }

    pub fn start(&self) {
        loop {
            log::info!("start a new round.");
            if let Err(err) = self.prove_and_submit() {
                log::error!("encounter error: {err}");
            } else {
                log::info!("prove & submit succeed.");
            }
        }
    }

    fn prove_and_submit(&self) -> Result<()> {
        let task_from_cache = self
            .task_cache
            .get_last_task()
            .context("failed to peek from stack")?;

        let mut task_wrapper = match task_from_cache {
            Some(t) => t,
            None => {
                let fetch_result = self.prover.fetch_task();
                if let Err(err) = fetch_result {
                    std::thread::sleep(time::Duration::from_secs(10));
                    return Err(err).context("failed to fetch task from coordinator");
                }
                let task_wrapper: TaskWrapper = fetch_result.unwrap().into();
                self.task_cache
                    .put_task(&task_wrapper)
                    .context("failed to push task into stack")?;
                task_wrapper
            }
        };

        if task_wrapper.get_count() <= 2 {
            task_wrapper.increment_count();
            self.task_cache
                .put_task(&task_wrapper)
                .context("failed to push task into stack, updating count")?;

            log::info!(
                "start to prove task, task_type: {:?}, task_id: {}",
                task_wrapper.task.task_type,
                task_wrapper.task.id
            );
            let result = match self.prover.prove_task(&task_wrapper.task) {
                Ok(proof_detail) => self
                    .prover
                    .submit_proof(&proof_detail, task_wrapper.task.uuid.clone()),
                Err(error) => self.prover.submit_error(
                    &task_wrapper.task,
                    types::ProofFailureType::NoPanic,
                    error,
                ),
            };
            return result;
        }

        // if tried times >= 3, it's probably due to circuit proving panic
        log::error!(
            "zk proving panic for task, task_type: {:?}, task_id: {}",
            task_wrapper.task.task_type,
            task_wrapper.task.id
        );
        self.prover.submit_error(
            &task_wrapper.task,
            types::ProofFailureType::Panic,
            anyhow::anyhow!("zk proving panic for task"),
        )
    }
}

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
