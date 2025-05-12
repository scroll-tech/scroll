pub mod euclid;
#[allow(non_snake_case)]
pub mod euclidV2;

use anyhow::Result;
use async_trait::async_trait;
use scroll_proving_sdk::prover::{proving_service::ProveRequest, ProofType};

#[async_trait]
pub trait CircuitsHandler: Sync + Send {
    async fn get_vk(&self, task_type: ProofType) -> Option<Vec<u8>>;

    async fn get_proof_data(&self, prove_request: ProveRequest) -> Result<String>;
}

use alloy::{
    providers::{Provider, ProviderBuilder, RootProvider},
    rpc::client::ClientBuilder,
    transports::layers::RetryBackoffLayer,
};
use sbv_primitives::{types::Network, ChainId};
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone)]
pub struct RpcConfig {
    pub rpc_url: String,
    // Concurrency Limit, default 10
    //pub max_concurrency: usize,
    // Retry parameters
    pub max_retry: u32,
    // backoff duration in milliseconds, default 100ms
    pub backoff: u64,
    // compute units per second: default 100
    pub cups: u64,
}

pub struct RequestPreHandler {
    provider: RootProvider<Network>,
}

impl RequestPreHandler {
    pub fn create(config: &RpcConfig) -> Result<Self> {
        let rpc = url::Url::parse(&config.rpc_url)?;
        tracing::info!("Using RPC: {}", rpc);
        let retry_layer = RetryBackoffLayer::new(config.max_retry, config.backoff, config.cups);
        let client = ClientBuilder::default().layer(retry_layer).http(rpc);

        Ok(Self {
            provider: ProviderBuilder::<_, _, Network>::default().on_client(client),
        })
    }

    async fn on_chunk_request(&self, input: String) -> Result<String> {
        use crate::types::ChunkTaskDetail;
        use alloy::network::primitives::BlockTransactionsKind;
        use sbv_utils::{rpc::ProviderExt, witness::WitnessBuilder};
        use scroll_zkvm_prover_euclid::task::chunk::ChunkProvingTask;

        let chunk_task: ChunkTaskDetail = serde_json::from_str(&input)?;

        let chain_id = self.provider.get_chain_id().await?;

        // we need block number but only get hashes, which cause much extra cost for query the block
        // number from hash according to https://github.com/scroll-tech/scroll/blob/932be72b88ba2ebb6f9457e8480ee08d612d35a7/coordinator/internal/orm/l2_block.go#L53
        // the hashes is ordered by ascending in block number so a heuristic way is applied

        let mut block_witnesses = Vec::new();

        for block_hash in chunk_task.block_hashes {
            // grep `dump_block_witness` in sbv here,
            // TODO: we do not need to do that
            // if we have block number or `dump_block_witness` support block hashes
            let block = self
                .provider
                .get_block_by_hash(block_hash, BlockTransactionsKind::Full)
                .await?
                .ok_or_else(|| anyhow::anyhow!("Block not found"))?;

            let number = block.header.number;

            let builder = WitnessBuilder::new()
                .block(block)
                .chain_id(chain_id)
                .execution_witness(self.provider.debug_execution_witness(number.into()).await?);

            let builder = builder
                .state_root(
                    self.provider
                        .scroll_disk_root(number.into())
                        .await?
                        .disk_root,
                )
                .unwrap()
                .prev_state_root(
                    self.provider
                        .scroll_disk_root((number - 1).into())
                        .await?
                        .disk_root,
                );

            block_witnesses.push(builder.build()?);
        }

        let input_repack = ChunkProvingTask {
            fork_name: chunk_task.fork_name,
            prev_msg_queue_hash: chunk_task.prev_msg_queue_hash,
            block_witnesses,
        };
        //self.provider.dump_block_witness(number)

        Ok(serde_json::to_string(&input_repack)?)
    }

    pub async fn on_request(&self, mut prove_request: ProveRequest) -> Result<ProveRequest> {
        match prove_request.proof_type {
            ProofType::Chunk => {
                prove_request.input = self.on_chunk_request(prove_request.input).await?;
            }
            _ => (),
        }
        Ok(prove_request)
    }
}
