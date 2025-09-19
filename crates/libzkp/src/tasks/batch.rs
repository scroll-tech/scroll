use c_kzg::Bytes48;
use eyre::Result;
use sbv_primitives::{B256, U256};
use scroll_zkvm_types::{
    batch::{
        build_point_eval_witness, BatchHeader, BatchHeaderV6, BatchHeaderV7, BatchHeaderV8,
        BatchHeaderValidium, BatchInfo, BatchWitness, Envelope, EnvelopeV6, EnvelopeV7, EnvelopeV8,
        LegacyBatchWitness, ReferenceHeader, N_BLOB_BYTES,
    },
    public_inputs::{ForkName, Version},
    task::ProvingTask,
    utils::{to_rkyv_bytes, RancorError},
    version::{Domain, STFVersion},
};

use crate::proofs::ChunkProof;

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
    V7_8(BatchHeaderV7),
    Validium(BatchHeaderValidium),
}

impl BatchHeaderV {
    pub fn batch_hash(&self) -> B256 {
        match self {
            BatchHeaderV::V6(h) => h.batch_hash(),
            BatchHeaderV::V7_8(h) => h.batch_hash(),
            BatchHeaderV::Validium(h) => h.batch_hash(),
        }
    }

    pub fn must_v6_header(&self) -> &BatchHeaderV6 {
        match self {
            BatchHeaderV::V6(h) => h,
            _ => panic!("try to pick other header type"),
        }
    }

    pub fn must_v7_header(&self) -> &BatchHeaderV7 {
        match self {
            BatchHeaderV::V7_8(h) => h,
            _ => panic!("try to pick other header type"),
        }
    }

    pub fn must_v8_header(&self) -> &BatchHeaderV8 {
        match self {
            BatchHeaderV::V7_8(h) => h,
            _ => panic!("try to pick other header type"),
        }
    }

    pub fn must_validium_header(&self) -> &BatchHeaderValidium {
        match self {
            BatchHeaderV::Validium(h) => h,
            _ => panic!("try to pick other header type"),
        }
    }
}

/// Defines a proving task for batch proof generation, the format
/// is compatible with both pre-euclidv2 and euclidv2
#[derive(Clone, serde::Deserialize, serde::Serialize)]
pub struct BatchProvingTask {
    /// The version of the chunks in the batch, as per [`Version`].
    pub version: u8,
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
        let serialized_witness = if crate::witness_use_legacy_mode() {
            let legacy_witness = LegacyBatchWitness::from(witness);
            to_rkyv_bytes::<RancorError>(&legacy_witness)?.into_vec()
        } else {
            super::encode_task_to_witness(&witness)?
        };

        Ok(ProvingTask {
            identifier: value.batch_header.batch_hash().to_string(),
            fork_name: value.fork_name,
            aggregated_proofs: value
                .chunk_proofs
                .into_iter()
                .map(|w_proof| w_proof.proof.into_stark_proof().expect("expect root proof"))
                .collect(),
            serialized_witness: vec![serialized_witness],
            vk: Vec::new(),
        })
    }
}

impl BatchProvingTask {
    fn build_guest_input(&self) -> BatchWitness {
        let version = Version::from(self.version);

        let point_eval_witness = if !version.is_validium() {
            // sanity check: calculate point eval needed and compare with task input
            let (kzg_commitment, kzg_proof, challenge_digest) = {
                let blob = point_eval::to_blob(&self.blob_bytes);
                let commitment = point_eval::blob_to_kzg_commitment(&blob);
                let versioned_hash = point_eval::get_versioned_hash(&commitment);
                let challenge_digest = match &self.batch_header {
                    BatchHeaderV::V6(_) => {
                        assert_eq!(
                            version.fork,
                            ForkName::EuclidV1,
                            "hardfork mismatch for da-codec@v6 header: found={:?}, expected={:?}",
                            version.fork,
                            ForkName::EuclidV1,
                        );
                        EnvelopeV6::from_slice(self.blob_bytes.as_slice())
                            .challenge_digest(versioned_hash)
                    }
                    BatchHeaderV::V7_8(_) => {
                        let padded_blob_bytes = {
                            let mut padded_blob_bytes = self.blob_bytes.to_vec();
                            padded_blob_bytes.resize(N_BLOB_BYTES, 0);
                            padded_blob_bytes
                        };

                        match version.fork {
                            ForkName::EuclidV2 => {
                                <EnvelopeV7 as Envelope>::from_slice(padded_blob_bytes.as_slice())
                                    .challenge_digest(versioned_hash)
                            }
                            ForkName::Feynman => {
                                <EnvelopeV8 as Envelope>::from_slice(padded_blob_bytes.as_slice())
                                    .challenge_digest(versioned_hash)
                            }
                            fork_name => unreachable!(
                                "hardfork mismatch for da-codec@v7 header: found={}, expected={:?}",
                                fork_name,
                                [ForkName::EuclidV2, ForkName::Feynman],
                            ),
                        }
                    }
                    BatchHeaderV::Validium(_) => unreachable!("version!=validium"),
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

            Some(build_point_eval_witness(
                kzg_commitment.into_inner(),
                kzg_proof.into_inner(),
            ))
        } else {
            assert!(self.kzg_proof.is_none(), "domain=validium has no blob-da");
            assert!(
                self.kzg_commitment.is_none(),
                "domain=validium has no blob-da"
            );
            assert!(
                self.challenge_digest.is_none(),
                "domain=validium has no blob-da"
            );
            None
        };

        let reference_header = match (version.domain, version.stf_version) {
            (Domain::Scroll, STFVersion::V6) => {
                ReferenceHeader::V6(*self.batch_header.must_v6_header())
            }
            (Domain::Scroll, STFVersion::V7) => {
                ReferenceHeader::V7(*self.batch_header.must_v7_header())
            }
            (Domain::Scroll, STFVersion::V8) => {
                ReferenceHeader::V8(*self.batch_header.must_v8_header())
            }
            (Domain::Validium, STFVersion::V1) => {
                ReferenceHeader::Validium(*self.batch_header.must_validium_header())
            }
            (domain, stf_version) => {
                unreachable!("unsupported domain={domain:?},stf-version={stf_version:?}")
            }
        };

        BatchWitness {
            version: version.as_version_byte(),
            fork_name: version.fork,
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
        // for every aggregation task, there are two steps needed to build the metadata:
        // 1. generate data for metadata from the witness
        // 2. validate every adjacent proof pair
        let witness = self.build_guest_input();
        let metadata = BatchInfo::from(&witness);
        super::check_aggregation_proofs(self.chunk_proofs.as_slice(), Version::from(self.version))?;

        Ok(metadata)
    }
}
