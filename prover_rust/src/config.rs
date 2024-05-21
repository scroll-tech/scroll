
use ethers_core::types::BlockNumber;
use serde::{Deserialize, Serialize};
// use serde_json::Error;
use std::error::Error;
use std::fs::File;

use crate::types::ProofType;


#[derive(Debug)]
#[derive(Serialize, Deserialize)]
pub struct ProverCoreConfig {
    pub params_path: String,
    pub assets_path: String,

    #[serde(default)]
    pub proof_type: ProofType,
    #[serde(default)]
    pub dump_dir: String,
}

#[derive(Debug)]
#[derive(Serialize, Deserialize)]
pub struct CoordinatorConfig{
    pub base_url: String,
    pub retry_count: u16,
    pub retry_wait_time_sec: u32,
    pub connection_timeout_sec: u32,
}

#[derive(Debug)]
#[derive(Serialize, Deserialize)]
pub struct L2GethConfig{
    pub endpoint: String,
    pub confirmations: BlockNumber,
}

#[derive(Debug)]
#[derive(Serialize, Deserialize)]
pub struct Config {
    pub prover_name: String,
    pub hard_fork_name: String,
    pub keystore_path: String,
    pub keystore_password: String,
    pub db_path: String,
    pub core: ProverCoreConfig,
    pub coordinator: CoordinatorConfig,
    pub l2geth: Option<L2GethConfig>,
}

impl Config {
    pub fn from_reader<R>(reader: R) -> Result<Self, Box<dyn Error>>
    where
        R: std::io::Read,
    {
        serde_json::from_reader(reader).map_err( |e| Box::new(e) as Box<dyn Error>)
    }

    pub fn from_file(file_name: String) -> Result<Self, Box<dyn Error>>
    {
        let file = File::open(file_name)?;
        Config::from_reader(&file)
    }
}