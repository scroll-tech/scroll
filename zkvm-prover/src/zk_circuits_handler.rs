pub mod euclid;

#[allow(non_snake_case)]
pub mod euclidV2;

use anyhow::Result;
use async_trait::async_trait;
use scroll_proving_sdk::prover::{proving_service::ProveRequest, ProofType};

#[async_trait]
pub trait CircuitsHandler: Sync + Send {
    async fn get_vk(&self, task_type: ProofType) -> Option<Vec<u8>>;

    async fn get_proof_data(&self, prove_request: ProveRequest) -> Result<String>;
}
