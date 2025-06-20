use alloy::{
    providers::{Provider, ProviderBuilder, RootProvider},
    rpc::client::ClientBuilder,
    transports::layers::RetryBackoffLayer,
};
use eyre::Result;
use libzkp::tasks::ChunkInterpreter;
use sbv_primitives::types::Network;
use serde::{Deserialize, Serialize};

fn default_max_retry() -> u32 {
    10
}
fn default_backoff() -> u64 {
    100
}
fn default_cups() -> u64 {
    100
}
fn default_workers() -> usize {
    4
}
fn default_max_concurrency() -> usize {
    10
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct RpcConfig {
    #[serde(alias = "endpoint")]
    pub rpc_url: String,
    // The threads used in rt, default 4
    #[serde(default = "default_workers")]
    pub workers: usize,
    // The blocking threads to handle rpc tasks, default 10
    #[serde(default = "default_max_concurrency")]
    pub max_concurrency: usize,
    // Retry parameters
    #[serde(default = "default_max_retry")]
    pub max_retry: u32,
    // backoff duration in milliseconds, default 100ms
    #[serde(default = "default_backoff")]
    pub backoff: u64,
    // compute units per second: default 100
    #[serde(default = "default_cups")]
    pub cups: u64,
}

/// An rpc client prover which carrying async runtime,
/// so it can be run in block mode (i.e. inside dynamic library without a global entry)
pub struct RpcClientCore {
    /// rpc prover
    provider: RootProvider<Network>,
    rt: tokio::runtime::Runtime,
}

#[derive(Clone, Copy)]
pub struct RpcClient<'a> {
    provider: &'a RootProvider<Network>,
    handle: &'a tokio::runtime::Handle,
}

impl RpcClientCore {
    pub fn create(config: &RpcConfig) -> Result<Self> {
        let rpc = url::Url::parse(&config.rpc_url)?;
        tracing::info!("Using RPC: {}", rpc);
        // note we MUST use multi rt since we have no a main thread for driving
        // for each call in our method we can acquire a handle of the rt to resolve one or more
        // async tasks
        let rt = tokio::runtime::Builder::new_multi_thread()
            .worker_threads(config.workers)
            .max_blocking_threads(config.max_concurrency)
            .enable_all()
            .build()?;

        let retry_layer = RetryBackoffLayer::new(config.max_retry, config.backoff, config.cups);
        let client = ClientBuilder::default().layer(retry_layer).http(rpc);

        Ok(Self {
            provider: ProviderBuilder::<_, _, Network>::default().on_client(client),
            rt,
        })
    }

    pub fn get_client(&self) -> RpcClient {
        RpcClient {
            provider: &self.provider,
            handle: self.rt.handle(),
        }
    }
}

impl ChunkInterpreter for RpcClient<'_> {
    fn try_fetch_block_witness(
        &self,
        block_hash: sbv_primitives::B256,
        prev_witness: Option<&sbv_primitives::types::BlockWitness>,
    ) -> Result<sbv_primitives::types::BlockWitness> {
        async fn fetch_witness_async(
            provider: &RootProvider<Network>,
            block_hash: sbv_primitives::B256,
            prev_witness: Option<&sbv_primitives::types::BlockWitness>,
        ) -> Result<sbv_primitives::types::BlockWitness> {
            use alloy::network::primitives::BlockTransactionsKind;
            use sbv_utils::{rpc::ProviderExt, witness::WitnessBuilder};

            let chain_id = provider.get_chain_id().await?;

            let block = provider
                .get_block_by_hash(block_hash, BlockTransactionsKind::Full)
                .await?
                .ok_or_else(|| eyre::eyre!("Block not found"))?;

            let number = block.header.number;
            if number == 0 {
                eyre::bail!("no number in header or use block 0");
            }

            let prev_state_root = if let Some(witness) = prev_witness {
                if witness.header.number != number - 1 {
                    eyre::bail!(
                        "the ref witness is not the previous block, expected {} get {}",
                        number - 1,
                        witness.header.number,
                    );
                }
                witness.header.state_root
            } else {
                provider
                    .scroll_disk_root((number - 1).into())
                    .await?
                    .disk_root
            };

            let witness = WitnessBuilder::new()
                .block(block)
                .chain_id(chain_id)
                .execution_witness(provider.debug_execution_witness(number.into()).await?)
                .state_root(provider.scroll_disk_root(number.into()).await?.disk_root)?
                .prev_state_root(prev_state_root)
                .build()?;

            Ok(witness)
        }

        tracing::debug!("fetch witness for {block_hash}");
        self.handle
            .block_on(fetch_witness_async(self.provider, block_hash, prev_witness))
    }

    fn try_fetch_storage_node(
        &self,
        node_hash: sbv_primitives::B256,
    ) -> Result<sbv_primitives::Bytes> {
        async fn fetch_storage_node_async(
            provider: &RootProvider<Network>,
            node_hash: sbv_primitives::B256,
        ) -> Result<sbv_primitives::Bytes> {
            let ret = provider
                .client()
                .request::<_, sbv_primitives::Bytes>("debug_dbGet", (node_hash,))
                .await?;
            Ok(ret)
        }

        tracing::debug!("fetch storage node for {node_hash}");
        self.handle
            .block_on(fetch_storage_node_async(self.provider, node_hash))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy::primitives::hex;
    use sbv_primitives::B256;
    use std::env;

    fn create_config_from_env() -> RpcConfig {
        let endpoint =
            env::var("L2GETH_ENDPOINT").expect("L2GETH_ENDPOINT environment variable must be set");

        let config_json = format!(r#"{{"endpoint": "{}"}}"#, endpoint);
        serde_json::from_str(&config_json).expect("Failed to parse RPC config")
    }

    #[test]
    #[ignore = "Requires L2GETH_ENDPOINT environment variable"]
    fn test_try_fetch_block_witness() {
        let config = create_config_from_env();
        let client_core = RpcClientCore::create(&config).expect("Failed to create RPC client");
        let client = client_core.get_client();

        // latest - 1 block in 2025.6.15
        let block_hash = B256::from(
            hex::const_decode_to_array(
                b"0x9535a6970bc4db9031749331a214e35ed8c8a3f585f6f456d590a0bc780a1368",
            )
            .unwrap(),
        );

        // This is expected to fail since we're using a dummy hash, but it tests the code path
        let wit1 = client
            .try_fetch_block_witness(block_hash, None)
            .expect("should success");

        // latest block in 2025.6.15
        let block_hash = B256::from(
            hex::const_decode_to_array(
                b"0xd47088cdb6afc68aa082e633bb7da9340d29c73841668afacfb9c1e66e557af0",
            )
            .unwrap(),
        );
        let wit2 = client
            .try_fetch_block_witness(block_hash, Some(&wit1))
            .expect("should success");

        println!("{}", serde_json::to_string_pretty(&wit2).unwrap());
    }

    #[test]
    #[ignore = "Requires L2GETH_ENDPOINT environment variable"]
    fn test_try_fetch_storage_node() {
        let config = create_config_from_env();
        let client_core = RpcClientCore::create(&config).expect("Failed to create RPC client");
        let client = client_core.get_client();

        // the root node (state root) of the block in unittest above
        let node_hash = B256::from(
            hex::const_decode_to_array(
                b"0xb9e67403a2eb35afbb0475fe942918cf9a330a1d7532704c24554506be62b27c",
            )
            .unwrap(),
        );

        // This is expected to fail since we're using a dummy hash, but it tests the code path
        let node = client
            .try_fetch_storage_node(node_hash)
            .expect("should success");
        println!("{}", serde_json::to_string_pretty(&node).unwrap());
    }
}
