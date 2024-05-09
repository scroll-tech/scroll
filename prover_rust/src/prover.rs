use anyhow::{Ok, Result};
use ethers_core::types::BlockNumber;
use eth_types::U64;

use crate::{config::Config, types::ProofType};
use crate::zk_circuits_handler::CircuitsHandlerProvider;
use crate::coordinator_client::CoordinatorClient;
use crate::coordinator_client::types::*;
use crate::geth_client::GethClient;

use super::types::{Task, ProofDetail};

pub struct Prover<'a> {
    config: &'a Config,
    circuits_client_provider: CircuitsHandlerProvider,
    coordinator_client: CoordinatorClient,
    geth_client: GethClient,
}

// a u64 is positive when it's 0 index bit not set
fn is_positive(n: &U64) -> bool {
    !n.bit(0)
}

impl<'a> Prover<'a> {
    pub fn new(config: &'a Config) -> Result<Self> {
        let proof_type = config.core.proof_type;
        let params_path = config.core.params_path;
        let assets_path = config.core.assets_path;

        let prover = Prover {
            config: config,
            circuits_client_provider: CircuitsHandlerProvider::new(proof_type, &params_path, &assets_path)?,
            coordinator_client: CoordinatorClient::new(),
            geth_client: GethClient::new("test", &config.l2geth.endpoint)?,
        };

        Ok(prover)
    }

    pub fn get_proof_type(&self) -> ProofType {
        self.config.core.proof_type
    }

    pub fn get_public_key(&self) -> String {
        "".to_string()
    }

    pub fn fetch_task(&self) -> Result<Task> {
        let mut req = GetTaskRequest {
            task_type: self.get_proof_type(),
            prover_height: None,
            vks: self.circuits_client_provider.get_vks(),
        };

        if self.get_proof_type() == ProofType::ProofTypeChunk {
            let latest_block_number = self.get_configured_block_number_value()?;
            if let Some(v) = latest_block_number {
                if v.as_u64() == 0 {
                    unreachable!()
                }
                req.prover_height = Some(v.as_u64());
            }

            unreachable!()
        }

        let resp = self.coordinator_client.get_task(req)?;
        
        Task::try_from(&resp.data.unwrap()).map_err(|e| anyhow::anyhow!(e))
    }

    pub fn prove_task(&self, task: &Task) -> Result<ProofDetail>  {

    }

    pub fn submit_proof(&self, proof_detail: &ProofDetail, uuid: String) -> Result<()> {
        Ok(())
    }

    fn get_configured_block_number_value(&self) -> Result<Option<U64>> {
        self.get_block_number_value(&self.config.l2geth.confirmations)
    }

    fn get_block_number_value(&self, block_number: &BlockNumber) -> Result<Option<U64>> {
        match block_number {
            BlockNumber::Safe | BlockNumber::Finalized => {
                let header = self.geth_client.header_by_number(block_number)?;
                Ok(header.get_number())
            },
            BlockNumber::Latest => {
                let number = self.geth_client.block_number()?;
                Ok(number.as_number())
            },
            BlockNumber::Number(n) if is_positive(n) => {
                let number = self.geth_client.block_number()?;
                let diff = number.as_number()
                    .filter(|m| m.as_u64() >= n.as_u64())
                    .map(|m| U64::from(m.as_u64() - n.as_u64()));
                Ok(diff)
            },
            _ => unreachable!(),
        }
    }
}