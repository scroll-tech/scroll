use serde::{Deserialize, Serialize};
use std::{error::Error, fs::File};

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
    pub retry_count: u16,
    pub retry_wait_time_sec: u32,
    pub connection_timeout_sec: u32,
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
    pub fn from_reader<R>(reader: R) -> Result<Self, Box<dyn Error>>
    where
        R: std::io::Read,
    {
        serde_json::from_reader(reader).map_err(|e| Box::new(e) as Box<dyn Error>)
    }

    pub fn from_file(file_name: String) -> Result<Self, Box<dyn Error>> {
        let file = File::open(file_name)?;
        Config::from_reader(&file)
    }
}
