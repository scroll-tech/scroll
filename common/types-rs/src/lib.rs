// re-export for a compatible interface with old circuit/types for prover

pub mod bundle {
    pub use types_base::public_inputs::bundle::{BundleInfo, BundleInfoV1, BundleInfoV2};
    pub use types_bundle::*;
}

pub mod batch {
    pub use types_base::public_inputs::batch::{ArchivedBatchInfo, BatchInfo, VersionedBatchInfo};
    pub use types_batch::*;
}

pub mod chunk {
    pub use types_base::public_inputs::chunk::{
        ArchivedChunkInfo, BlockContextV2, ChunkInfo, SIZE_BLOCK_CTX, VersionedChunkInfo,
    };
    pub use types_chunk::*;
}

pub use types_agg;
pub use types_base::{public_inputs, utils};
