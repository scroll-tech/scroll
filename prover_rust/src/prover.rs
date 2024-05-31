use anyhow::{bail, Context, Error, Ok, Result};
use ethers_core::types::U64;
use once_cell::sync::Lazy;
use std::{cell::RefCell, cmp::Ordering, env, rc::Rc};
use log;

use crate::{
    config::Config,
    coordinator_client::{
        listener::Listener, types::*, CoordinatorClient,
    },
    geth_client::{get_block_number, GethClient},
    key_signer::KeySigner,
    types::{CommonHash, ProofFailureType, ProofStatus, ProofType},
    zk_circuits_handler::{CircuitsHandler, CircuitsHandlerProvider},
};

use super::types::{ProofDetail, Task};
use prover::{BlockTrace, ChunkHash, ChunkProof};

// Only used for debugging.
pub(crate) static OUTPUT_DIR: Lazy<Option<String>> =
    Lazy::new(|| env::var("PROVER_OUTPUT_DIR").ok());

pub struct Prover<'a> {
    config: &'a Config,
    key_signer: Rc<KeySigner>,
    circuits_handler_provider: CircuitsHandlerProvider,
    coordinator_client: RefCell<CoordinatorClient<'a>>,
    geth_client: Option<RefCell<GethClient>>,
    vks: Vec<String>,
}

impl<'a> Prover<'a> {
    pub fn new(config: &'a Config, coordinator_listener: Box<dyn Listener>) -> Result<Self> {
        let proof_type = config.proof_type;
        let keystore_path = &config.keystore_path;
        let keystore_password = &config.keystore_password;

        let key_signer = Rc::new(KeySigner::new(&keystore_path, &keystore_password)?);
        let coordinator_client = CoordinatorClient::new(
            config,
            Rc::clone(&key_signer),
            coordinator_listener,
        ).context("failed to create coordinator_client")?;

        let provider = CircuitsHandlerProvider::new(
            proof_type,
            config
        ).context("failed to create circuits handler provider")?;
        let vks = provider.get_vks();

        let mut prover = Prover {
            config,
            key_signer: Rc::clone(&key_signer),
            circuits_handler_provider: provider,
            coordinator_client: RefCell::new(coordinator_client),
            geth_client: None,
            vks,
        };

        if config.proof_type == ProofType::ProofTypeChunk {
            prover.geth_client = Some(RefCell::new(GethClient::new(
                &config.prover_name,
                &config.l2geth.as_ref().unwrap().endpoint,
            ).context("failed to create l2 geth_client")?));
        }

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
        let vks = self.vks.clone();
        let vk = vks[0].clone();
        let mut req = GetTaskRequest {
            task_type: self.get_proof_type(),
            prover_height: None,
            vks,
            vk,
        };

        if self.get_proof_type() == ProofType::ProofTypeChunk {
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

        Task::try_from(&resp.data.unwrap()).map_err(|e| anyhow::anyhow!(e))
    }

    pub fn prove_task(&self, task: &Task) -> Result<ProofDetail> {
        log::info!("[prover] start to prove_task, task id: {}", task.id);
        if let Some(handler) = self.circuits_handler_provider.get_circuits_client(&task.hard_fork_name) {
            self.do_prove(task, handler)
        } else {
            log::error!("failed to get a circuit handler");
            bail!("failed to get a circuit handler")
        }
    }

    fn do_prove(&self, task: &Task, handler: &Box<dyn CircuitsHandler>) -> Result<ProofDetail> {
        let mut proof_detail = ProofDetail {
            id: task.id.clone(),
            proof_type: task.task_type,
            ..Default::default()
        };
        match task.task_type {
            ProofType::ProofTypeBatch => {
                let chunk_hashes_proofs: Vec<(ChunkHash, ChunkProof)> =
                    self.gen_chunk_hashes_proofs(task)?;
                let chunk_proofs: Vec<ChunkProof> =
                    chunk_hashes_proofs.iter().map(|t| t.1.clone()).collect();
                let is_valid = handler.aggregator_check_chunk_proofs(&chunk_proofs)?;
                if !is_valid {
                    bail!("non-match chunk protocol, task-id: {}", &task.id)
                }
                let batch_proof = handler.aggregator_gen_agg_evm_proof(
                    chunk_hashes_proofs,
                    None,
                    self.get_output_dir(),
                )?;

                proof_detail.batch_proof = Some(batch_proof);
                Ok(proof_detail)
            }
            ProofType::ProofTypeChunk => {
                let chunk_trace = self.gen_chunk_traces(task)?;
                let chunk_proof = handler.prover_gen_chunk_proof(
                    chunk_trace,
                    None,
                    None,
                    self.get_output_dir(),
                )?;

                proof_detail.chunk_proof = Some(chunk_proof);
                Ok(proof_detail)
            }
            _ => bail!("task type invalid"),
        }
    }

    pub fn submit_proof(&self, proof_detail: &ProofDetail, uuid: String) -> Result<()> {
        log::info!("[prover] start to submit_proof, task id: {}", proof_detail.id);
        let proof_data = match proof_detail.proof_type {
            ProofType::ProofTypeBatch => {
                serde_json::to_string(proof_detail.batch_proof.as_ref().unwrap())?
            }
            ProofType::ProofTypeChunk => {
                serde_json::to_string(proof_detail.chunk_proof.as_ref().unwrap())?
            }
            _ => unreachable!(),
        };

        let request = SubmitProofRequest {
            uuid,
            task_id: proof_detail.id.clone(),
            task_type: proof_detail.proof_type,
            status: ProofStatus::Ok,
            proof: proof_data,
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

    fn get_output_dir(&self) -> Option<&str> {
        OUTPUT_DIR.as_deref()
    }

    fn gen_chunk_traces(&self, task: &Task) -> Result<Vec<BlockTrace>> {
        if let Some(chunk_detail) = task.chunk_task_detail.as_ref() {
            self.get_sorted_traces_by_hashes(&chunk_detail.block_hashes)
        } else {
            log::error!("[prover] failed to get chunk_detail from task");
            bail!("invalid task")
        }
    }

    fn gen_chunk_hashes_proofs(&self, task: &Task) -> Result<Vec<(ChunkHash, ChunkProof)>> {
        if let Some(batch_detail) = task.batch_task_detail.as_ref() {
            Ok(batch_detail
                .chunk_infos
                .clone()
                .into_iter()
                .zip(batch_detail.chunk_proofs.clone())
                .collect())
        } else {
            log::error!("[prover] failed to get batch_detail from task");
            bail!("invalid task")
        }
    }

    fn get_sorted_traces_by_hashes(
        &self,
        block_hashes: &Vec<CommonHash>,
    ) -> Result<Vec<BlockTrace>> {
        if block_hashes.len() == 0 {
            log::error!("[prover] failed to get sorted traces: block_hashes are empty");
            bail!("block_hashes are empty")
        }

        let mut block_traces = Vec::new();
        for (_, hash) in block_hashes.into_iter().enumerate() {
            let trace = self
                .geth_client
                .as_ref()
                .unwrap()
                .borrow_mut()
                .get_block_trace_by_hash(hash)?;
            block_traces.push(trace.block_trace);
        }

        block_traces.sort_by(|a, b| {
            if get_block_number(a) == None {
                Ordering::Less
            } else if get_block_number(b) == None {
                Ordering::Greater
            } else {
                get_block_number(a)
                    .unwrap()
                    .cmp(&get_block_number(b).unwrap())
            }
        });

        let block_numbers: Vec<u64> = block_traces
            .iter()
            .map(|trace| match get_block_number(trace) {
                Some(v) => v,
                None => 0,
            })
            .collect();
        let mut i = 0;
        while i < block_numbers.len() - 1 {
            if block_numbers[i] + 1 != block_numbers[i + 1] {
                log::error!("[prover] block numbers are not continuous, got {} and {}", block_numbers[i], block_numbers[i + 1]);
                bail!(
                    "block numbers are not continuous, got {} and {}",
                    block_numbers[i],
                    block_numbers[i + 1]
                )
            }
            i += 1;
        }

        Ok(block_traces)
    }
}
