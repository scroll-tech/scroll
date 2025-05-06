use sbv_core::{EvmDatabase, EvmExecutor};
use sbv_primitives::{
    BlockWitness,
    chainspec::{
        BaseFeeParams, BaseFeeParamsKind, Chain, MAINNET,
        reth_chainspec::ChainSpec,
        scroll::{ScrollChainConfig, ScrollChainSpec},
    },
    ext::{BlockWitnessChunkExt, TxBytesHashExt},
    hardforks::SCROLL_DEV_HARDFORKS,
    types::{
        consensus::BlockHeader,
        reth::{Block, BlockWitnessRethExt, RecoveredBlock},
        scroll::ChunkInfoBuilder,
    },
};

use crate::{ArchivedChunkWitness, make_providers, manually_drop_on_zkvm};
use types_base::public_inputs::{
    ForkName,
    chunk::{BlockContextV2, ChunkInfo},
};

fn block_ctxv2_from_block(value: &RecoveredBlock<Block>) -> BlockContextV2 {
    use alloy_primitives::U256;
    BlockContextV2 {
        timestamp: value.timestamp,
        gas_limit: value.gas_limit,
        base_fee: U256::from(value.base_fee_per_gas().expect("base_fee_expected")),
        num_txs: u16::try_from(value.body().transactions.len()).expect("num txs u16"),
        num_l1_msgs: u16::try_from(
            value
                .body()
                .transactions
                .iter()
                .filter(|tx| tx.is_l1_message())
                .count(),
        )
        .expect("num l1 msgs u16"),
    }
}

type Witness = ArchivedChunkWitness;

pub fn execute(witness: &Witness) -> Result<ChunkInfo, String> {
    if witness.blocks.is_empty() {
        return Err("At least one witness must be provided in chunk mode".into());
    }
    if !witness.blocks.has_same_chain_id() {
        return Err("All witnesses must have the same chain id in chunk mode".into());
    }
    if !witness.blocks.has_seq_block_number() {
        return Err("All witnesses must have sequential block numbers in chunk mode".into());
    }
    // Get the blocks to build the basic chunk-info.
    let blocks = manually_drop_on_zkvm!(
        witness
            .blocks
            .iter()
            .map(|w| w.build_reth_block())
            .collect::<Result<Vec<RecoveredBlock<Block>>, _>>()
            .map_err(|e| e.to_string())?
    );
    let pre_state_root = witness.blocks[0].pre_state_root;

    let fork_name = ForkName::from(&witness.fork_name);
    let chain = Chain::from_id(witness.blocks[0].chain_id());

    // SCROLL_DEV_HARDFORKS will enable all forks
    let mut hardforks = (*SCROLL_DEV_HARDFORKS).clone();
    if fork_name == ForkName::EuclidV1 {
        // disable EuclidV2 fork for legacy chunk
        use sbv_primitives::{chainspec::ForkCondition, hardforks::ScrollHardfork};
        hardforks.insert(ScrollHardfork::EuclidV2, ForkCondition::Never);
    }

    let inner = ChainSpec {
        chain,
        genesis_hash: Default::default(),
        genesis: Default::default(),
        genesis_header: Default::default(),
        paris_block_and_final_difficulty: Default::default(),
        hardforks,
        deposit_contract: Default::default(),
        base_fee_params: BaseFeeParamsKind::Constant(BaseFeeParams::ethereum()),
        prune_delete_limit: 20000,
        blob_params: Default::default(),
    };
    let config = ScrollChainConfig::mainnet();
    let chain_spec: ScrollChainSpec = ScrollChainSpec { inner, config };

    let (code_db, nodes_provider, block_hashes) = make_providers(&witness.blocks);
    let nodes_provider = manually_drop_on_zkvm!(nodes_provider);

    let prev_state_root = witness.blocks[0].pre_state_root();
    let mut db = manually_drop_on_zkvm!(
        EvmDatabase::new_from_root(code_db, prev_state_root, &nodes_provider, block_hashes)
            .map_err(|e| format!("failed to create EvmDatabase: {}", e))?
    );
    for block in blocks.iter() {
        let output = manually_drop_on_zkvm!(
            EvmExecutor::new(std::sync::Arc::new(chain_spec.clone()), &db, block)
                .execute()
                .map_err(|e| format!("failed to execute block: {}", e))?
        );
        db.update(&nodes_provider, output.state.state.iter())
            .map_err(|e| format!("failed to update db: {}", e))?;
    }

    let post_state_root = db.commit_changes();

    let withdraw_root = db
        .withdraw_root()
        .map_err(|e| format!("failed to get withdraw root: {}", e))?;

    let mut rlp_buffer = manually_drop_on_zkvm!(Vec::with_capacity(2048));
    let (tx_data_length, tx_data_digest) = blocks
        .iter()
        .flat_map(|b| b.body().transactions.iter())
        .tx_bytes_hash_in(rlp_buffer.as_mut());
    let _ = tx_data_length;

    let sbv_chunk_info = {
        #[allow(unused_mut)]
        let mut builder = ChunkInfoBuilder::new(&chain_spec, pre_state_root.into(), &blocks);
        if fork_name == ForkName::EuclidV2 {
            builder.set_prev_msg_queue_hash(witness.prev_msg_queue_hash.into());
        }
        builder.build(withdraw_root)
    };
    if post_state_root != sbv_chunk_info.post_state_root() {
        return Err(format!(
            "state root mismatch: expected={}, found={}",
            sbv_chunk_info.post_state_root(),
            post_state_root
        ));
    }

    let chunk_info = ChunkInfo {
        chain_id: sbv_chunk_info.chain_id(),
        prev_state_root: sbv_chunk_info.prev_state_root(),
        post_state_root: sbv_chunk_info.post_state_root(),
        data_hash: sbv_chunk_info
            .clone()
            .into_legacy()
            .map(|x| x.data_hash)
            .unwrap_or_default(),
        withdraw_root,
        tx_data_digest,
        tx_data_length: u64::try_from(tx_data_length).expect("tx_data_length: u64"),
        initial_block_number: blocks[0].header().number,
        prev_msg_queue_hash: witness.prev_msg_queue_hash.into(),
        post_msg_queue_hash: sbv_chunk_info
            .into_euclid_v2()
            .map(|x| x.post_msg_queue_hash)
            .unwrap_or_default(),
        block_ctxs: blocks.iter().map(block_ctxv2_from_block).collect(),
    };

    openvm::io::println(format!("withdraw_root = {:?}", withdraw_root));
    openvm::io::println(format!("tx_bytes_hash = {:?}", tx_data_digest));

    // We should never touch that lazy lock... Or else we introduce 40M useless cycles.
    assert!(std::sync::LazyLock::get(&MAINNET).is_none());

    Ok(chunk_info)
}
