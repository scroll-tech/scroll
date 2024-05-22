use anyhow::{bail, Error, Ok, Result};
use eth_types::U64;
use ethers_core::types::BlockNumber;
use once_cell::sync::Lazy;
use std::{cell::RefCell, cmp::Ordering, env, rc::Rc};

use crate::{
    config::Config,
    coordinator_client::{types::*, Config as CoordinatorConfig, CoordinatorClient},
    geth_client::{types::get_block_number, GethClient},
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
    coordinator_client: RefCell<CoordinatorClient>,
    geth_client: Option<RefCell<GethClient>>,
}

// a u64 is positive when it's 63th index bit not set
fn is_positive(n: &U64) -> bool {
    !n.bit(63)
}

impl<'a> Prover<'a> {
    pub fn new(config: &'a Config) -> Result<Self> {
        let proof_type = config.core.proof_type;
        let params_path = &config.core.params_path;
        let assets_path = &config.core.assets_path;
        let keystore_path = &config.keystore_path;
        let keystore_password = &config.keystore_password;

        let coordinator_config = CoordinatorConfig {
            endpoint: config.coordinator.base_url.clone(),
            prover_name: config.prover_name.clone(),
            prover_version: crate::version::get_version(),
            hard_fork_name: config.hard_fork_name.clone(),
        };

        let key_signer = Rc::new(KeySigner::new(&keystore_path, &keystore_password)?);
        let coordinator_client =
            CoordinatorClient::new(coordinator_config, Rc::clone(&key_signer))?;

        let mut prover = Prover {
            config,
            key_signer: Rc::clone(&key_signer),
            circuits_handler_provider: CircuitsHandlerProvider::new(
                proof_type,
                params_path,
                assets_path,
            )?,
            coordinator_client: RefCell::new(coordinator_client),
            geth_client: None,
        };

        if config.core.proof_type == ProofType::ProofTypeChunk {
            prover.geth_client = Some(RefCell::new(GethClient::new(
                "test",
                &config.l2geth.as_ref().unwrap().endpoint,
            )?));
        }

        Ok(prover)
    }

    pub fn get_proof_type(&self) -> ProofType {
        self.config.core.proof_type
    }

    pub fn get_public_key(&self) -> String {
        self.key_signer.get_public_key()
    }

    pub fn fetch_task(&self) -> Result<Task> {
        let vks = self.circuits_handler_provider.get_vks();
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
                bail!("failed to fetch latest confirmed block number, got None")
            }
        }
        let resp = self.coordinator_client.borrow_mut().get_task(&req)?;

        Task::try_from(&resp.data.unwrap()).map_err(|e| anyhow::anyhow!(e))
    }

    pub fn prove_task(&self, task: &Task) -> Result<ProofDetail> {
        let version = task.get_version();
        if let Some(handler) = self.circuits_handler_provider.get_circuits_client(version) {
            self.do_prove(task, handler)
        } else {
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
                let chunk_hashes_proofs = self.gen_chunk_hashes_proofs(task)?;
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

    // fn get_configured_block_number_value(&self) -> Result<Option<U64>> {
    //     self.get_block_number_value(&self.config.l2geth.as_ref().unwrap().confirmations)
    // }

    // fn get_block_number_value(&self, block_number: &BlockNumber) -> Result<Option<U64>> {
    //     match block_number {
    //         BlockNumber::Safe | BlockNumber::Finalized => {
    //             let header =
    // self.geth_client.as_ref().unwrap().borrow_mut().header_by_number(block_number)?;
    //             Ok(header.get_number())
    //         },
    //         BlockNumber::Latest => {
    //             let number = self.geth_client.as_ref().unwrap().borrow_mut().block_number()?;
    //             Ok(number.as_number())
    //         },
    //         BlockNumber::Number(n) if is_positive(n) => {
    //             let number = self.geth_client.as_ref().unwrap().borrow_mut().block_number()?;
    //             let diff = number.as_number()
    //                 .filter(|m| m.as_u64() >= n.as_u64())
    //                 .map(|m| U64::from(m.as_u64() - n.as_u64()));
    //             Ok(diff)
    //         },
    //         _ => bail!("unknown confirmation type"),
    //     }
    // }

    fn get_output_dir(&self) -> Option<&str> {
        OUTPUT_DIR.as_deref()
    }

    fn gen_chunk_traces(&self, task: &Task) -> Result<Vec<BlockTrace>> {
        if let Some(chunk_detail) = task.chunk_task_detail.as_ref() {
            self.get_sorted_traces_by_hashes(&chunk_detail.block_hashes)
        } else {
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
            bail!("invalid task")
        }
    }

    fn get_sorted_traces_by_hashes(
        &self,
        block_hashes: &Vec<CommonHash>,
    ) -> Result<Vec<BlockTrace>> {
        if block_hashes.len() == 0 {
            bail!("blockHashes is empty")
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
