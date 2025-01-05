use anyhow::{bail, Result};

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
