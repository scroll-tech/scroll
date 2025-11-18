use c_kzg::Bytes48;
use eyre::Result;
use sbv_primitives::{B256, U256};
use scroll_zkvm_types::{
    batch::{
        build_point_eval_witness, BatchHeader, BatchHeaderV6, BatchHeaderV7, BatchHeaderV8,
        BatchHeaderValidium, BatchInfo, BatchWitness, Envelope, EnvelopeV6, EnvelopeV7, EnvelopeV8,
        LegacyBatchWitness, ReferenceHeader, N_BLOB_BYTES,
    },
    chunk::ChunkInfo,
    public_inputs::{ForkName, Version},
    task::ProvingTask,
    utils::{to_rkyv_bytes, RancorError},
    version::{Codec, Domain, STFVersion},
};

use crate::proofs::ChunkProof;

mod utils;
use utils::{base64, point_eval};

#[derive(Clone, serde::Deserialize, serde::Serialize)]
pub struct BatchHeaderValidiumWithHash {
    #[serde(flatten)]
    header: BatchHeaderValidium,
    batch_hash: B256,
}

/// Parse header types passed from golang side and adapt to the
/// defination in zkvm-prover's types
/// We distinguish the header type in golang side according to the codec
/// version, i.e. v6 - v9 (current), and validium
/// And adapt it to different header version used in zkvm-prover's witness
/// defination, i.e. v6- v8 (current), and validium
#[derive(Clone, serde::Deserialize, serde::Serialize)]
#[serde(untagged)]
pub enum BatchHeaderV {
    Validium(BatchHeaderValidiumWithHash),
    V6(BatchHeaderV6),
    V7_8_9(BatchHeaderV7),
}

impl core::fmt::Display for BatchHeaderV {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            BatchHeaderV::V6(_) => write!(f, "V6"),
            BatchHeaderV::V7_8_9(_) => write!(f, "V7_8_9"),
            BatchHeaderV::Validium(_) => write!(f, "Validium"),
        }
    }
}

impl BatchHeaderV {
    pub fn batch_hash(&self) -> B256 {
        match self {
            BatchHeaderV::V6(h) => h.batch_hash(),
            BatchHeaderV::V7_8_9(h) => h.batch_hash(),
            BatchHeaderV::Validium(h) => h.header.batch_hash(),
        }
    }

    pub fn must_v6_header(&self) -> &BatchHeaderV6 {
        match self {
            BatchHeaderV::V6(h) => h,
            _ => unreachable!("A header of {} is considered to be v6", self),
        }
    }

    pub fn must_v7_header(&self) -> &BatchHeaderV7 {
        match self {
            BatchHeaderV::V7_8_9(h) => h,
            _ => unreachable!("A header of {} is considered to be v7", self),
        }
    }

    pub fn must_v8_header(&self) -> &BatchHeaderV8 {
        match self {
            BatchHeaderV::V7_8_9(h) => h,
            _ => unreachable!("A header of {} is considered to be v8", self),
        }
    }

    pub fn must_validium_header(&self) -> &BatchHeaderValidium {
        match self {
            BatchHeaderV::Validium(h) => &h.header,
            _ => unreachable!("A header of {} is considered to be validium", self),
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
        let serialized_witness = if crate::witness_use_legacy_mode(&value.fork_name)? {
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
        tracing::info!(
            "Handling batch task for input, version byte {}, Version data: {:?}",
            self.version,
            version
        );
        // sanity check for if result of header type parsing match to version
        match &self.batch_header {
            BatchHeaderV::Validium(_) => assert!(
                version.is_validium(),
                "version {:?} is not match with parsed header, get validium header but version is not validium", version,
            ),
            BatchHeaderV::V6(_) => assert_eq!(version.fork, ForkName::EuclidV1,
                "hardfork mismatch for da-codec@v6 header: found={:?}, expected={:?}",
                version.fork,
                ForkName::EuclidV1,
            ),
            BatchHeaderV::V7_8_9(_) => assert!(
                version.fork == ForkName::EuclidV2 ||
                version.fork == ForkName::Feynman ||
                version.fork == ForkName::Galileo,
                "hardfork mismatch for da-codec@v7/8/9 header: found={}, expected={:?}",
                version.fork,
                [ForkName::EuclidV2, ForkName::Feynman, ForkName::Galileo],
            ),
        }

        let point_eval_witness = if !version.is_validium() {
            // sanity check: calculate point eval needed and compare with task input
            let (kzg_commitment, kzg_proof, challenge_digest) = {
                let blob = point_eval::to_blob(&self.blob_bytes);
                let commitment = point_eval::blob_to_kzg_commitment(&blob);
                let versioned_hash = point_eval::get_versioned_hash(&commitment);

                let padded_blob_bytes = {
                    let mut padded_blob_bytes = self.blob_bytes.to_vec();
                    padded_blob_bytes.resize(N_BLOB_BYTES, 0);
                    padded_blob_bytes
                };
                let challenge_digest = match version.codec {
                    Codec::V6 => {
                        // notice v6 do not use padded blob bytes
                        <EnvelopeV6 as Envelope>::from_slice(self.blob_bytes.as_slice())
                            .challenge_digest(versioned_hash)
                    }
                    Codec::V7 => <EnvelopeV7 as Envelope>::from_slice(padded_blob_bytes.as_slice())
                        .challenge_digest(versioned_hash),
                    Codec::V8 => <EnvelopeV8 as Envelope>::from_slice(padded_blob_bytes.as_slice())
                        .challenge_digest(versioned_hash),
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

            match &self.batch_header {
                BatchHeaderV::Validium(h) => assert_eq!(
                    h.header.batch_hash(),
                    h.batch_hash,
                    "calculated batch hash match which from coordinator"
                ),
                _ => panic!("unexpected header type"),
            }
            None
        };

        let reference_header = match (version.domain, version.stf_version) {
            (Domain::Scroll, STFVersion::V6) => {
                ReferenceHeader::V6(*self.batch_header.must_v6_header())
            }
            (Domain::Scroll, STFVersion::V7) => {
                ReferenceHeader::V7(*self.batch_header.must_v7_header())
            }
            (Domain::Scroll, STFVersion::V8) | (Domain::Scroll, STFVersion::V9) => {
                ReferenceHeader::V8(*self.batch_header.must_v8_header())
            }
            (Domain::Validium, STFVersion::V1) => {
                ReferenceHeader::Validium(*self.batch_header.must_validium_header())
            }
            (domain, stf_version) => {
                unreachable!("unsupported domain={domain:?},stf-version={stf_version:?}")
            }
        };

        // patch: ensure block_hash field is ZERO for scroll domain
        let chunk_infos = self
            .chunk_proofs
            .iter()
            .map(|p| {
                if version.domain == Domain::Scroll {
                    ChunkInfo {
                        prev_blockhash: B256::ZERO,
                        post_blockhash: B256::ZERO,
                        ..p.metadata.chunk_info.clone()
                    }
                } else {
                    p.metadata.chunk_info.clone()
                }
            })
            .collect();

        BatchWitness {
            version: version.as_version_byte(),
            fork_name: version.fork,
            chunk_proofs: self.chunk_proofs.iter().map(|proof| proof.into()).collect(),
            chunk_infos,
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
        super::check_aggregation_proofs(
            witness.chunk_infos.as_slice(),
            Version::from(self.version),
        )?;

        Ok(metadata)
    }
}

#[test]
fn test_deserde_batch_header_v_validium() {
    use std::str::FromStr;

    // Top-level JSON: flattened enum tag "V1" + batch_hash
    let json = r#"{
        "V1": {
        "version": 1,
        "batch_index": 42,
        "parent_batch_hash": "0x1111111111111111111111111111111111111111111111111111111111111111",
        "post_state_root": "0x2222222222222222222222222222222222222222222222222222222222222222",
        "withdraw_root": "0x3333333333333333333333333333333333333333333333333333333333333333",
        "commitment": "0x4444444444444444444444444444444444444444444444444444444444444444"
        },
        "batch_hash": "0x5555555555555555555555555555555555555555555555555555555555555555"
    }"#;

    let parsed: BatchHeaderV = serde_json::from_str(json).expect("deserialize BatchHeaderV");

    match parsed {
        BatchHeaderV::Validium(v) => {
            // Check the batch_hash field
            let expected_batch_hash = B256::from_str(
                "0x5555555555555555555555555555555555555555555555555555555555555555",
            )
            .unwrap();
            assert_eq!(v.batch_hash, expected_batch_hash);

            // Check the inner header variant and fields
            match v.header {
                BatchHeaderValidium::V1(h) => {
                    assert_eq!(h.version, 1);
                    assert_eq!(h.batch_index, 42);

                    let p = B256::from_str(
                        "0x1111111111111111111111111111111111111111111111111111111111111111",
                    )
                    .unwrap();
                    let s = B256::from_str(
                        "0x2222222222222222222222222222222222222222222222222222222222222222",
                    )
                    .unwrap();
                    let w = B256::from_str(
                        "0x3333333333333333333333333333333333333333333333333333333333333333",
                    )
                    .unwrap();
                    let c = B256::from_str(
                        "0x4444444444444444444444444444444444444444444444444444444444444444",
                    )
                    .unwrap();

                    assert_eq!(h.parent_batch_hash, p);
                    assert_eq!(h.post_state_root, s);
                    assert_eq!(h.withdraw_root, w);
                    assert_eq!(h.commitment, c);

                    // Sanity: computed batch hash equals the provided one (if method available)
                    // assert_eq!(v.header.batch_hash(), expected_batch_hash);
                }
            }
        }
        _ => panic!("expected validium header variant"),
    }
}
