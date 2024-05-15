mod version;
mod types;
mod config;
mod key_signer;
mod prover;
mod zk_circuits_handler;
mod coordinator_client;
mod geth_client;

use config::Config;
use prover::Prover;
use anyhow::Result;
use log;

struct TaskProcesser<'a> {
    prover: &'a Prover<'a>,
}

impl<'a> TaskProcesser<'a> {
    pub fn new(prover: &'a Prover) -> Self {
        TaskProcesser {
            prover: prover,
        }
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
        let task = self.prover.fetch_task()?;

        match self.prover.prove_task(&task) {
            Ok(proof_detail) => {
                self.prover.submit_proof(&proof_detail, task.uuid)
            },
            Err(error) => {
                self.prover.submit_error(&task, types::ProofFailureType::NoPanic, error)
            },
        }
    }
}

fn main() -> Result<(), Box<dyn std::error::Error>> {

    let file_name = "config.json";
    let config: Config = Config::from_file(file_name.to_string())?;

    println!("{:?}", config);
    
    let prover = Prover::new(&config)?;

    let task_processer = TaskProcesser::new(&prover);

    task_processer.start();

    Ok(())
}
