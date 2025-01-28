pub mod euclid;

use crate::{
    geth_client::GethClient,
    types::{Task, TaskType},
};
use anyhow::Result;

pub mod utils {
    pub fn encode_vk(vk: Vec<u8>) -> String {
        base64::encode(vk)
    }
}

pub trait CircuitsHandler {
    fn get_vk(&self, task_type: TaskType) -> Option<Vec<u8>>;

    fn get_proof_data(&self, task: &Task, geth_client: &GethClient) -> Result<String>;
}
