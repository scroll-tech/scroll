use prover::BatchProof;
use serde::{Deserialize, Serialize, Serializer, Deserializer};
use eth_types::H256;

use crate::coordinator_client::types::GetTaskResponseData;

pub type CommonHash = H256;

pub type Bytes = Vec<u8>;

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum ProofType {
    ProofTypeUndefined,
    ProofTypeChunk,
    ProofTypeBatch,
}

impl ProofType {
    fn from_u8(v: u8) -> Self {
        match v {
            1 => ProofType::ProofTypeChunk,
            2 => ProofType::ProofTypeBatch,
            _ => ProofType::ProofTypeUndefined,
        }
    }
}

impl Serialize for ProofType {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        match *self {
            ProofType::ProofTypeUndefined => serializer.serialize_i8(0),
            ProofType::ProofTypeChunk => serializer.serialize_i8(1),
            ProofType::ProofTypeBatch => serializer.serialize_i8(2),
        }
    }
}

impl<'de> Deserialize<'de> for ProofType {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        let v: u8 = u8::deserialize(deserializer)?;
        Ok(ProofType::from_u8(v))
    }
}

impl Default for ProofType {
    fn default() -> Self {
        Self::ProofTypeUndefined
    }
}

#[derive(Serialize, Deserialize)]
pub struct ChunkInfo {
    pub chain_id: u64,
    pub prev_state_root: CommonHash,
    pub post_state_root: CommonHash,
    pub withdraw_root: CommonHash,
    pub data_hash: CommonHash,
    pub is_padding: bool,
    pub tx_bytes: Bytes,
}

#[derive(Serialize, Deserialize)]
pub struct ChunkProof {
    pub storage_trace: Bytes,
    pub protocol: Bytes,
    pub proof: Bytes,
    pub instances: Bytes,
    pub vk: Bytes,
    pub chunk_info: ChunkInfo,
    pub git_version: String,
}

#[derive(Serialize, Deserialize)]
pub struct BatchTaskDetail {
    chunk_infos: Vec<ChunkInfo>,
    chunk_proofs: Vec<ChunkProof>,
}

#[derive(Serialize, Deserialize)]
pub struct ChunkTaskDetail {
    block_hashes: Vec<CommonHash>,
}

#[derive(Serialize, Deserialize, Default)]
pub struct Task {
    pub uuid: String,
    pub id: String,
    #[serde(rename = "type", default)]
    pub task_type: ProofType,
    #[serde(default)]
    pub batch_task_detail: Option<BatchTaskDetail>,
    #[serde(default)]
    pub chunk_task_detail: Option<ChunkTaskDetail>,
}

impl TryFrom<&GetTaskResponseData> for Task {
    type Error = serde_json::Error;

    fn try_from(value: &GetTaskResponseData) -> Result<Self, Self::Error> {
        let mut task = Task {
            uuid: value.uuid,
            id: value.task_id,
            task_type: value.task_type,
            chunk_task_detail: None,
            batch_task_detail: None,
        };
        match task.task_type {
            ProofType::ProofTypeBatch => {
                task.batch_task_detail = serde_json::from_str(&value.task_data)?;
            },
            ProofType::ProofTypeChunk => {
                task.chunk_task_detail = serde_json::from_str(&value.task_data)?;
            },
            _ => unreachable!()
        }
        Ok(task)
    }
}

#[derive(Serialize, Deserialize)]
pub struct ProofDetail {
    pub id: String,
    #[serde(rename = "type", default)]
    pub proof_type: ProofType,
    pub status: u32,
    pub chunk_proof: Option<ChunkProof>,
    pub batch_proof: Option<BatchProof>,
    pub error: String,
}