use ethers_core::types::H256;
use serde::{Deserialize, Deserializer, Serialize, Serializer};

use crate::coordinator_client::types::GetTaskResponseData;

pub type CommonHash = H256;

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

#[derive(Serialize, Deserialize, Default)]
pub struct Task {
    pub uuid: String,
    pub id: String,
    #[serde(rename = "type", default)]
    pub task_type: ProofType,
    pub task_data: String,
    #[serde(default)]
    pub hard_fork_name: String,
}

impl From<GetTaskResponseData> for Task {
    fn from(value: GetTaskResponseData) -> Self {
        Self {
            uuid: value.uuid,
            id: value.task_id,
            task_type: value.task_type,
            task_data: value.task_data,
            hard_fork_name: value.hard_fork_name,
        }
    }
}

#[derive(Serialize, Deserialize, Default)]
pub struct TaskWrapper {
    pub task: Task,
    count: usize,
}

impl TaskWrapper {
    pub fn increment_count(&mut self) {
        self.count += 1;
    }

    pub fn get_count(&self) -> usize {
        self.count
    }
}

impl From<Task> for TaskWrapper {
    fn from(task: Task) -> Self {
        TaskWrapper { task, count: 0 }
    }
}

#[derive(Serialize, Deserialize, Default)]
pub struct ProofDetail {
    pub id: String,
    #[serde(rename = "type", default)]
    pub proof_type: ProofType,
    pub proof_data: String,
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
