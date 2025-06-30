use std::path::Path;

use crate::utils::short_git_version;
use eyre::Result;
use sbv_primitives::B256;
use scroll_zkvm_types::{
    batch::BatchInfo,
    bundle::BundleInfo,
    chunk::ChunkInfo,
    proof::{EvmProof, OpenVmEvmProof, ProofEnum, RootProof},
    public_inputs::{ForkName, MultiVersionPublicInputs},
    types_agg::{AggregationInput, ProgramCommitment},
    util::vec_as_base64,
};
use serde::{de::DeserializeOwned, Deserialize, Serialize};

/// A wrapper around the actual inner proof.
#[derive(Clone, Serialize, Deserialize)]
pub struct WrappedProof<Metadata> {
    /// Generic metadata carried by a proof.
    pub metadata: Metadata,
    /// The inner proof, either a [`RootProof`] or [`EvmProof`] depending on the
    /// [`crate::ProverType`].
    pub proof: ProofEnum,
    /// Represents the verifying key in serialized form. The purpose of including the verifying key
    /// along with the proof is to allow a verifier-only mode to identify the source of proof
    /// generation.
    ///
    /// For [`RootProof`] the verifying key is denoted by the digest of the VM's program.
    ///
    /// For [`EvmProof`] its the raw bytes of the halo2 circuit's `VerifyingKey`.
    ///
    /// We encode the vk in base64 format during JSON serialization.
    #[serde(with = "vec_as_base64", default)]
    pub vk: Vec<u8>,
    /// Represents the git ref for `zkvm-prover` that was used to construct the proof.
    ///
    /// This is useful for debugging.
    pub git_version: String,
}

pub trait AsRootProof {
    fn as_root_proof(&self) -> &RootProof;
}

pub trait AsEvmProof {
    fn as_evm_proof(&self) -> &EvmProof;
}

pub trait IntoEvmProof {
    fn into_evm_proof(self) -> OpenVmEvmProof;
}

/// Alias for convenience.
pub type ChunkProof = WrappedProof<ChunkProofMetadata>;

/// Alias for convenience.
pub type BatchProof = WrappedProof<BatchProofMetadata>;

/// Alias for convenience.
pub type BundleProof = WrappedProof<BundleProofMetadata>;

impl AsRootProof for ChunkProof {
    fn as_root_proof(&self) -> &RootProof {
        self.proof
            .as_root_proof()
            .expect("batch proof use root proof")
    }
}

impl AsRootProof for BatchProof {
    fn as_root_proof(&self) -> &RootProof {
        self.proof
            .as_root_proof()
            .expect("batch proof use root proof")
    }
}

impl AsEvmProof for BundleProof {
    fn as_evm_proof(&self) -> &EvmProof {
        self.proof
            .as_evm_proof()
            .expect("bundle proof use evm proof")
    }
}

impl IntoEvmProof for BundleProof {
    fn into_evm_proof(self) -> OpenVmEvmProof {
        self.proof
            .as_evm_proof()
            .expect("bundle proof use evm proof")
            .clone()
            .into()
    }
}

/// Trait to enable operations in metadata
pub trait ProofMetadata: Serialize + DeserializeOwned + std::fmt::Debug {
    type PublicInputs: MultiVersionPublicInputs;

    fn pi_hash_info(&self) -> &Self::PublicInputs;

    fn new_proof<P: Into<ProofEnum>>(self, proof: P, vk: Option<&[u8]>) -> WrappedProof<Self> {
        WrappedProof {
            metadata: self,
            proof: proof.into(),
            vk: vk.map(Vec::from).unwrap_or_default(),
            git_version: short_git_version(),
        }
    }
}

pub trait PersistableProof: Sized {
    /// Read and deserialize the proof.
    fn from_json<P: AsRef<Path>>(path_proof: P) -> Result<Self>;
    /// Serialize the proof and dumping at the given path.
    fn dump<P: AsRef<Path>>(&self, path_proof: P) -> Result<()>;
}

/// Metadata attached to [`ChunkProof`].
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ChunkProofMetadata {
    /// The chunk information describing the list of blocks contained within the chunk.
    pub chunk_info: ChunkInfo,
}

impl ProofMetadata for ChunkProofMetadata {
    type PublicInputs = ChunkInfo;

    fn pi_hash_info(&self) -> &Self::PublicInputs {
        &self.chunk_info
    }
}

/// Metadata attached to [`BatchProof`].
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct BatchProofMetadata {
    /// The batch information describing the list of chunks.
    pub batch_info: BatchInfo,
    /// The [`scroll_zkvm_types::batch::BatchHeader`]'s digest.
    pub batch_hash: B256,
}

impl ProofMetadata for BatchProofMetadata {
    type PublicInputs = BatchInfo;

    fn pi_hash_info(&self) -> &Self::PublicInputs {
        &self.batch_info
    }
}

/// Metadata attached to [`BundleProof`].
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct BundleProofMetadata {
    /// The bundle information describing the list of batches to be finalised on-chain.
    pub bundle_info: BundleInfo,
    /// The public-input digest for the bundle.
    pub bundle_pi_hash: B256,
}

impl ProofMetadata for BundleProofMetadata {
    type PublicInputs = BundleInfo;

    fn pi_hash_info(&self) -> &Self::PublicInputs {
        &self.bundle_info
    }
}

impl<Metadata> From<&WrappedProof<Metadata>> for AggregationInput {
    fn from(value: &WrappedProof<Metadata>) -> Self {
        Self {
            public_values: value.proof.public_values(),
            commitment: ProgramCommitment::deserialize(&value.vk),
        }
    }
}

impl<Metadata: ProofMetadata> WrappedProof<Metadata> {
    /// Sanity checks on the wrapped proof:
    ///
    /// - pi_hash computed in host does in fact match pi_hash computed in guest
    pub fn pi_hash_check(&self, fork_name: ForkName) -> bool {
        let proof_pi = self.proof.public_values();

        let expected_pi = self
            .metadata
            .pi_hash_info()
            .pi_hash_by_fork(fork_name)
            .0
            .as_ref()
            .iter()
            .map(|&v| v as u32)
            .collect::<Vec<_>>();

        let ret = expected_pi == proof_pi;
        if !ret {
            tracing::warn!("pi mismatch: expected={expected_pi:?}, found={proof_pi:?}");
        }
        ret
    }
}

impl<Metadata: ProofMetadata> PersistableProof for WrappedProof<Metadata> {
    fn from_json<P: AsRef<Path>>(path_proof: P) -> Result<Self> {
        crate::utils::read_json_deep(path_proof)
    }

    fn dump<P: AsRef<Path>>(&self, path_proof: P) -> Result<()> {
        crate::utils::write_json(path_proof, &self)
    }
}

#[cfg(test)]
mod tests {
    use base64::{prelude::BASE64_STANDARD, Engine};
    use sbv_primitives::B256;
    use scroll_zkvm_types::{
        bundle::{BundleInfo, BundleInfoV1},
        proof::EvmProof,
        public_inputs::PublicInputs,
    };

    use super::*;

    #[test]
    fn test_roundtrip() -> eyre::Result<()> {
        macro_rules! assert_roundtrip {
            ($fd:expr, $proof:ident) => {
                let proof_str_expected =
                    std::fs::read_to_string(std::path::Path::new("./testdata").join($fd))?;
                let proof = serde_json::from_str::<$proof>(&proof_str_expected)?;
                let proof_str_got = serde_json::to_string(&proof)?;
                assert_eq!(proof_str_got, proof_str_expected);
            };
        }

        assert_roundtrip!("chunk-proof.json", ChunkProof);
        assert_roundtrip!("batch-proof.json", BatchProof);
        assert_roundtrip!("bundle-proof.json", BundleProof);

        Ok(())
    }

    #[test]
    fn test_dummy_proof() -> eyre::Result<()> {
        // 1. Metadata
        let metadata = {
            let bundle_info: BundleInfoV1 = BundleInfo {
                chain_id: 12345,
                num_batches: 12,
                prev_state_root: B256::repeat_byte(1),
                prev_batch_hash: B256::repeat_byte(2),
                post_state_root: B256::repeat_byte(3),
                batch_hash: B256::repeat_byte(4),
                withdraw_root: B256::repeat_byte(5),
                msg_queue_hash: B256::repeat_byte(6),
            }
            .into();
            let bundle_pi_hash = bundle_info.pi_hash();
            BundleProofMetadata {
                bundle_info: bundle_info.0,
                bundle_pi_hash,
            }
        };

        // 2. Proof
        let (proof, proof_base64) = {
            let proof = std::iter::empty()
                .chain(std::iter::repeat_n(1, 1))
                .chain(std::iter::repeat_n(2, 2))
                .chain(std::iter::repeat_n(3, 3))
                .chain(std::iter::repeat_n(4, 4))
                .chain(std::iter::repeat_n(5, 5))
                .chain(std::iter::repeat_n(6, 6))
                .chain(std::iter::repeat_n(7, 7))
                .chain(std::iter::repeat_n(8, 8))
                .chain(std::iter::repeat_n(9, 9))
                .collect::<Vec<u8>>();
            let proof_base64 = BASE64_STANDARD.encode(&proof);
            (proof, proof_base64)
        };

        // 3. Instances
        let (instances, instances_base64) = {
            // LE: [0x56, 0x34, 0x12, 0x00, 0x00, ..., 0x00]
            // LE: [0x32, 0x54, 0x76, 0x98, 0x00, ..., 0x00]
            let instances = std::iter::empty()
                .chain(std::iter::repeat_n(0x00, 29))
                .chain(std::iter::once(0x12))
                .chain(std::iter::once(0x34))
                .chain(std::iter::once(0x56))
                .chain(std::iter::repeat_n(0x00, 28))
                .chain(std::iter::once(0x98))
                .chain(std::iter::once(0x76))
                .chain(std::iter::once(0x54))
                .chain(std::iter::once(0x32))
                .collect::<Vec<u8>>();
            let instances_base64 = BASE64_STANDARD.encode(&instances);
            (instances, instances_base64)
        };

        // 4. VK
        let (vk, vk_base64) = {
            let vk = std::iter::empty()
                .chain(std::iter::repeat_n(1, 9))
                .chain(std::iter::repeat_n(2, 8))
                .chain(std::iter::repeat_n(3, 7))
                .chain(std::iter::repeat_n(4, 6))
                .chain(std::iter::repeat_n(5, 5))
                .chain(std::iter::repeat_n(6, 4))
                .chain(std::iter::repeat_n(7, 3))
                .chain(std::iter::repeat_n(8, 2))
                .chain(std::iter::repeat_n(9, 1))
                .collect::<Vec<u8>>();
            let vk_base64 = BASE64_STANDARD.encode(&vk);
            (vk, vk_base64)
        };

        let evm_proof = EvmProof { instances, proof };
        let bundle_proof = metadata.new_proof(evm_proof, Some(vk.as_slice()));
        let bundle_proof_json = serde_json::to_value(&bundle_proof)?;

        assert_eq!(
            bundle_proof_json.get("proof").unwrap(),
            &serde_json::json!({
                "proof": proof_base64,
                "instances": instances_base64,
            }),
        );
        assert_eq!(
            bundle_proof_json.get("vk").unwrap(),
            &serde_json::Value::String(vk_base64),
        );

        let bundle_proof_de = serde_json::from_value::<BundleProof>(bundle_proof_json)?;

        assert_eq!(
            bundle_proof_de.proof.as_evm_proof(),
            bundle_proof.proof.as_evm_proof()
        );
        assert_eq!(bundle_proof_de.vk, bundle_proof.vk);

        Ok(())
    }
}
