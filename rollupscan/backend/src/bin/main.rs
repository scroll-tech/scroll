use anyhow::Result;
use dotenv::dotenv;
use rollup_explorer::{cache, open_api, Settings};
use std::sync::Arc;

#[tokio::main]
async fn main() -> Result<()> {
    dotenv().ok();
    env_logger::init();

    Settings::init()?;
    log::debug!("{:?}", Settings::get());

    let mut cache = Arc::new(cache::run()?);
    open_api::run(cache.clone()).await?;
    Arc::get_mut(&mut cache).unwrap().stop().await
}
