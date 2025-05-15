use std::path::Path;

use openvm_continuations::verifier::root::types::RootVmVerifierInput;
use openvm_native_recursion::halo2::RawEvmProof as OpenVmEvmProof;
use openvm_stark_sdk::{
    openvm_stark_backend::{p3_field::PrimeField32, proof::Proof},
    p3_baby_bear::BabyBear,
};
use sbv_primitives::B256;
use scroll_zkvm_circuit_input_types::{
    batch::BatchInfo,
    bundle::BundleInfo,
    chunk::ChunkInfo,
    public_inputs::{ForkName, MultiVersionPublicInputs},
};
use serde::{Deserialize, Serialize, de::DeserializeOwned};
use snark_verifier_sdk::snark_verifier::{
    halo2_base::halo2_proofs::halo2curves::bn256::Fr, util::arithmetic::PrimeField,
};

use crate::{
    Error, SC,
    utils::{as_base64, base64 as vec_as_base64, short_git_version},
};

/// Helper type for convenience that implements [`From`] and [`Into`] traits between
/// [`OpenVmEvmProof`]. The difference is that the instances in [`EvmProof`] are the byte-encoding
/// of the flattened [`Fr`] elements.
#[derive(Clone, Serialize, Deserialize)]
pub struct EvmProof {
    /// The proof bytes.
    #[serde(with = "vec_as_base64")]
    pub proof: Vec<u8>,
    /// Byte-encoding of the flattened scalar fields representing the public inputs of the SNARK
    /// proof.
    #[serde(with = "vec_as_base64")]
    pub instances: Vec<u8>,
}

impl From<&OpenVmEvmProof> for EvmProof {
    fn from(value: &OpenVmEvmProof) -> Self {
        let instances = value
            .instances
            .iter()
            .flat_map(|fr| {
                let mut be_bytes = fr.to_bytes();
                be_bytes.reverse();
                be_bytes
            })
            .collect::<Vec<u8>>();

        Self {
            proof: value.proof.to_vec(),
            instances,
        }
    }
}

impl From<&EvmProof> for OpenVmEvmProof {
    fn from(value: &EvmProof) -> Self {
        assert_eq!(
            value.instances.len() % 32,
            0,
            "expect len(instances) % 32 == 0"
        );

        let instances = value
            .instances
            .chunks_exact(32)
            .map(|be_bytes| {
                Fr::from_repr({
                    let mut le_bytes: [u8; 32] = be_bytes
                        .try_into()
                        .expect("instances.len() % 32 == 0 has already been asserted");
                    le_bytes.reverse();
                    le_bytes
                })
                .expect("Fr::from_repr failed")
            })
            .collect::<Vec<Fr>>();

        Self {
            proof: value.proof.to_vec(),
            instances,
        }
    }
}

/// Helper to modify serde implementations on the remote [`RootProof`] type.
#[derive(Serialize, Deserialize)]
#[serde(remote = "RootProof")]
struct RootProofDef {
    /// The proofs.
    #[serde(with = "as_base64")]
    proofs: Vec<Proof<SC>>,
    /// The public values for the proof.
    #[serde(with = "as_base64")]
    public_values: Vec<BabyBear>,
}

/// Lists the proof variants possible in Scroll's proving architecture.
#[derive(Clone, Serialize, Deserialize)]
#[serde(untagged)]
pub enum ProofEnum {
    /// Represents a STARK proof used for intermediary layers, i.e. chunk and batch.
    #[serde(with = "RootProofDef")]
    Root(RootProof),
    /// Represents a SNARK proof used for the final layer to be verified on-chain, i.e. bundle.
    Evm(EvmProof),
}

impl From<RootProof> for ProofEnum {
    fn from(value: RootProof) -> Self {
        Self::Root(value)
    }
}

impl From<EvmProof> for ProofEnum {
    fn from(value: EvmProof) -> Self {
        Self::Evm(value)
    }
}

impl From<OpenVmEvmProof> for ProofEnum {
    fn from(value: OpenVmEvmProof) -> Self {
        Self::Evm(EvmProof::from(&value))
    }
}

impl ProofEnum {
    /// Get the root proof as reference.
    pub fn as_root_proof(&self) -> Option<&RootProof> {
        match self {
            Self::Root(proof) => Some(proof),
            _ => None,
        }
    }

    /// Get the EVM proof as defined in [`openvm_native_recursion`].
    ///
    /// Essentially construct a [`OpenVmEvmProof`] from the inner contained [`EvmProof`].
    pub fn as_evm_proof(&self) -> Option<OpenVmEvmProof> {
        match self {
            Self::Evm(proof) => Some(OpenVmEvmProof::from(proof)),
            _ => None,
        }
    }

    /// Consumes the proof enum and returns the contained root proof.
    pub fn into_root_proof(self) -> Option<RootProof> {
        match self {
            Self::Root(proof) => Some(proof),
            _ => None,
        }
    }

    /// Consumes the proof enum and returns the [`OpenVmEvmProof`].
    pub fn into_evm_proof(self) -> Option<OpenVmEvmProof> {
        match self {
            Self::Evm(ref proof) => Some(OpenVmEvmProof::from(proof)),
            _ => None,
        }
    }

    /// Derive public inputs from the proof.
    pub fn public_values(&self) -> Vec<u32> {
        match self {
            Self::Root(root_proof) => root_proof
                .public_values
                .iter()
                .map(|x| x.as_canonical_u32())
                .collect::<Vec<u32>>(),
            Self::Evm(evm_proof) => {
                // The first 12 scalars are accumulators.
                // The next 2 scalars are digests.
                // The next 32 scalars are the public input hash.
                let pi_hash_bytes = evm_proof
                    .instances
                    .iter()
                    .skip(14 * 32)
                    .take(32 * 32)
                    .cloned()
                    .collect::<Vec<u8>>();

                // The 32 scalars of public input hash actually only have the LSB that is the
                // meaningful byte.
                pi_hash_bytes
                    .chunks_exact(32)
                    .map(|bytes32_chunk| bytes32_chunk[31] as u32)
                    .collect::<Vec<u32>>()
            }
        }
    }
}

/// A wrapper around the actual inner proof.
#[derive(Clone, Serialize, Deserialize)]
pub struct WrappedProof<Metadata> {
    /// Generic metadata carried by a proof.
    pub metadata: Metadata,
    /// The inner proof, either a [`RootProof`] or [`EvmProof`] depending on the [`crate::ProverType`].
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

/// Alias for convenience.
pub type RootProof = RootVmVerifierInput<SC>;

/// Alias for convenience.
pub type ChunkProof = WrappedProof<ChunkProofMetadata>;

/// Alias for convenience.
pub type BatchProof = WrappedProof<BatchProofMetadata>;

/// Alias for convenience.
pub type BundleProof = WrappedProof<BundleProofMetadata>;

/// Trait to enable operations in metadata
pub trait ProofMetadata: Serialize + DeserializeOwned + std::fmt::Debug {
    type PublicInputs: MultiVersionPublicInputs;

    fn pi_hash_info(&self) -> &Self::PublicInputs;
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
    /// The [`scroll_zkvm_circuit_input_types::batch::BatchHeader`]'s digest.
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

impl<Metadata: ProofMetadata> WrappedProof<Metadata>
where
    Metadata: DeserializeOwned + Serialize,
{
    /// Wrap a proof with some metadata.
    pub fn new<P: Into<ProofEnum>>(metadata: Metadata, proof: P, vk: Option<&[u8]>) -> Self {
        Self {
            metadata,
            proof: proof.into(),
            vk: vk.map(Vec::from).unwrap_or_default(),
            git_version: short_git_version(),
        }
    }

    /// Sanity checks on the wrapped proof:
    ///
    /// - pi_hash computed in host does in fact match pi_hash computed in guest
    pub fn sanity_check(&self, fork_name: ForkName) {
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

        assert_eq!(
            expected_pi, proof_pi,
            "pi mismatch: expected={expected_pi:?}, found={proof_pi:?}"
        );
    }

    /// Read and deserialize the proof.
    pub fn from_json<P: AsRef<Path>>(path_proof: P) -> Result<Self, Error> {
        crate::utils::read_json_deep(path_proof)
    }

    /// Serialize the proof and dumping at the given path.
    pub fn dump<P: AsRef<Path>>(&self, path_proof: P) -> Result<(), Error> {
        crate::utils::write_json(path_proof, &self)
    }
}

impl ChunkProof {
    /// Get the contained root proof as reference.
    pub fn as_proof(&self) -> &RootProof {
        self.proof
            .as_root_proof()
            .expect("ChunkProof contains RootProof")
    }

    /// Consume self and return the contained root proof.
    pub fn into_proof(self) -> RootProof {
        self.proof
            .into_root_proof()
            .expect("ChunkProof contains RootProof")
    }
}

impl BatchProof {
    /// Get the contained root proof as reference.
    pub fn as_proof(&self) -> &RootProof {
        self.proof
            .as_root_proof()
            .expect("BatchProof contains RootProof")
    }

    /// Consume self and return the contained root proof.
    pub fn into_proof(self) -> RootProof {
        self.proof
            .into_root_proof()
            .expect("BatchProof contains RootProof")
    }
}

impl BundleProof {
    /// Get the contained evm proof (as [`OpenVmEvmProof`]).
    pub fn as_proof(&self) -> OpenVmEvmProof {
        self.proof
            .as_evm_proof()
            .expect("BundleProof contains EvmProof")
    }

    /// Consume self and return the contained evm proof (as [`OpenVmEvmProof`]).
    pub fn into_proof(self) -> OpenVmEvmProof {
        self.proof
            .into_evm_proof()
            .expect("BundleProof contains EvmProof")
    }
}

#[cfg(test)]
mod tests {
    use alloy_primitives::B256;
    use base64::{Engine, prelude::BASE64_STANDARD};
    use openvm_native_recursion::halo2::RawEvmProof;
    use scroll_zkvm_circuit_input_types::{
        bundle::{BundleInfo, BundleInfoV1},
        public_inputs::PublicInputs,
    };
    use snark_verifier_sdk::snark_verifier::halo2_base::halo2_proofs::halo2curves::bn256::Fr;

    use super::{BatchProof, BundleProof, BundleProofMetadata, ChunkProof};

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
            let instances = vec![
                Fr::from(0x123456),   // LE: [0x56, 0x34, 0x12, 0x00, 0x00, ..., 0x00]
                Fr::from(0x98765432), // LE: [0x32, 0x54, 0x76, 0x98, 0x00, ..., 0x00]
            ];
            let instances_flattened = std::iter::empty()
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
            let instances_base64 = BASE64_STANDARD.encode(&instances_flattened);
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

        let evm_proof = RawEvmProof { instances, proof };
        let bundle_proof = BundleProof::new(metadata, evm_proof, Some(vk.as_slice()));
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
            bundle_proof_de.as_proof().proof,
            bundle_proof.as_proof().proof
        );
        assert_eq!(
            bundle_proof_de.as_proof().instances,
            bundle_proof.as_proof().instances,
        );
        assert_eq!(bundle_proof_de.vk, bundle_proof.vk);

        Ok(())
    }
}
