use anyhow::{anyhow, Result};
use async_trait::async_trait;
use scroll_proving_sdk::prover::{proving_service::ProveRequest, ProofType};

use super::CircuitsHandler;
pub struct EuclidHandler {}

impl EuclidHandler {
    pub fn new(_workspace_path: &str) -> Self {
        Self {}
    }
}

#[async_trait]
impl CircuitsHandler for EuclidHandler {
    async fn get_vk(&self, _task_type: ProofType) -> Option<Vec<u8>> {
        None
    }

    async fn get_proof_data(&self, _prove_request: ProveRequest) -> Result<String> {
        Err(anyhow!("Not implemented"))
    }
}
