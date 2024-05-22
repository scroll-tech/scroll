use eth_types::H256;
use prover::{BatchProof, ChunkHash, ChunkProof};
use serde::{Deserialize, Deserializer, Serialize, Serializer};

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
pub struct BatchTaskDetail {
    pub chunk_infos: Vec<ChunkHash>,
    pub chunk_proofs: Vec<ChunkProof>,
}

#[derive(Serialize, Deserialize)]
pub struct ChunkTaskDetail {
    pub block_hashes: Vec<CommonHash>,
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
    #[serde(default)]
    pub hard_fork_name: Option<String>,
}

impl Task {
    pub fn get_version(&self) -> String {
        match self.hard_fork_name.as_ref() {
            Some(v) => v.clone(),
            None => "".to_string(),
        }
    }
}

impl TryFrom<&GetTaskResponseData> for Task {
    type Error = serde_json::Error;

    fn try_from(value: &GetTaskResponseData) -> Result<Self, Self::Error> {
        let mut task = Task {
            uuid: value.uuid.clone(),
            id: value.task_id.clone(),
            task_type: value.task_type,
            chunk_task_detail: None,
            batch_task_detail: None,
            hard_fork_name: value.hard_fork_name.clone(),
        };
        match task.task_type {
            ProofType::ProofTypeBatch => {
                task.batch_task_detail = Some(serde_json::from_str(&value.task_data)?);
            }
            ProofType::ProofTypeChunk => {
                task.chunk_task_detail = Some(serde_json::from_str(&value.task_data)?);
            }
            _ => unreachable!(),
        }
        Ok(task)
    }
}

#[derive(Serialize, Deserialize, Default)]
pub struct ProofDetail {
    pub id: String,
    #[serde(rename = "type", default)]
    pub proof_type: ProofType,
    pub chunk_proof: Option<ChunkProof>,
    pub batch_proof: Option<BatchProof>,
    pub error: String,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum ProofFailureType {
    Undefined,
    Panic,
    NoPanic,
}

impl ProofFailureType {
    fn from_u8(v: u8) -> Self {
        match v {
            1 => ProofFailureType::Panic,
            2 => ProofFailureType::NoPanic,
            _ => ProofFailureType::Undefined,
        }
    }
}

impl Serialize for ProofFailureType {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        match *self {
            ProofFailureType::Undefined => serializer.serialize_u8(0),
            ProofFailureType::Panic => serializer.serialize_u8(1),
            ProofFailureType::NoPanic => serializer.serialize_u8(2),
        }
    }
}

impl<'de> Deserialize<'de> for ProofFailureType {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        let v: u8 = u8::deserialize(deserializer)?;
        Ok(ProofFailureType::from_u8(v))
    }
}

impl Default for ProofFailureType {
    fn default() -> Self {
        Self::Undefined
    }
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum ProofStatus {
    Ok,
    Error,
}

impl ProofStatus {
    fn from_u8(v: u8) -> Self {
        match v {
            0 => ProofStatus::Ok,
            _ => ProofStatus::Error,
        }
    }
}

impl Serialize for ProofStatus {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        match *self {
            ProofStatus::Ok => serializer.serialize_u8(0),
            ProofStatus::Error => serializer.serialize_u8(1),
        }
    }
}

impl<'de> Deserialize<'de> for ProofStatus {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        let v: u8 = u8::deserialize(deserializer)?;
        Ok(ProofStatus::from_u8(v))
    }
}

impl Default for ProofStatus {
    fn default() -> Self {
        Self::Ok
    }
}
