use serde::{Deserialize, Serialize};
use std::fs::File;
use anyhow::Result;

use crate::types::ProofType;

#[derive(Debug, Serialize, Deserialize)]
pub struct CircuitConfig {
    pub hard_fork_name: String,
    pub params_path: String,
    pub assets_path: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct CoordinatorConfig {
    pub base_url: String,
    pub retry_count: u32,
    pub retry_wait_time_sec: u64,
    pub connection_timeout_sec: u64,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct L2GethConfig {
    pub endpoint: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Config {
    pub prover_name: String,
    pub keystore_path: String,
    pub keystore_password: String,
    pub db_path: String,
    #[serde(default)]
    pub proof_type: ProofType,
    pub low_version_circuit: CircuitConfig,
    pub high_version_circuit: CircuitConfig,
    pub coordinator: CoordinatorConfig,
    pub l2geth: Option<L2GethConfig>,
}

impl Config {
    pub fn from_reader<R>(reader: R) -> Result<Self>
    where
        R: std::io::Read,
    {
        serde_json::from_reader(reader).map_err(|e| anyhow::anyhow!(e))
    }

    pub fn from_file(file_name: String) -> Result<Self> {
        let file = File::open(file_name)?;
        Config::from_reader(&file)
    }
}
