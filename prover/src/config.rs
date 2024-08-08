use anyhow::{bail, Result};
use regex::Regex;
use serde::{Deserialize, Serialize};
use std::fs::File;

use crate::types::ProverType;

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
pub struct SentryConfig {
    pub dsn: String,
    pub enabled: bool,
}

#[derive(Debug, Deserialize)]
pub struct Config {
    pub prover_name: String,
    pub keystore_path: String,
    pub keystore_password: String,
    pub db_path: String,
    pub prover_type: ProverType,
    pub low_version_circuit: CircuitConfig,
    pub high_version_circuit: CircuitConfig,
    pub coordinator: CoordinatorConfig,
    pub l2geth: Option<L2GethConfig>,
    pub sentry: Option<SentryConfig>,
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

    pub fn partner_name(&self) -> String {
        let prover_name = &self.prover_name;
        let scroll_prefix = Regex::new(r"^scroll-.*").unwrap();
        let idc_prefix = Regex::new(r"^idc-.*").unwrap();

        if scroll_prefix.is_match(prover_name) || idc_prefix.is_match(prover_name) {
            let parts = prover_name.split('-').collect::<Vec<&str>>();
            format!("{}-{}", parts[0], parts[1])
        } else {
            let split_re = Regex::new(r"[-_]").unwrap();
            let parts = split_re.split(prover_name).collect::<Vec<&str>>();
            parts[0].to_string()
        }
    }
}

static SCROLL_PROVER_ASSETS_DIR_ENV_NAME: &str = "SCROLL_PROVER_ASSETS_DIR";
static mut SCROLL_PROVER_ASSETS_DIRS: Vec<String> = vec![];

#[derive(Debug)]
pub struct AssetsDirEnvConfig {}

impl AssetsDirEnvConfig {
    pub fn init() -> Result<()> {
        let value = std::env::var(SCROLL_PROVER_ASSETS_DIR_ENV_NAME)?;
        let dirs: Vec<&str> = value.split(',').collect();
        if dirs.len() != 2 {
            bail!("env variable SCROLL_PROVER_ASSETS_DIR value must be 2 parts seperated by comma.")
        }
        unsafe {
            SCROLL_PROVER_ASSETS_DIRS = dirs.into_iter().map(|s| s.to_string()).collect();
            log::info!(
                "init SCROLL_PROVER_ASSETS_DIRS: {:?}",
                SCROLL_PROVER_ASSETS_DIRS
            );
        }
        Ok(())
    }

    pub fn enable_first() {
        unsafe {
            log::info!(
                "set env {SCROLL_PROVER_ASSETS_DIR_ENV_NAME} to {}",
                &SCROLL_PROVER_ASSETS_DIRS[0]
            );
            std::env::set_var(
                SCROLL_PROVER_ASSETS_DIR_ENV_NAME,
                &SCROLL_PROVER_ASSETS_DIRS[0],
            );
        }
    }

    pub fn enable_second() {
        unsafe {
            log::info!(
                "set env {SCROLL_PROVER_ASSETS_DIR_ENV_NAME} to {}",
                &SCROLL_PROVER_ASSETS_DIRS[1]
            );
            std::env::set_var(
                SCROLL_PROVER_ASSETS_DIR_ENV_NAME,
                &SCROLL_PROVER_ASSETS_DIRS[1],
            );
        }
    }
}
