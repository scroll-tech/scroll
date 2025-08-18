//pub mod euclid;

#[allow(non_snake_case)]
pub mod universal;

use async_trait::async_trait;
use eyre::Result;
use scroll_zkvm_types::ProvingTask;

#[async_trait]
pub trait CircuitsHandler: Sync + Send {
    #[allow(dead_code)]
    async fn get_vk(&self) -> String;

    async fn get_proof_data(&self, u_task: &ProvingTask, need_snark: bool) -> Result<String>;
}
