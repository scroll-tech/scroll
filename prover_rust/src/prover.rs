use anyhow::{bail, Context, Error, Ok, Result};
use ethers_core::types::U64;

use std::{cell::RefCell, rc::Rc};

use crate::{
    config::Config,
    coordinator_client::{listener::Listener, types::*, CoordinatorClient},
    geth_client::GethClient,
    key_signer::KeySigner,
    types::{ProofFailureType, ProofStatus, ProofType},
    zk_circuits_handler::{CircuitsHandler, CircuitsHandlerProvider},
};

use super::types::{ProofDetail, Task};

pub struct Prover<'a> {
    config: &'a Config,
    key_signer: Rc<KeySigner>,
    circuits_handler_provider: RefCell<CircuitsHandlerProvider<'a>>,
    coordinator_client: RefCell<CoordinatorClient<'a>>,
    geth_client: Option<Rc<RefCell<GethClient>>>,
}

impl<'a> Prover<'a> {
    pub fn new(config: &'a Config, coordinator_listener: Box<dyn Listener>) -> Result<Self> {
        let proof_type = config.proof_type;
        let keystore_path = &config.keystore_path;
        let keystore_password = &config.keystore_password;

        let key_signer = Rc::new(KeySigner::new(keystore_path, keystore_password)?);
        let coordinator_client =
            CoordinatorClient::new(config, Rc::clone(&key_signer), coordinator_listener)
                .context("failed to create coordinator_client")?;

        let geth_client = if config.proof_type == ProofType::Chunk {
            Some(Rc::new(RefCell::new(
                GethClient::new(
                    &config.prover_name,
                    &config.l2geth.as_ref().unwrap().endpoint,
                )
                .context("failed to create l2 geth_client")?,
            )))
        } else {
            None
        };

        let provider = CircuitsHandlerProvider::new(proof_type, config, geth_client.clone())
            .context("failed to create circuits handler provider")?;

        let prover = Prover {
            config,
            key_signer: Rc::clone(&key_signer),
            circuits_handler_provider: RefCell::new(provider),
            coordinator_client: RefCell::new(coordinator_client),
            geth_client,
        };

        Ok(prover)
    }

    pub fn get_proof_type(&self) -> ProofType {
        self.config.proof_type
    }

    pub fn get_public_key(&self) -> String {
        self.key_signer.get_public_key()
    }

    pub fn fetch_task(&self) -> Result<Task> {
        log::info!("[prover] start to fetch_task");
        let mut req = GetTaskRequest {
            task_type: self.get_proof_type(),
            prover_height: None,
            vks: self.circuits_handler_provider.borrow().get_vks(),
        };

        if self.get_proof_type() == ProofType::Chunk {
            let latest_block_number = self.get_latest_block_number_value()?;
            if let Some(v) = latest_block_number {
                if v.as_u64() == 0 {
                    bail!("omit to prove task of the genesis block")
                }
                req.prover_height = Some(v.as_u64());
            } else {
                log::error!("[prover] failed to fetch latest confirmed block number, got None");
                bail!("failed to fetch latest confirmed block number, got None")
            }
        }
        let resp = self.coordinator_client.borrow_mut().get_task(&req)?;

        match resp.data {
            Some(d) => Ok(Task::from(d)),
            None => {
                bail!("data of get_task empty, while error_code is success. there may be something wrong in response data or inner logic.")
            }
        }
    }

    pub fn prove_task(&self, task: &Task) -> Result<ProofDetail> {
        log::info!("[prover] start to prove_task, task id: {}", task.id);
        let handler: Rc<Box<dyn CircuitsHandler>> = self
            .circuits_handler_provider
            .borrow_mut()
            .get_circuits_handler(&task.hard_fork_name)
            .context("failed to get circuit handler")?;
        self.do_prove(task, handler)
    }

    fn do_prove(&self, task: &Task, handler: Rc<Box<dyn CircuitsHandler>>) -> Result<ProofDetail> {
        let mut proof_detail = ProofDetail {
            id: task.id.clone(),
            proof_type: task.task_type,
            ..Default::default()
        };

        proof_detail.proof_data = handler.get_proof_data(task.task_type, task)?;
        Ok(proof_detail)
    }

    pub fn submit_proof(&self, proof_detail: ProofDetail, task: &Task) -> Result<()> {
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
            hard_fork_name: task.hard_fork_name.clone(),
            ..Default::default()
        };

        self.do_submit(&request)
    }

    pub fn submit_error(
        &self,
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
            failure_msg: Some(error.to_string()),
            hard_fork_name: task.hard_fork_name.clone(),
            ..Default::default()
        };
        self.do_submit(&request)
    }

    fn do_submit(&self, request: &SubmitProofRequest) -> Result<()> {
        self.coordinator_client.borrow_mut().submit_proof(request)?;
        Ok(())
    }

    fn get_latest_block_number_value(&self) -> Result<Option<U64>> {
        let number = self
            .geth_client
            .as_ref()
            .unwrap()
            .borrow_mut()
            .block_number()?;
        Ok(number.as_number())
    }
}
