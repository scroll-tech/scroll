use alloy_primitives::{B256, U256};
use c_kzg::Bytes48;
use openvm_native_recursion::hints::Hintable;
use openvm_sdk::StdIn;
use scroll_zkvm_circuit_input_types::{
    batch::{
        BatchHeader, BatchHeaderV6, BatchHeaderV7, BatchInfo, BatchWitness, EnvelopeV6, EnvelopeV7,
        N_BLOB_BYTES, PointEvalWitness, ReferenceHeader,
    },
    public_inputs::ForkName,
};

use crate::{
    ChunkProof,
    task::{ProvingTask, flatten_wrapped_proof},
    utils::{base64, point_eval},
};

/// Define variable batch header type, since BatchHeaderV6 can not
/// be decoded as V7 we can always has correct deserialization
/// Notice: V6 header MUST be put above V7 since untagged enum
/// try to decode each defination in order
#[derive(Clone, serde::Deserialize, serde::Serialize)]
#[serde(untagged)]
pub enum BatchHeaderV {
    V6(BatchHeaderV6),
    V7(BatchHeaderV7),
}

impl From<BatchHeaderV> for ReferenceHeader {
    fn from(value: BatchHeaderV) -> Self {
        match value {
            BatchHeaderV::V6(h) => ReferenceHeader::V6(h),
            BatchHeaderV::V7(h) => ReferenceHeader::V7(h),
        }
    }
}

impl BatchHeaderV {
    pub fn batch_hash(&self) -> B256 {
        match self {
            BatchHeaderV::V6(h) => h.batch_hash(),
            BatchHeaderV::V7(h) => h.batch_hash(),
        }
    }

    pub fn must_v6_header(&self) -> &BatchHeaderV6 {
        match self {
            BatchHeaderV::V6(h) => h,
            BatchHeaderV::V7(_) => panic!("try to pick v7 header"),
        }
    }

    pub fn must_v7_header(&self) -> &BatchHeaderV7 {
        match self {
            BatchHeaderV::V7(h) => h,
            BatchHeaderV::V6(_) => panic!("try to pick v6 header"),
        }
    }
}

/// Defines a proving task for batch proof generation, the format
/// is compatible with both pre-euclidv2 and euclidv2
#[derive(Clone, serde::Deserialize, serde::Serialize)]
pub struct BatchProvingTask {
    /// Chunk proofs for the contiguous list of chunks within the batch.
    pub chunk_proofs: Vec<ChunkProof>,
    /// The [`BatchHeaderV6/V7`], as computed on-chain for this batch.
    pub batch_header: BatchHeaderV,
    /// The bytes encoding the batch data that will finally be published on-chain in the form of an
    /// EIP-4844 blob.
    #[serde(with = "base64")]
    pub blob_bytes: Vec<u8>,
    /// Challenge digest computed using the blob's bytes and versioned hash.
    pub challenge_digest: Option<U256>,
    /// KZG commitment for the blob.
    pub kzg_commitment: Option<Bytes48>,
    /// KZG proof.
    pub kzg_proof: Option<Bytes48>,
    /// fork version specify, for sanity check with batch_header and chunk proof
    pub fork_name: String,
}

impl ProvingTask for BatchProvingTask {
    fn identifier(&self) -> String {
        self.batch_header.batch_hash().to_string()
    }

    fn fork_name(&self) -> ForkName {
        ForkName::from(self.fork_name.as_str())
    }

    fn build_guest_input(&self) -> Result<StdIn, rkyv::rancor::Error> {
        let fork_name = self.fork_name.to_lowercase().as_str().into();

        // calculate point eval needed and compare with task input
        let (kzg_commitment, kzg_proof, challenge_digest) = {
            let blob = point_eval::to_blob(&self.blob_bytes);
            let commitment = point_eval::blob_to_kzg_commitment(&blob);
            let versioned_hash = point_eval::get_versioned_hash(&commitment);
            let challenge_digest = match &self.batch_header {
                BatchHeaderV::V6(_) => {
                    assert_eq!(
                        fork_name,
                        ForkName::EuclidV1,
                        "hardfork mismatch for da-codec@v6 header: found={fork_name:?}, expected={:?}",
                        ForkName::EuclidV1,
                    );
                    EnvelopeV6::from(self.blob_bytes.as_slice()).challenge_digest(versioned_hash)
                }
                BatchHeaderV::V7(_) => {
                    assert_eq!(
                        fork_name,
                        ForkName::EuclidV2,
                        "hardfork mismatch for da-codec@v7 header: found={fork_name:?}, expected={:?}",
                        ForkName::EuclidV2,
                    );
                    let padded_blob_bytes = {
                        let mut padded_blob_bytes = self.blob_bytes.to_vec();
                        padded_blob_bytes.resize(N_BLOB_BYTES, 0);
                        padded_blob_bytes
                    };
                    EnvelopeV7::from(padded_blob_bytes.as_slice()).challenge_digest(versioned_hash)
                }
            };

            let (proof, _) = point_eval::get_kzg_proof(&blob, challenge_digest);

            (commitment.to_bytes(), proof.to_bytes(), challenge_digest)
        };

        if let Some(k) = self.kzg_commitment {
            assert_eq!(k, kzg_commitment);
        }

        if let Some(c) = self.challenge_digest {
            assert_eq!(c, U256::from_be_bytes(challenge_digest.0));
        }

        if let Some(p) = self.kzg_proof {
            assert_eq!(p, kzg_proof);
        }

        let point_eval_witness = PointEvalWitness {
            kzg_commitment: kzg_commitment.into_inner(),
            kzg_proof: kzg_proof.into_inner(),
        };

        let reference_header = self.batch_header.clone().into();

        let witness = BatchWitness {
            fork_name,
            chunk_proofs: self
                .chunk_proofs
                .iter()
                .map(flatten_wrapped_proof)
                .collect(),
            chunk_infos: self
                .chunk_proofs
                .iter()
                .map(|p| p.metadata.chunk_info.clone())
                .collect(),
            blob_bytes: self.blob_bytes.clone(),
            reference_header,
            point_eval_witness,
        };

        let serialized = rkyv::to_bytes::<rkyv::rancor::Error>(&witness)?;
        let mut stdin = StdIn::default();
        stdin.write_bytes(&serialized);
        for chunk_proof in &self.chunk_proofs {
            let root_input = &chunk_proof.as_proof();
            let streams = root_input.write();
            for s in &streams {
                stdin.write_field(s);
            }
        }
        Ok(stdin)
    }
}

impl From<&BatchProvingTask> for BatchInfo {
    fn from(task: &BatchProvingTask) -> Self {
        let fork_name = ForkName::from(task.fork_name.as_str());
        let (parent_state_root, state_root, chain_id, withdraw_root) = (
            task.chunk_proofs
                .first()
                .expect("at least one chunk in batch")
                .metadata
                .chunk_info
                .prev_state_root,
            task.chunk_proofs
                .last()
                .expect("at least one chunk in batch")
                .metadata
                .chunk_info
                .post_state_root,
            task.chunk_proofs
                .last()
                .expect("at least one chunk in batch")
                .metadata
                .chunk_info
                .chain_id,
            task.chunk_proofs
                .last()
                .expect("at least one chunk in batch")
                .metadata
                .chunk_info
                .withdraw_root,
        );
        let (parent_batch_hash, prev_msg_queue_hash, post_msg_queue_hash) = match task.batch_header
        {
            BatchHeaderV::V6(h) => {
                assert_eq!(
                    fork_name,
                    ForkName::EuclidV1,
                    "hardfork mismatch for da-codec@v6 header: found={fork_name:?}, expected={:?}",
                    ForkName::EuclidV1,
                );
                (h.parent_batch_hash, Default::default(), Default::default())
            }
            BatchHeaderV::V7(h) => {
                assert_eq!(
                    fork_name,
                    ForkName::EuclidV2,
                    "hardfork mismatch for da-codec@v7 header: found={fork_name:?}, expected={:?}",
                    ForkName::EuclidV2,
                );
                (
                    h.parent_batch_hash,
                    task.chunk_proofs
                        .first()
                        .expect("at least one chunk in batch")
                        .metadata
                        .chunk_info
                        .prev_msg_queue_hash,
                    task.chunk_proofs
                        .last()
                        .expect("at least one chunk in batch")
                        .metadata
                        .chunk_info
                        .post_msg_queue_hash,
                )
            }
        };

        let batch_hash = task.batch_header.batch_hash();

        Self {
            parent_state_root,
            parent_batch_hash,
            state_root,
            batch_hash,
            chain_id,
            withdraw_root,
            prev_msg_queue_hash,
            post_msg_queue_hash,
        }
    }
}
