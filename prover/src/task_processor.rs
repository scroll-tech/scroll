use super::{coordinator_client::ProofStatusNotOKError, prover::Prover, task_cache::TaskCache};
use anyhow::{Context, Result};
use std::rc::Rc;

pub struct TaskProcessor<'a> {
    prover: &'a Prover<'a>,
    task_cache: Rc<TaskCache>,
}

impl<'a> TaskProcessor<'a> {
    pub fn new(prover: &'a Prover<'a>, task_cache: Rc<TaskCache>) -> Self {
        TaskProcessor { prover, task_cache }
    }

    pub fn start(&self) {
        loop {
            log::info!("start a new round.");
            if let Err(err) = self.prove_and_submit() {
                if err.is::<ProofStatusNotOKError>() {
                    log::info!("proof status not ok, downgrade level to info.");
                } else {
                    log::error!("encounter error: {:#}", err);
                }
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
                    std::thread::sleep(core::time::Duration::from_secs(10));
                    return Err(err).context("failed to fetch task from coordinator");
                }
                fetch_result.unwrap().into()
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
                Ok(proof_detail) => self.prover.submit_proof(proof_detail, &task_wrapper.task),
                Err(error) => {
                    log::error!(
                        "failed to prove task, id: {}, error: {:#}",
                        &task_wrapper.task.id,
                        error
                    );
                    self.prover.submit_error(
                        &task_wrapper.task,
                        super::types::ProofFailureType::NoPanic,
                        error,
                    )
                }
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
            super::types::ProofFailureType::Panic,
            anyhow::anyhow!("zk proving panic for task"),
        )
    }
}
