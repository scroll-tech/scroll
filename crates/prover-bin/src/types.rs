use serde::{Deserialize, Deserializer, Serialize, Serializer};

#[derive(Serialize, Deserialize, Default)]
pub struct Task {
    #[serde(rename = "type", default)]
    pub task_data: String,
    #[serde(default)]
    pub hard_fork_name: String,
}

#[derive(Serialize, Deserialize, Default)]
pub struct ProofDetail {
    pub id: String,
    #[serde(rename = "type", default)]
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
