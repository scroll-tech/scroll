use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, fs::File};

use crate::types::ProverType;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct CircuitConfig {
    pub hard_fork_name: String,
    pub workspace_path: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct CoordinatorConfig {
    pub base_url: String,
    pub retry_count: u32,
    pub retry_wait_time_sec: u64,
    pub connection_timeout_sec: u64,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct L2GethConfig {
    pub endpoint: String,
}

#[derive(Debug, Deserialize, Clone)]
pub struct Config {
    pub prover_name: String,
    pub keystore_path: String,
    pub keystore_password: String,
    pub db_path: String,
    pub prover_type: ProverType,
    pub circuits: HashMap<String, CircuitConfig>,
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
