use alloy_primitives::{B256, U256};
use sbv_primitives::types::{
    consensus::BlockHeader,
    reth::{Block, RecoveredBlock},
};

use types_base::public_inputs::chunk::BlockContextV2;

impl From<&RecoveredBlock<Block>> for BlockContextV2 {
    fn from(value: &RecoveredBlock<Block>) -> Self {
        Self {
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
}
