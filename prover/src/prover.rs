use anyhow::{bail, Context, Error, Ok, Result};

use crate::{
    config::Config,
    coordinator_client::{listener::Listener, types::*, CoordinatorClient},
    geth_client::GethClient,
    key_signer::KeySigner,
    types::{ProofFailureType, ProofStatus, TaskType},
    zk_circuits_handler::{euclid::EuclidHandler, CircuitsHandler},
};

use super::types::{ProofDetail, Task};

pub struct Prover {
    config: Config,
    pub public_key: String,
    coordinator_client: CoordinatorClient,
    geth_client: GethClient,

    active_handler: Option<(String, Box<dyn CircuitsHandler>)>,
}

impl Prover {
    pub fn new(config: Config, coordinator_listener: Box<dyn Listener>) -> Result<Self> {
        let keystore_path = &config.keystore_path;
        let keystore_password = &config.keystore_password;

        let geth_client = GethClient::new(
            &config.prover_name,
            &config.l2geth.as_ref().unwrap().endpoint,
        )
        .context("failed to create l2 geth_client")?;

        let key_signer = KeySigner::new(keystore_path, keystore_password)?;
        let public_key = key_signer.get_public_key();

        let coordinator_client = CoordinatorClient::new(
            config.clone(),
            key_signer,
            coordinator_listener,
            vec![], /* todo: vks */
        )
        .context("failed to create coordinator_client")?;

        let prover = Prover {
            config,
            public_key,
            coordinator_client,
            geth_client,
            active_handler: None,
        };

        Ok(prover)
    }

    pub fn fetch_task(&mut self) -> Result<Task> {
        log::info!("[prover] start to fetch_task");
        let mut req = GetTaskRequest {
            task_types: vec![TaskType::Chunk, TaskType::Batch, TaskType::Bundle],
            prover_height: None,
        };

        let latest_block_number = self.geth_client.block_number()?.as_number();
        if let Some(v) = latest_block_number {
            if v.as_u64() == 0 {
                bail!("omit to prove task of the genesis block")
            }
            req.prover_height = Some(v.as_u64());
        } else {
            log::error!("[prover] failed to fetch latest confirmed block number, got None");
            bail!("failed to fetch latest confirmed block number, got None")
        }
        let resp = self.coordinator_client.get_task(&req)?;

        match resp.data {
            Some(d) => Ok(Task::from(d)),
            None => {
                bail!("data of get_task empty, while error_code is success. there may be something wrong in response data or inner logic.")
            }
        }
    }

    fn set_active_handler(&mut self, hard_fork_name: &str) {
        if let Some(handler) = &self.active_handler {
            if handler.0 == hard_fork_name {
                return;
            }
        }

        // if we got assigned a task for an unknown hard fork, there is something wrong in the
        // coordinator
        let config = self.config.circuits.get(hard_fork_name).unwrap();

        let handler = Box::new(match hard_fork_name {
            "euclid" => EuclidHandler::new(&config.workspace_path),
            _ => unreachable!(),
        }) as Box<dyn CircuitsHandler>;
        self.active_handler = Some((hard_fork_name.to_string(), handler));
    }

    pub fn prove_task(&mut self, task: &Task) -> Result<ProofDetail> {
        log::info!("[prover] start to prove_task, task id: {}", task.id);
        let mut proof_detail = ProofDetail {
            id: task.id.clone(),
            proof_type: task.task_type,
            ..Default::default()
        };

        self.set_active_handler(&task.hard_fork_name);
        proof_detail.proof_data = self
            .active_handler
            .as_ref()
            .unwrap()
            .1
            .get_proof_data(task, &self.geth_client)?;
        Ok(proof_detail)
    }

    pub fn submit_proof(&mut self, proof_detail: ProofDetail, task: &Task) -> Result<()> {
        log::info!(
            "[prover] start to submit_proof, task id: {}",
            proof_detail.id
        );

        let request = SubmitProofRequest {
            uuid: task.uuid.clone(),
            task_id: proof_detail.id,
            task_type: proof_detail.proof_type,
            status: ProofStatus::Ok,
            proof: proof_detail.proof_data,
            ..Default::default()
        };

        self.coordinator_client.submit_proof(&request).map(|_r| ())
    }

    pub fn submit_error(
        &mut self,
        task: &Task,
        failure_type: ProofFailureType,
        error: Error,
    ) -> Result<()> {
        log::info!("[prover] start to submit_error, task id: {}", task.id);
        let request = SubmitProofRequest {
            uuid: task.uuid.clone(),
            task_id: task.id.clone(),
            task_type: task.task_type,
            status: ProofStatus::Error,
            failure_type: Some(failure_type),
            failure_msg: Some(format!("{:#}", error)),
            ..Default::default()
        };

        self.coordinator_client.submit_proof(&request).map(|_r| ())
    }
}
