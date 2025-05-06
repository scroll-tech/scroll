use alloy_primitives::B256;

use crate::{
    public_inputs::{ForkName, MultiVersionPublicInputs, PublicInputs},
    utils::keccak256,
};

/// Represents fields required to compute the public-inputs digest of a bundle.
#[derive(Clone, Debug, serde::Deserialize, serde::Serialize)]
pub struct BundleInfo {
    /// The EIP-155 chain ID of all txs in the bundle.
    pub chain_id: u64,
    /// The L1 msg queue hash at the end of the last batch in the bundle.
    /// Not a phase 1 field so we make it omitable
    #[serde(default)]
    pub msg_queue_hash: B256,
    /// The number of batches bundled together in the bundle.
    pub num_batches: u32,
    /// The last finalized on-chain state root.
    pub prev_state_root: B256,
    /// The last finalized on-chain batch hash.
    pub prev_batch_hash: B256,
    /// The state root after applying every batch in the bundle.
    ///
    /// Upon verification of the EVM-verifiable bundle proof, this state root will be finalized
    /// on-chain.
    pub post_state_root: B256,
    /// The batch hash of the last batch in the bundle.
    ///
    /// Upon verification of the EVM-verifiable bundle proof, this batch hash will be finalized
    /// on-chain.
    pub batch_hash: B256,
    /// The withdrawals root at the last block in the last chunk in the last batch in the bundle.
    pub withdraw_root: B256,
}

impl BundleInfo {
    /// Public input hash for a bundle (euclidv1 or da-codec@v6) is defined as
    ///
    /// keccak(
    ///     chain id ||
    ///     num batches ||
    ///     prev state root ||
    ///     prev batch hash ||
    ///     post state root ||
    ///     batch hash ||
    ///     withdraw root
    /// )
    pub fn pi_hash_euclidv1(&self) -> B256 {
        keccak256(
            std::iter::empty()
                .chain(self.chain_id.to_be_bytes().as_slice())
                .chain(self.num_batches.to_be_bytes().as_slice())
                .chain(self.prev_state_root.as_slice())
                .chain(self.prev_batch_hash.as_slice())
                .chain(self.post_state_root.as_slice())
                .chain(self.batch_hash.as_slice())
                .chain(self.withdraw_root.as_slice())
                .cloned()
                .collect::<Vec<u8>>(),
        )
    }

    /// Public input hash for a bundle (euclidv2 or da-codec@v7) is defined as
    ///
    /// keccak(
    ///     chain id ||
    ///     msg_queue_hash ||
    ///     num batches ||
    ///     prev state root ||
    ///     prev batch hash ||
    ///     post state root ||
    ///     batch hash ||
    ///     withdraw root
    /// )   
    pub fn pi_hash_euclidv2(&self) -> B256 {
        keccak256(
            std::iter::empty()
                .chain(self.chain_id.to_be_bytes().as_slice())
                .chain(self.msg_queue_hash.as_slice())
                .chain(self.num_batches.to_be_bytes().as_slice())
                .chain(self.prev_state_root.as_slice())
                .chain(self.prev_batch_hash.as_slice())
                .chain(self.post_state_root.as_slice())
                .chain(self.batch_hash.as_slice())
                .chain(self.withdraw_root.as_slice())
                .cloned()
                .collect::<Vec<u8>>(),
        )
    }

    pub fn pi_hash(&self, fork_name: ForkName) -> B256 {
        match fork_name {
            ForkName::EuclidV1 => self.pi_hash_euclidv1(),
            ForkName::EuclidV2 => self.pi_hash_euclidv2(),
        }
    }
}

impl MultiVersionPublicInputs for BundleInfo {
    fn pi_hash_by_fork(&self, fork_name: ForkName) -> B256 {
        match fork_name {
            ForkName::EuclidV1 => self.pi_hash_euclidv1(),
            ForkName::EuclidV2 => self.pi_hash_euclidv2(),
        }
    }

    fn validate(&self, _prev_pi: &Self, _fork_name: ForkName) {
        unreachable!("bundle is the last layer and is not aggregated by any other circuit");
    }
}

#[derive(Clone, Debug)]
pub struct BundleInfoV1(pub BundleInfo);

#[derive(Clone, Debug)]
pub struct BundleInfoV2(pub BundleInfo);

impl From<BundleInfo> for BundleInfoV1 {
    fn from(value: BundleInfo) -> Self {
        Self(value)
    }
}

impl From<BundleInfo> for BundleInfoV2 {
    fn from(value: BundleInfo) -> Self {
        Self(value)
    }
}

impl PublicInputs for BundleInfoV1 {
    fn pi_hash(&self) -> B256 {
        self.0.pi_hash_euclidv1()
    }

    fn validate(&self, _prev_pi: &Self) {
        unreachable!("bundle is the last layer and is not aggregated by any other circuit");
    }
}

impl PublicInputs for BundleInfoV2 {
    fn pi_hash(&self) -> B256 {
        self.0.pi_hash_euclidv2()
    }

    fn validate(&self, _prev_pi: &Self) {
        unreachable!("bundle is the last layer and is not aggregated by any other circuit");
    }
}
