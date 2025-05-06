use super::BatchHeader;
use alloy_primitives::B256;
use types_base::utils::keccak256;

/// Represents the header summarising the batch of chunks as per DA-codec v6.
#[derive(
    Clone,
    Copy,
    Debug,
    Default,
    rkyv::Archive,
    rkyv::Deserialize,
    rkyv::Serialize,
    serde::Deserialize,
    serde::Serialize,
)]
#[rkyv(derive(Debug))]
pub struct BatchHeaderV6 {
    /// The DA-codec version for the batch.
    #[rkyv()]
    pub version: u8,
    /// The index of the batch
    #[rkyv()]
    pub batch_index: u64,
    /// Number of L1 messages popped in the batch
    #[rkyv()]
    pub l1_message_popped: u64,
    /// Number of total L1 messages popped after the batch
    #[rkyv()]
    pub total_l1_message_popped: u64,
    /// The parent batch hash
    #[rkyv()]
    pub parent_batch_hash: B256,
    /// The timestamp of the last block in this batch
    #[rkyv()]
    pub last_block_timestamp: u64,
    /// The data hash of the batch
    #[rkyv()]
    pub data_hash: B256,
    /// The versioned hash of the blob with this batch's data
    #[rkyv()]
    pub blob_versioned_hash: B256,
    /// The blob data proof: z (32), y (32)
    #[rkyv()]
    pub blob_data_proof: [B256; 2],
}

impl BatchHeader for BatchHeaderV6 {
    fn version(&self) -> u8 {
        self.version
    }

    fn index(&self) -> u64 {
        self.batch_index
    }

    fn parent_batch_hash(&self) -> B256 {
        self.parent_batch_hash
    }

    /// Batch hash as per DA-codec v6:
    ///
    /// keccak(
    ///     version ||
    ///     batch index ||
    ///     l1 message popped ||
    ///     total l1 message popped ||
    ///     batch data hash ||
    ///     versioned hash ||
    ///     parent batch hash ||
    ///     last block timestamp ||
    ///     z ||
    ///     y
    /// )
    fn batch_hash(&self) -> B256 {
        keccak256(
            std::iter::empty()
                .chain(vec![self.version].as_slice())
                .chain(self.batch_index.to_be_bytes().as_slice())
                .chain(self.l1_message_popped.to_be_bytes().as_slice())
                .chain(self.total_l1_message_popped.to_be_bytes().as_slice())
                .chain(self.data_hash.as_slice())
                .chain(self.blob_versioned_hash.as_slice())
                .chain(self.parent_batch_hash.as_slice())
                .chain(self.last_block_timestamp.to_be_bytes().as_slice())
                .chain(self.blob_data_proof[0].as_slice())
                .chain(self.blob_data_proof[1].as_slice())
                .cloned()
                .collect::<Vec<u8>>(),
        )
    }
}

impl BatchHeader for ArchivedBatchHeaderV6 {
    fn version(&self) -> u8 {
        self.version
    }

    fn index(&self) -> u64 {
        self.batch_index.into()
    }

    fn parent_batch_hash(&self) -> B256 {
        self.parent_batch_hash.into()
    }

    fn batch_hash(&self) -> B256 {
        let batch_index: u64 = self.batch_index.into();
        let l1_message_popped: u64 = self.l1_message_popped.into();
        let total_l1_message_popped: u64 = self.total_l1_message_popped.into();
        let data_hash: B256 = self.data_hash.into();
        let blob_versioned_hash: B256 = self.blob_versioned_hash.into();
        let parent_batch_hash: B256 = self.parent_batch_hash.into();
        let last_block_timestamp: u64 = self.last_block_timestamp.into();
        let blob_data_proof: [B256; 2] = self.blob_data_proof.map(|h| h.into());
        keccak256(
            std::iter::empty()
                .chain(vec![self.version].as_slice())
                .chain(batch_index.to_be_bytes().as_slice())
                .chain(l1_message_popped.to_be_bytes().as_slice())
                .chain(total_l1_message_popped.to_be_bytes().as_slice())
                .chain(data_hash.as_slice())
                .chain(blob_versioned_hash.as_slice())
                .chain(parent_batch_hash.as_slice())
                .chain(last_block_timestamp.to_be_bytes().as_slice())
                .chain(blob_data_proof[0].as_slice())
                .chain(blob_data_proof[1].as_slice())
                .cloned()
                .collect::<Vec<u8>>(),
        )
    }
}

impl From<&ArchivedBatchHeaderV6> for BatchHeaderV6 {
    fn from(archived: &ArchivedBatchHeaderV6) -> Self {
        Self {
            version: archived.version,
            batch_index: archived.batch_index.into(),
            l1_message_popped: archived.l1_message_popped.into(),
            total_l1_message_popped: archived.total_l1_message_popped.into(),
            parent_batch_hash: archived.parent_batch_hash.into(),
            last_block_timestamp: archived.last_block_timestamp.into(),
            data_hash: archived.data_hash.into(),
            blob_versioned_hash: archived.blob_versioned_hash.into(),
            blob_data_proof: [
                archived.blob_data_proof[0].into(),
                archived.blob_data_proof[1].into(),
            ],
        }
    }
}
