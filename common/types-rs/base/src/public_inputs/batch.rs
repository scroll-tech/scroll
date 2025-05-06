use alloy_primitives::B256;

use crate::{
    public_inputs::{ForkName, MultiVersionPublicInputs},
    utils::keccak256,
};

/// Represents public-input values for a batch.
#[derive(
    Clone,
    Debug,
    rkyv::Archive,
    rkyv::Deserialize,
    rkyv::Serialize,
    serde::Deserialize,
    serde::Serialize,
)]
#[rkyv(derive(Debug))]
pub struct BatchInfo {
    /// The state root before applying the batch.
    #[rkyv()]
    pub parent_state_root: B256,
    /// The batch hash of the parent batch.
    #[rkyv()]
    pub parent_batch_hash: B256,
    /// The state root after applying txs in the batch.
    #[rkyv()]
    pub state_root: B256,
    /// The batch header hash of the batch.
    #[rkyv()]
    pub batch_hash: B256,
    /// The EIP-155 chain ID of all txs in the batch.
    #[rkyv()]
    pub chain_id: u64,
    /// The withdraw root of the last block in the last chunk in the batch.
    #[rkyv()]
    pub withdraw_root: B256,
    /// The L1 msg queue hash at the end of the previous batch.
    #[rkyv()]
    pub prev_msg_queue_hash: B256,
    /// The L1 msg queue hash at the end of the current batch.
    #[rkyv()]
    pub post_msg_queue_hash: B256,
}

impl From<&ArchivedBatchInfo> for BatchInfo {
    fn from(archived: &ArchivedBatchInfo) -> Self {
        Self {
            parent_state_root: archived.parent_state_root.into(),
            parent_batch_hash: archived.parent_batch_hash.into(),
            state_root: archived.state_root.into(),
            batch_hash: archived.batch_hash.into(),
            chain_id: archived.chain_id.into(),
            withdraw_root: archived.withdraw_root.into(),
            prev_msg_queue_hash: archived.prev_msg_queue_hash.into(),
            post_msg_queue_hash: archived.post_msg_queue_hash.into(),
        }
    }
}

impl BatchInfo {
    /// Public input hash for a batch (euclidv1 or da-codec@v6) is defined as
    ///
    /// keccak(
    ///     parent state root ||
    ///     parent batch hash ||
    ///     state root ||
    ///     batch hash ||
    ///     chain id ||
    ///     withdraw root ||
    /// )
    fn pi_hash_euclidv1(&self) -> B256 {
        keccak256(
            std::iter::empty()
                .chain(self.parent_state_root.as_slice())
                .chain(self.parent_batch_hash.as_slice())
                .chain(self.state_root.as_slice())
                .chain(self.batch_hash.as_slice())
                .chain(self.chain_id.to_be_bytes().as_slice())
                .chain(self.withdraw_root.as_slice())
                .cloned()
                .collect::<Vec<u8>>(),
        )
    }

    /// Public input hash for a batch (euclidv2 or da-codec@v7) is defined as
    ///
    /// keccak(
    ///     parent state root ||
    ///     parent batch hash ||
    ///     state root ||
    ///     batch hash ||
    ///     chain id ||
    ///     withdraw root ||
    ///     prev msg queue hash ||
    ///     post msg queue hash
    /// )    
    fn pi_hash_euclidv2(&self) -> B256 {
        keccak256(
            std::iter::empty()
                .chain(self.parent_state_root.as_slice())
                .chain(self.parent_batch_hash.as_slice())
                .chain(self.state_root.as_slice())
                .chain(self.batch_hash.as_slice())
                .chain(self.chain_id.to_be_bytes().as_slice())
                .chain(self.withdraw_root.as_slice())
                .chain(self.prev_msg_queue_hash.as_slice())
                .chain(self.post_msg_queue_hash.as_slice())
                .cloned()
                .collect::<Vec<u8>>(),
        )
    }
}

pub type VersionedBatchInfo = (BatchInfo, ForkName);

impl MultiVersionPublicInputs for BatchInfo {
    fn pi_hash_by_fork(&self, fork_name: ForkName) -> B256 {
        match fork_name {
            ForkName::EuclidV1 => self.pi_hash_euclidv1(),
            ForkName::EuclidV2 => self.pi_hash_euclidv2(),
        }
    }

    /// Validate public inputs between 2 contiguous batches.
    ///
    /// - chain id MUST match
    /// - state roots MUST be chained
    /// - batch hashes MUST be chained
    /// - L1 msg queue hashes MUST be chained
    fn validate(&self, prev_pi: &Self, fork_name: ForkName) {
        assert_eq!(self.chain_id, prev_pi.chain_id);
        assert_eq!(self.parent_state_root, prev_pi.state_root);
        assert_eq!(self.parent_batch_hash, prev_pi.batch_hash);
        assert_eq!(self.prev_msg_queue_hash, prev_pi.post_msg_queue_hash);

        if fork_name == ForkName::EuclidV1 {
            assert_eq!(self.prev_msg_queue_hash, B256::ZERO);
            assert_eq!(prev_pi.prev_msg_queue_hash, B256::ZERO);
            assert_eq!(self.post_msg_queue_hash, B256::ZERO);
            assert_eq!(prev_pi.post_msg_queue_hash, B256::ZERO);
        }
    }
}
