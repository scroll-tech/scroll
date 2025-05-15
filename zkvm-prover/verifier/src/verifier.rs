use std::{marker::PhantomData, path::Path};

use openvm_circuit::{arch::SingleSegmentVmExecutor, system::program::trace::VmCommittedExe};
use openvm_continuations::verifier::root::types::RootVmVerifierInput;
use openvm_native_circuit::NativeConfig;
use openvm_native_recursion::{halo2::RawEvmProof, hints::Hintable};
use openvm_sdk::{F, RootSC, SC};
use scroll_zkvm_circuit_input_types::types_agg::ProgramCommitment;
use snark_verifier_sdk::snark_verifier::halo2_base::halo2_proofs::halo2curves::bn256::Fr;

use crate::commitments::{
    batch::{EXE_COMMIT as BATCH_EXE_COMMIT, LEAF_COMMIT as BATCH_LEAF_COMMIT},
    bundle, bundle_euclidv1,
    chunk::{EXE_COMMIT as CHUNK_EXE_COMMIT, LEAF_COMMIT as CHUNK_LEAF_COMMIT},
    chunk_rv32,
};

fn compress_commitment(commitment: &[u32; 8]) -> Fr {
    use openvm_stark_sdk::{openvm_stark_backend::p3_field::PrimeField32, p3_baby_bear::BabyBear};
    let order = Fr::from(BabyBear::ORDER_U32 as u64);
    let mut base = Fr::one();
    let mut ret = Fr::zero();

    for v in commitment {
        ret += Fr::from(*v as u64) * base;
        base *= order;
    }

    ret
}

pub trait VerifierType {
    const EXE_COMMIT: [u32; 8];
    const LEAF_COMMIT: [u32; 8];
    fn get_app_vk() -> Vec<u8> {
        ProgramCommitment {
            exe: Self::EXE_COMMIT,
            leaf: Self::LEAF_COMMIT,
        }
        .serialize()
    }
    fn match_exe_commitment_with_pi(pi: &[Option<u32>]) -> bool {
        &pi[..8] == Self::EXE_COMMIT.map(Some).as_slice()
    }
    fn match_leaf_commitment_with_pi(pi: &[Option<u32>]) -> bool {
        &pi[8..16] == Self::LEAF_COMMIT.map(Some).as_slice()
    }
    fn match_exe_commitment_with_evm_ins(ins: &[Fr]) -> bool {
        ins[12] == compress_commitment(&Self::EXE_COMMIT)
    }
    fn match_leaf_commitment_with_evm_ins(ins: &[Fr]) -> bool {
        ins[13] == compress_commitment(&Self::LEAF_COMMIT)
    }
}

pub struct Rv32ChunkVerifierType;
pub struct ChunkVerifierType;
pub struct BatchVerifierType;
pub struct BundleVerifierTypeEuclidV1;
pub struct BundleVerifierTypeEuclidV2;

impl VerifierType for ChunkVerifierType {
    const EXE_COMMIT: [u32; 8] = CHUNK_EXE_COMMIT;
    const LEAF_COMMIT: [u32; 8] = CHUNK_LEAF_COMMIT;
    fn match_exe_commitment_with_pi(pi: &[Option<u32>]) -> bool {
        &pi[..8] == Self::EXE_COMMIT.map(Some).as_slice()
            || Rv32ChunkVerifierType::match_exe_commitment_with_pi(pi)
    }
    fn match_leaf_commitment_with_pi(pi: &[Option<u32>]) -> bool {
        &pi[8..16] == Self::LEAF_COMMIT.map(Some).as_slice()
            || Rv32ChunkVerifierType::match_leaf_commitment_with_pi(pi)
    }
}

impl VerifierType for Rv32ChunkVerifierType {
    const EXE_COMMIT: [u32; 8] = chunk_rv32::EXE_COMMIT;
    const LEAF_COMMIT: [u32; 8] = chunk_rv32::LEAF_COMMIT;
}
impl VerifierType for BatchVerifierType {
    const EXE_COMMIT: [u32; 8] = BATCH_EXE_COMMIT;
    const LEAF_COMMIT: [u32; 8] = BATCH_LEAF_COMMIT;
}
impl VerifierType for BundleVerifierTypeEuclidV1 {
    const EXE_COMMIT: [u32; 8] = bundle_euclidv1::EXE_COMMIT;
    const LEAF_COMMIT: [u32; 8] = bundle_euclidv1::LEAF_COMMIT;
}
impl VerifierType for BundleVerifierTypeEuclidV2 {
    const EXE_COMMIT: [u32; 8] = bundle::EXE_COMMIT;
    const LEAF_COMMIT: [u32; 8] = bundle::LEAF_COMMIT;
}
pub struct Verifier<Type> {
    pub vm_executor: SingleSegmentVmExecutor<F, NativeConfig>,
    pub root_committed_exe: VmCommittedExe<RootSC>,
    pub evm_verifier: Vec<u8>,

    _type: PhantomData<Type>,
}

pub type AnyVerifier = Verifier<ChunkVerifierType>;
pub type ChunkVerifier = Verifier<ChunkVerifierType>;
pub type BatchVerifier = Verifier<BatchVerifierType>;
pub type BundleVerifierEuclidV1 = Verifier<BundleVerifierTypeEuclidV1>;
pub type BundleVerifierEuclidV2 = Verifier<BundleVerifierTypeEuclidV2>;

impl<Type> Verifier<Type> {
    pub fn setup<P: AsRef<Path>>(
        path_vm_config: P,
        path_root_committed_exe: P,
        path_verifier_code: P,
    ) -> eyre::Result<Self> {
        let vm_executor = {
            let bytes = std::fs::read(path_vm_config.as_ref())?;
            let vm_config: NativeConfig = bincode_v1::deserialize(&bytes)?;
            SingleSegmentVmExecutor::new(vm_config)
        };

        let root_committed_exe = std::fs::read(path_root_committed_exe.as_ref())
            .map_err(|e| e.into())
            .and_then(|bytes| bincode_v1::deserialize(&bytes))?;

        let evm_verifier = std::fs::read(path_verifier_code.as_ref())?;

        Ok(Self {
            vm_executor,
            root_committed_exe,
            evm_verifier,
            _type: PhantomData,
        })
    }

    pub fn switch_to<AnotherType>(self) -> Verifier<AnotherType> {
        Verifier::<AnotherType> {
            vm_executor: self.vm_executor,
            root_committed_exe: self.root_committed_exe,
            evm_verifier: self.evm_verifier,
            _type: PhantomData,
        }
    }

    pub fn to_chunk_verifier(self) -> ChunkVerifier {
        self.switch_to()
    }
    pub fn to_batch_verifier(self) -> BatchVerifier {
        self.switch_to()
    }
    pub fn to_bundle_verifier_v1(self) -> BundleVerifierEuclidV1 {
        self.switch_to()
    }
    pub fn to_bundle_verifier_v2(self) -> BundleVerifierEuclidV2 {
        self.switch_to()
    }
}

impl<Type: VerifierType> Verifier<Type> {
    pub fn get_app_vk(&self) -> Vec<u8> {
        Type::get_app_vk()
    }

    pub fn verify_proof(&self, root_proof: &RootVmVerifierInput<SC>) -> bool {
        match self.verify_proof_inner(root_proof) {
            Ok(pi) => {
                assert!(pi.len() >= 16, "unexpected len(pi)<16");
                assert!(
                    Type::match_exe_commitment_with_pi(&pi),
                    "mismatch EXE commitment"
                );
                assert!(
                    Type::match_leaf_commitment_with_pi(&pi),
                    "mismatch LEAF commitment"
                );
                true
            }
            Err(_) => false,
        }
    }

    pub fn verify_proof_evm(&self, evm_proof: &RawEvmProof) -> bool {
        assert!(
            Type::match_exe_commitment_with_evm_ins(&evm_proof.instances),
            "mismatch EXE commitment"
        );
        assert!(
            Type::match_leaf_commitment_with_evm_ins(&evm_proof.instances),
            "mismatch LEAF commitment"
        );
        crate::evm::verify_evm_proof(&self.evm_verifier, evm_proof).is_ok()
    }

    pub(crate) fn verify_proof_inner(
        &self,
        root_proof: &RootVmVerifierInput<SC>,
    ) -> eyre::Result<Vec<Option<u32>>> {
        use openvm_stark_sdk::openvm_stark_backend::p3_field::PrimeField32;
        Ok(self
            .vm_executor
            .execute_and_compute_heights(self.root_committed_exe.exe.clone(), root_proof.write())
            .map(|exec_res| {
                exec_res
                    .public_values
                    .iter()
                    .map(|op_f| op_f.map(|f| f.as_canonical_u32()))
                    .collect()
            })?)
    }
}

#[cfg(test)]
mod tests {
    use std::path::Path;

    use scroll_zkvm_circuit_input_types::types_agg::ProgramCommitment;
    use scroll_zkvm_prover::{BatchProof, BundleProof, ChunkProof, utils::read_json_deep};

    use super::{BatchVerifier, BundleVerifierEuclidV2, ChunkVerifier};

    const PATH_TESTDATA: &str = "./testdata";

    #[ignore = "need release assets"]
    #[test]
    fn verify_chunk_proof() -> eyre::Result<()> {
        let chunk_proof = read_json_deep::<_, ChunkProof>(
            Path::new(PATH_TESTDATA)
                .join("proofs")
                .join("chunk-proof-phase2.json"),
        )?;

        // Note: the committed exe has to match the version of openvm
        // which is used to generate the proof
        let verifier = ChunkVerifier::setup(
            Path::new(PATH_TESTDATA).join("root-verifier-vm-config"),
            Path::new(PATH_TESTDATA).join("root-verifier-committed-exe"),
            Path::new(PATH_TESTDATA).join("verifier.bin"),
        )?;

        let commitment = ProgramCommitment::deserialize(&chunk_proof.vk);
        let root_proof = chunk_proof.as_proof();
        let pi = verifier.verify_proof_inner(root_proof).unwrap();
        assert_eq!(
            &pi[..8],
            commitment.exe.map(Some).as_slice(),
            "the output is not match with exe commitment in root proof!",
        );
        assert_eq!(
            &pi[8..16],
            commitment.leaf.map(Some).as_slice(),
            "the output is not match with leaf commitment in root proof!",
        );
        assert!(
            verifier.verify_proof(root_proof),
            "proof verification failed",
        );

        Ok(())
    }

    #[ignore = "need release assets"]
    #[test]
    fn verify_batch_proof() -> eyre::Result<()> {
        let batch_proof = read_json_deep::<_, BatchProof>(
            Path::new(PATH_TESTDATA)
                .join("proofs")
                .join("batch-proof-phase2.json"),
        )?;

        let verifier = BatchVerifier::setup(
            Path::new(PATH_TESTDATA).join("root-verifier-vm-config"),
            Path::new(PATH_TESTDATA).join("root-verifier-committed-exe"),
            Path::new(PATH_TESTDATA).join("verifier.bin"),
        )?;

        let commitment = ProgramCommitment::deserialize(&batch_proof.vk);
        let root_proof = batch_proof.as_proof();
        let pi = verifier.verify_proof_inner(root_proof).unwrap();
        assert_eq!(
            &pi[..8],
            commitment.exe.map(Some).as_slice(),
            "the output is not match with exe commitment in root proof!",
        );
        assert_eq!(
            &pi[8..16],
            commitment.leaf.map(Some).as_slice(),
            "the output is not match with leaf commitment in root proof!",
        );
        assert!(
            verifier.verify_proof(root_proof),
            "proof verification failed",
        );

        Ok(())
    }

    #[ignore = "need released assets"]
    #[test]
    fn verify_bundle_proof() -> eyre::Result<()> {
        let evm_proof = read_json_deep::<_, BundleProof>(
            Path::new(PATH_TESTDATA)
                .join("proofs")
                .join("bundle-proof-phase2.json"),
        )?;

        let verifier = BundleVerifierEuclidV2::setup(
            Path::new(PATH_TESTDATA).join("root-verifier-vm-config"),
            Path::new(PATH_TESTDATA).join("root-verifier-committed-exe"),
            Path::new(PATH_TESTDATA).join("verifier.bin"),
        )?;

        assert!(verifier.verify_proof_evm(&evm_proof.as_proof()));

        Ok(())
    }
}
