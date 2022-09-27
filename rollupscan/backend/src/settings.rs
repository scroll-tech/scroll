use anyhow::{anyhow, Result};
use config::{Config, Environment, File};
use serde::Deserialize;
use std::env;
use std::sync::OnceLock;

static SETTINGS: OnceLock<Settings> = OnceLock::new();

#[derive(Debug, Deserialize)]
pub struct Settings {
    /// As format of `postgres://USERNAME:PASSWORD@DB_HOST:DB_PORT/DATABASE`
    pub db_url: String,
    /// As format of `HTTP_HOST:HTTP_PORT`
    pub open_api_addr: String,
    /// `development` or `production`
    run_mode: String,
}

impl Settings {
    pub fn init() -> Result<()> {
        let run_mode = env::var("RUN_MODE").unwrap_or_else(|_| "development".into());
        let config = Config::builder()
            .set_default("run_mode", run_mode.clone())?
            .add_source(File::with_name("config/default"))
            .add_source(File::with_name(&format!("config/{}", run_mode)).required(false))
            .add_source(Environment::default())
            .build()?;

        let settings: Settings = config.try_deserialize()?;
        SETTINGS
            .set(settings)
            .map_err(|s| anyhow!("Wrong settings: {:?}", s))?;

        Ok(())
    }

    pub fn get() -> &'static Self {
        SETTINGS.get().unwrap()
    }

    pub fn is_dev(&self) -> bool {
        self.run_mode == "development"
    }

    pub fn is_prod(&self) -> bool {
        self.run_mode == "production"
    }
}
