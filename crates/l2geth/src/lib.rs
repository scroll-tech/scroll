pub mod rpc_client;
pub use rpc_client::RpcConfig;

use std::sync::{Arc, OnceLock};

static GLOBAL_L2GETH_CLI: OnceLock<Arc<rpc_client::RpcClientCore>> = OnceLock::new();

pub fn init(config: &str) -> eyre::Result<()> {
    let cfg: RpcConfig = serde_json::from_str(config)?;
    GLOBAL_L2GETH_CLI.get_or_init(|| Arc::new(rpc_client::RpcClientCore::create(&cfg).unwrap()));
    Ok(())
}

pub fn get_client() -> rpc_client::RpcClient<'static> {
    GLOBAL_L2GETH_CLI
        .get()
        .expect("must has been inited")
        .get_client()
}
