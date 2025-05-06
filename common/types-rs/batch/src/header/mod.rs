use alloy_primitives::B256;

pub mod v6;

pub mod v7;

pub trait BatchHeader {
    /// The DA-codec version for the batch header.
    fn version(&self) -> u8;

    /// The incremental index of the batch.
    fn index(&self) -> u64;

    /// The batch header digest of the parent batch.
    fn parent_batch_hash(&self) -> B256;

    /// The batch header digest.
    fn batch_hash(&self) -> B256;
}

/// Reference header indicate the version of batch header base on which batch hash
/// should be calculated.
#[derive(Clone, Debug, rkyv::Archive, rkyv::Deserialize, rkyv::Serialize)]
#[rkyv(derive(Debug))]
pub enum ReferenceHeader {
    /// Represents DA-codec v6.
    V6(v6::BatchHeaderV6),
    /// Represents DA-codec v7.
    V7(v7::BatchHeaderV7),
}
