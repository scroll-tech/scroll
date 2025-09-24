use alloy::{
    providers::{Provider, ProviderBuilder},
    rpc::client::ClientBuilder,
    transports::layers::RetryBackoffLayer,
};
use eyre::Result;
use libzkp::tasks::ChunkInterpreter;
use sbv_primitives::types::{consensus::TxL1Message, Network};
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
    client: alloy::rpc::client::RpcClient,
    rt: tokio::runtime::Runtime,
}

#[derive(Clone, Copy)]
pub struct RpcClient<'a, T: Provider<Network>> {
    provider: T,
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

        Ok(Self { client, rt })
    }

    pub fn get_client(&self) -> RpcClient<'_, impl Provider<Network>> {
        RpcClient {
            provider: ProviderBuilder::<_, _, Network>::default()
                .connect_client(self.client.clone()),
            handle: self.rt.handle(),
        }
    }
}

impl<T: Provider<Network>> ChunkInterpreter for RpcClient<'_, T> {
    fn try_fetch_block_witness(
        &self,
        block_hash: sbv_primitives::B256,
        prev_witness: Option<&sbv_core::BlockWitness>,
    ) -> Result<sbv_core::BlockWitness> {
        async fn fetch_witness_async(
            provider: impl Provider<Network>,
            block_hash: sbv_primitives::B256,
            prev_witness: Option<&sbv_core::BlockWitness>,
        ) -> Result<sbv_core::BlockWitness> {
            use sbv_utils::rpc::ProviderExt;

            let (chain_id, block_num, prev_state_root) = if let Some(w) = prev_witness {
                (w.chain_id, w.header.number + 1, w.header.state_root)
            } else {
                let chain_id = provider.get_chain_id().await?;
                let block = provider
                    .get_block_by_hash(block_hash)
                    .full()
                    .await?
                    .ok_or_else(|| eyre::eyre!("Block {block_hash} not found"))?;

                let parent_block = provider
                    .get_block_by_hash(block.header.parent_hash)
                    .await?
                    .ok_or_else(|| {
                        eyre::eyre!(
                            "parent block for block {} should exist",
                            block.header.number
                        )
                    })?;

                (
                    chain_id,
                    block.header.number,
                    parent_block.header.state_root,
                )
            };

            let req = provider
                .dump_block_witness(block_num)
                .with_chain_id(chain_id)
                .with_prev_state_root(prev_state_root);

            let witness = req
                .send()
                .await
                .transpose()
                .ok_or_else(|| eyre::eyre!("Block witness {block_num} not available"))??;

            Ok(witness)
        }

        tracing::debug!("fetch witness for {block_hash}");
        self.handle.block_on(fetch_witness_async(
            &self.provider,
            block_hash,
            prev_witness,
        ))
    }

    fn try_fetch_storage_node(
        &self,
        node_hash: sbv_primitives::B256,
    ) -> Result<sbv_primitives::Bytes> {
        async fn fetch_storage_node_async(
            provider: impl Provider<Network>,
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
            .block_on(fetch_storage_node_async(&self.provider, node_hash))
    }

    fn try_fetch_l1_msgs(&self, block_number: u64) -> Result<Vec<TxL1Message>> {
        async fn fetch_l1_msgs(
            provider: impl Provider<Network>,
            block_number: u64,
        ) -> Result<Vec<TxL1Message>> {
            let block_number_hex = format!("0x{:x}", block_number);
            Ok(provider
                .client()
                .request::<_, Vec<TxL1Message>>("scroll_getL1MessagesInBlock", (block_number_hex, "synced"))
                .await?)
        }

        tracing::debug!("fetch L1 msgs for {block_number}");
        self.handle
            .block_on(fetch_l1_msgs(&self.provider, block_number))
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
    // #[ignore = "Requires L2GETH_ENDPOINT environment variable"]
    fn test_try_fetch_block_witness() {
        let config = create_config_from_env();
        let client_core = RpcClientCore::create(&config).expect("Failed to create RPC client");
        let client = client_core.get_client();

        // latest - 1 block in 2025.9.11
        let block_hash = B256::from(
            hex::const_decode_to_array(
                b"0x093fb6bf2e556a659b35428ac447cd9f0635382fc40ffad417b5910824f9e932",
            )
            .unwrap(),
        );

        // This is expected to fail since we're using a dummy hash, but it tests the code path
        let wit1 = client
            .try_fetch_block_witness(block_hash, None)
            .expect("should success");

        // block selected in 2025.9.11
        let block_hash = B256::from(
            hex::const_decode_to_array(
                b"0x77cc84dd7a4dedf6fe5fb9b443aeb5a4fb0623ad088a365d3232b7b23fc848e5",
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
    fn test_try_fetch_l1_messages() {
        let config = create_config_from_env();
        let client_core = RpcClientCore::create(&config).expect("Failed to create RPC client");
        let client = client_core.get_client();

        let msgs = client
            .try_fetch_l1_msgs(32)
            .expect("should success");

        println!("{}", serde_json::to_string_pretty(&msgs).unwrap());
    }
}
