pub mod euclid;

use anyhow::Result;
use async_trait::async_trait;
use scroll_proving_sdk::prover::{proving_service::ProveRequest, ProofType};

pub mod utils {
    pub fn encode_vk(vk: Vec<u8>) -> String {
        base64::encode(vk)
    }
}

#[async_trait]
pub trait CircuitsHandler: Send + Sync {
    async fn get_vk(&self, task_type: ProofType) -> Option<Vec<u8>>;

    async fn get_proof_data(&self, prove_request: ProveRequest) -> Result<String>;
}
