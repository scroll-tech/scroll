use crate::proofs::ChunkProof;
use c_kzg::Bytes48;
use eyre::Result;
use sbv_primitives::{B256, U256};
use scroll_zkvm_types::{
    batch::{
        BatchHeader, BatchHeaderV6, BatchHeaderV7, BatchInfo, BatchWitness, EnvelopeV6, EnvelopeV7,
        PointEvalWitness, ReferenceHeader, ToArchievedWitness, N_BLOB_BYTES,
    },
    public_inputs::ForkName,
    task::ProvingTask,
    utils::{to_rkyv_bytes, RancorError},
};

mod utils;
use utils::{base64, point_eval};

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

impl TryFrom<BatchProvingTask> for ProvingTask {
    type Error = eyre::Error;

    fn try_from(value: BatchProvingTask) -> Result<Self> {
        let witness = value.build_guest_input();

        Ok(ProvingTask {
            identifier: value.batch_header.batch_hash().to_string(),
            fork_name: value.fork_name,
            aggregated_proofs: value
                .chunk_proofs
                .into_iter()
                .map(|w_proof| w_proof.proof.into_root_proof().expect("expect root proof"))
                .collect(),
            serialized_witness: vec![to_rkyv_bytes::<RancorError>(&witness)?.into_vec()],
            vk: Vec::new(),
        })
    }
}

impl BatchProvingTask {
    fn build_guest_input(&self) -> BatchWitness {
        let fork_name = self.fork_name.to_lowercase().as_str().into();

        // sanity check: calculate point eval needed and compare with task input
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
                    match fork_name {
                        ForkName::EuclidV2 => (),
                        _ => unreachable!("hardfork mismatch for da-codec@v6 header: found={fork_name:?}, expected={:?}",
                                [ForkName::EuclidV2],
                            ),
                    }
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

        BatchWitness {
            fork_name,
            chunk_proofs: self.chunk_proofs.iter().map(|proof| proof.into()).collect(),
            chunk_infos: self
                .chunk_proofs
                .iter()
                .map(|p| p.metadata.chunk_info.clone())
                .collect(),
            blob_bytes: self.blob_bytes.clone(),
            reference_header,
            point_eval_witness,
        }
    }

    pub fn precheck_and_build_metadata(&self) -> Result<BatchInfo> {
        let fork_name = ForkName::from(self.fork_name.as_str());
        // for every aggregation task, there are two steps needed to build the metadata:
        // 1. generate data for metadata from the witness
        // 2. validate every adjacent proof pair
        let witness = self.build_guest_input();
        let archieved = ToArchievedWitness::create(&witness)
            .map_err(|e| eyre::eyre!("archieve batch witness fail: {e}"))?;
        let archieved_witness = archieved
            .access()
            .map_err(|e| eyre::eyre!("access archieved batch witness fail: {e}"))?;
        let metadata: BatchInfo = archieved_witness.into();

        super::check_aggregation_proofs(self.chunk_proofs.as_slice(), fork_name)?;

        Ok(metadata)
    }
}
