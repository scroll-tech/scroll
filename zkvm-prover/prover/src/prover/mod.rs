use std::{
    marker::PhantomData,
    path::{Path, PathBuf},
    sync::Arc,
};

use once_cell::sync::Lazy;
use openvm_circuit::{arch::SingleSegmentVmExecutor, system::program::trace::VmCommittedExe};
use openvm_native_recursion::{
    halo2::{
        RawEvmProof,
        utils::{CacheHalo2ParamsReader, Halo2ParamsReader},
        wrapper::Halo2WrapperProvingKey,
    },
    hints::Hintable,
};
use openvm_sdk::{
    DefaultStaticVerifierPvHandler, NonRootCommittedExe, Sdk, StdIn,
    commit::AppExecutionCommit,
    config::{AggConfig, AggStarkConfig, SdkVmConfig},
    keygen::{AggStarkProvingKey, AppProvingKey},
    prover::{AggStarkProver, AppProver, EvmHalo2Prover},
};
use openvm_stark_sdk::config::baby_bear_poseidon2::BabyBearPoseidon2Engine;
use serde::{Serialize, de::DeserializeOwned};
use tracing::{debug, instrument};

// Re-export from openvm_sdk.
pub use openvm_sdk::{self, F, SC};

use crate::{
    Error, WrappedProof,
    proof::{ProofMetadata, RootProof},
    setup::{read_app_config, read_app_exe},
    task::{ProvingTask, flatten_wrapped_proof},
};

mod batch;
pub use batch::{BatchProver, BatchProverType};

mod bundle;
pub use bundle::{
    BundleProverEuclidV1, BundleProverEuclidV2, BundleProverTypeEuclidV1, BundleProverTypeEuclidV2,
    GenericBundleProverType,
};

mod chunk;
pub use chunk::{ChunkProver, ChunkProverType, ChunkProverTypeRv32, GenericChunkProverType};

/// Proving key for STARK aggregation. Primarily used to aggregate
/// [continuation proofs][openvm_sdk::prover::vm::ContinuationVmProof].
static AGG_STARK_PROVING_KEY: Lazy<AggStarkProvingKey> =
    Lazy::new(|| AggStarkProvingKey::keygen(AggStarkConfig::default()));

/// The default directory to locate openvm's halo2 SRS parameters.
const DEFAULT_PARAMS_DIR: &str = concat!(env!("HOME"), "/.openvm/params/");

/// The environment variable that needs to be set in order to configure the directory from where
/// Prover can read HALO2 trusted setup parameters.
const ENV_HALO2_PARAMS_DIR: &str = "ENV_HALO2_PARAMS_DIR";

/// File descriptor for the root verifier's VM config.
const FD_ROOT_VERIFIER_VM_CONFIG: &str = "root-verifier-vm-config";

/// File descriptor for the root verifier's committed exe.
const FD_ROOT_VERIFIER_COMMITTED_EXE: &str = "root-verifier-committed-exe";

pub trait Commitments {
    const EXE_COMMIT: [u32; 8];
    const LEAF_COMMIT: [u32; 8];
}

/// Types used in the outermost proof construction and verification, i.e. the EVM-compatible layer.
pub struct EvmProverVerifier {
    /// This is required only for [BundleProver].
    pub halo2_prover: EvmHalo2Prover<SdkVmConfig, BabyBearPoseidon2Engine>,
    /// The halo2 proving key.
    pub halo2_pk: Halo2WrapperProvingKey,
    /// The contract bytecode for the EVM verifier contract.
    pub verifier_contract: Vec<u8>,
}

/// Generic prover.
pub struct Prover<Type> {
    /// Commitment to app exe.
    pub app_committed_exe: Arc<NonRootCommittedExe>,
    /// App specific proving key.
    pub app_pk: Arc<AppProvingKey<SdkVmConfig>>,
    /// Optional data for the outermost layer, i.e. EVM-compatible.
    pub evm_prover: Option<EvmProverVerifier>,
    /// Optional directory to cache generated proofs. If such a cached proof is located, then its
    /// returned instead of re-generating a proof.
    pub cache_dir: Option<PathBuf>,

    _type: PhantomData<Type>,
}

/// Alias for convenience.
type InitRes = (
    Arc<VmCommittedExe<SC>>,
    Arc<AppProvingKey<SdkVmConfig>>,
    AppExecutionCommit<F>,
);

/// Configure the [`Prover`].
#[derive(Debug, Clone, Default)]
pub struct ProverConfig {
    /// Path to find applications's app.vmexe.
    pub path_app_exe: PathBuf,
    /// Path to find application's OpenVM config.
    pub path_app_config: PathBuf,
    /// An optional directory to cache generated proofs.
    ///
    /// If a proof is already available in the cache directory, the proof generation method will
    /// early return with the available proof on disk.
    pub dir_cache: Option<PathBuf>,
    /// An optional directory to locate HALO2 trusted setup parameters.
    pub dir_halo2_params: Option<PathBuf>,
    /// The maximum length for a single OpenVM segment.
    pub segment_len: Option<usize>,
}

impl<Type: ProverType> Prover<Type> {
    /// Setup the [`Prover`] given paths to the application's exe and proving key.
    #[instrument("Prover::setup")]
    pub fn setup(config: ProverConfig) -> Result<Self, Error> {
        let (app_committed_exe, app_pk, _) = Self::init(&config)?;

        let evm_prover = Type::EVM
            .then(|| Self::setup_evm_prover(&config, &app_committed_exe, &app_pk))
            .transpose()?;

        Ok(Self {
            app_committed_exe,
            app_pk,
            evm_prover,
            cache_dir: config.dir_cache,
            _type: PhantomData,
        })
    }

    /// Read app exe, proving key and return committed data.
    #[instrument("Prover::init")]
    pub fn init(config: &ProverConfig) -> Result<InitRes, Error> {
        let app_exe = read_app_exe(&config.path_app_exe)?;
        let mut app_config = read_app_config(&config.path_app_config)?;
        let segment_len = config.segment_len.unwrap_or(Type::SEGMENT_SIZE);
        app_config.app_vm_config.system.config = app_config
            .app_vm_config
            .system
            .config
            .with_max_segment_len(segment_len);

        let sdk = Sdk::new();
        let app_pk = sdk
            .app_keygen(app_config)
            .map_err(|e| Error::Keygen(e.to_string()))?;
        let app_committed_exe = sdk
            .commit_app_exe(app_pk.app_fri_params(), app_exe)
            .map_err(|e| Error::Commit(e.to_string()))?;

        let (commits, _) = Self::get_verify_program_commitment(&app_committed_exe, &app_pk, true);

        Ok((app_committed_exe, Arc::new(app_pk), commits))
    }

    /// Dump assets required to setup verifier-only mode.
    pub fn dump_verifier<P: AsRef<Path>>(&self, dir: P) -> Result<(PathBuf, PathBuf), Error> {
        if !Type::EVM {
            return Err(Error::Custom(
                "dump_verifier only at bundle-prover".to_string(),
            ));
        };
        let root_verifier_pk = &AGG_STARK_PROVING_KEY.root_verifier_pk;
        let vm_config = root_verifier_pk.vm_pk.vm_config.clone();
        let root_committed_exe: &VmCommittedExe<_> = &root_verifier_pk.root_committed_exe;

        let path_vm_config = dir.as_ref().join(FD_ROOT_VERIFIER_VM_CONFIG);
        let path_root_committed_exe = dir.as_ref().join(FD_ROOT_VERIFIER_COMMITTED_EXE);

        crate::utils::write_bin(&path_vm_config, &vm_config)?;
        crate::utils::write_bin(&path_root_committed_exe, &root_committed_exe)?;

        Ok((path_vm_config, path_root_committed_exe))
    }

    /// Pick up app commit as "vk" in proof, to distinguish from which circuit the proof comes
    pub fn get_app_vk(&self) -> Vec<u8> {
        let (_, [exe, leaf]) =
            Self::get_verify_program_commitment(&self.app_committed_exe, &self.app_pk, false);

        scroll_zkvm_circuit_input_types::types_agg::ProgramCommitment { exe, leaf }.serialize()
    }

    /// Pick up the actual vk (serialized) for evm proof, would be empty if prover
    /// do not contain evm prover
    pub fn get_evm_vk(&self) -> Vec<u8> {
        self.evm_prover
            .as_ref()
            .map(|evm_prover| {
                scroll_zkvm_verifier::evm::serialize_vk(evm_prover.halo2_pk.pinning.pk.get_vk())
            })
            .unwrap_or_default()
    }

    /// Early-return if a proof is found in disc, otherwise generate and return the proof after
    /// writing to disc.
    #[instrument("Prover::gen_proof", skip_all, fields(task_id, prover_name = Type::NAME))]
    pub fn gen_proof(
        &self,
        task: &Type::ProvingTask,
    ) -> Result<WrappedProof<Type::ProofMetadata>, Error> {
        let task_id = task.identifier();

        // Try reading proof from cache if available, and early return in that case.
        if let Some(dir) = &self.cache_dir {
            let path_proof = dir.join(Self::fd_proof(task));
            debug!(name: "try_read_proof", ?task_id, ?path_proof);

            if let Ok(proof) = crate::utils::read_json_deep(&path_proof) {
                debug!(name: "early_return_proof", ?task_id);
                return Ok(proof);
            }
        }

        // Generate a new proof.
        assert!(!Type::EVM, "Prover::gen_proof not for EVM-prover");
        let metadata = Self::metadata_with_prechecks(task)?;
        let proof = self.gen_proof_stark(task)?;
        let wrapped_proof = WrappedProof::new(metadata, proof, Some(self.get_app_vk().as_slice()));

        wrapped_proof.sanity_check(task.fork_name());

        // Write proof to disc if caching was enabled.
        if let Some(dir) = &self.cache_dir {
            let path_proof = dir.join(Self::fd_proof(task));
            debug!(name: "try_write_proof", ?task_id, ?path_proof);

            crate::utils::write_json(&path_proof, &wrapped_proof)?;
        }

        Ok(wrapped_proof)
    }

    /// Early-return if a proof is found in disc, otherwise generate and return the proof after
    /// writing to disc.
    #[instrument("Prover::gen_proof_evm", skip_all, fields(task_id))]
    pub fn gen_proof_evm(
        &self,
        task: &Type::ProvingTask,
    ) -> Result<WrappedProof<Type::ProofMetadata>, Error> {
        let task_id = task.identifier();

        // Try reading proof from cache if available, and early return in that case.
        if let Some(dir) = &self.cache_dir {
            let path_proof = dir.join(Self::fd_proof(task));
            debug!(name: "try_read_proof", ?task_id, ?path_proof);

            if let Ok(proof) = crate::utils::read_json_deep(&path_proof) {
                debug!(name: "early_return_proof", ?task_id);
                return Ok(proof);
            }
        }

        // Generate a new proof.
        assert!(Type::EVM, "Prover::gen_proof_evm only for EVM-prover");
        let metadata = Self::metadata_with_prechecks(task)?;
        let proof = self.gen_proof_snark(task)?;
        let wrapped_proof = WrappedProof::new(metadata, proof, Some(self.get_evm_vk().as_slice()));

        wrapped_proof.sanity_check(task.fork_name());

        // Write proof to disc if caching was enabled.
        if let Some(dir) = &self.cache_dir {
            let path_proof = dir.join(Self::fd_proof(task));
            debug!(name: "try_write_proof", ?task_id, ?path_proof);

            crate::utils::write_json(&path_proof, &wrapped_proof)?;
        }

        Ok(wrapped_proof)
    }

    /// Validate some pre-checks on the proving task and construct proof metadata.
    #[instrument("Prover::metadata_with_prechecks", skip_all, fields(?task_id = task.identifier()))]
    pub fn metadata_with_prechecks(task: &Type::ProvingTask) -> Result<Type::ProofMetadata, Error> {
        Type::metadata_with_prechecks(task)
    }

    /// Verify a [root proof][root_proof].
    /// TODO: currently this method is only used in testing. Move it else
    /// [root_proof][RootProof]
    #[instrument("Prover::verify_proof", skip_all, fields(?metadata = proof.metadata))]
    pub fn verify_proof(&self, proof: &WrappedProof<Type::ProofMetadata>) -> Result<(), Error> {
        let agg_stark_pk = &AGG_STARK_PROVING_KEY;

        let root_verifier_pk = &agg_stark_pk.root_verifier_pk;
        let vm_executor = SingleSegmentVmExecutor::new(root_verifier_pk.vm_pk.vm_config.clone());
        let exe: &VmCommittedExe<_> = &root_verifier_pk.root_committed_exe;

        let root_proof = proof.proof.as_root_proof().ok_or(Error::VerifyProof(
            "verify_proof expects RootProof".to_string(),
        ))?;
        vm_executor
            .execute_and_compute_heights(exe.exe.clone(), root_proof.write())
            .map_err(|e| Error::VerifyProof(e.to_string()))?;

        let aggregation_input = flatten_wrapped_proof(proof);
        if aggregation_input.commitment.exe != Type::EXE_COMMIT {
            return Err(Error::VerifyProof(format!(
                "EXE_COMMIT mismatch: expected={:?}, got={:?}",
                Type::EXE_COMMIT,
                aggregation_input.commitment.exe,
            )));
        }
        if aggregation_input.commitment.leaf != Type::LEAF_COMMIT {
            return Err(Error::VerifyProof(format!(
                "LEAF_COMMIT mismatch: expected={:?}, got={:?}",
                Type::LEAF_COMMIT,
                aggregation_input.commitment.leaf,
            )));
        }

        Ok(())
    }

    /// Verify an [evm proof][evm_proof].
    ///
    /// [evm_proof][openvm_native_recursion::halo2::EvmProof]
    #[instrument("Prover::verify_proof_evm", skip_all)]
    pub fn verify_proof_evm(&self, proof: &WrappedProof<Type::ProofMetadata>) -> Result<(), Error> {
        let evm_proof = proof.proof.as_evm_proof().ok_or(Error::VerifyProof(
            "verify_proof_evm expects EvmProof".to_string(),
        ))?;
        let contract = &self
            .evm_prover
            .as_ref()
            .expect("uninited")
            .verifier_contract;
        let gas_cost = scroll_zkvm_verifier::evm::verify_evm_proof(contract, &evm_proof)
            .map_err(|e| Error::VerifyProof(format!("EVM-proof verification failed: {e}")))?;

        tracing::info!(name: "verify_evm_proof", ?gas_cost);

        Ok(())
    }

    /// Execute the guest program to get the cycle count.
    pub fn execute_and_check(&self, stdin: &StdIn, mock_prove: bool) -> Result<u64, Error> {
        let config = self.app_pk.app_vm_pk.vm_config.clone();
        let exe = self.app_committed_exe.exe.clone();
        let debug_input = crate::utils::vm::DebugInput {
            mock_prove,
            commited_exe: mock_prove.then(|| self.app_committed_exe.clone()),
        };
        let exec_result = crate::utils::vm::execute_guest(config, exe, stdin, &debug_input)?;
        Ok(exec_result.total_cycle as u64)
    }

    /// Setup the EVM prover-verifier.
    fn setup_evm_prover(
        config: &ProverConfig,
        app_committed_exe: &Arc<NonRootCommittedExe>,
        app_pk: &Arc<AppProvingKey<SdkVmConfig>>,
    ) -> Result<EvmProverVerifier, Error> {
        // The HALO2 directory is set in the following order:
        // 1. If the optional dir_halo2_params is set: use it.
        // 2. If the optional dir_halo2_params is not set: try to read from env variable.
        // 3. If the env var is not set: use the default directory.
        let dir_halo2_params = config
            .dir_halo2_params
            .clone()
            .ok_or(std::env::var(ENV_HALO2_PARAMS_DIR))
            .unwrap_or(Path::new(DEFAULT_PARAMS_DIR).to_path_buf());

        let halo2_params_reader = CacheHalo2ParamsReader::new(&dir_halo2_params);
        let agg_pk = Sdk::new()
            .agg_keygen(
                AggConfig::default(),
                &halo2_params_reader,
                &DefaultStaticVerifierPvHandler,
            )
            .map_err(|e| Error::Setup {
                path: dir_halo2_params,
                src: e.to_string(),
            })?;

        let halo2_params = halo2_params_reader
            .read_params(agg_pk.halo2_pk.wrapper.pinning.metadata.config_params.k);
        let path_verifier_sol = config
            .path_app_exe
            .parent()
            .map(|dir| dir.join("verifier.sol"));
        let path_verifier_bin = config
            .path_app_exe
            .parent()
            .map(|dir| dir.join("verifier.bin"));
        let verifier_contract = scroll_zkvm_verifier::evm::gen_evm_verifier::<
            scroll_zkvm_verifier::evm::halo2_aggregation::AggregationCircuit,
        >(
            &halo2_params,
            agg_pk.halo2_pk.wrapper.pinning.pk.get_vk(),
            agg_pk.halo2_pk.wrapper.pinning.metadata.num_pvs.clone(),
            path_verifier_sol.as_deref(),
        );
        if let Some(path) = path_verifier_bin {
            crate::utils::write(path, &verifier_contract)?;
        }

        let halo2_pk = agg_pk.halo2_pk.wrapper.clone();
        let halo2_prover = EvmHalo2Prover::new(
            &halo2_params_reader,
            Arc::clone(app_pk),
            Arc::clone(app_committed_exe),
            agg_pk,
            Default::default(),
        );

        Ok(EvmProverVerifier {
            halo2_prover,
            halo2_pk,
            verifier_contract,
        })
    }

    /// File descriptor for the proof saved to disc.
    #[instrument("Prover::fd_proof", skip_all, fields(task_id = task.identifier(), path_proof))]
    fn fd_proof(task: &Type::ProvingTask) -> String {
        let path_proof = format!("{}-{}.json", Type::NAME, task.identifier());
        path_proof
    }

    /// Generate a [root proof][root_proof].
    ///
    /// [root_proof][openvm_sdk::verifier::root::types::RootVmVerifierInput]
    fn gen_proof_stark(&self, task: &Type::ProvingTask) -> Result<RootProof, Error> {
        let stdin = task
            .build_guest_input()
            .map_err(|e| Error::GenProof(e.to_string()))?;

        let mock_prove = std::env::var("MOCK_PROVE").as_deref() == Ok("true");
        // Here we always do an execution of the guest program to get the cycle count.
        // and do precheck before proving like ensure PI != 0
        self.execute_and_check(&stdin, mock_prove)?;

        let task_id = task.identifier();

        // sanity check
        let _ = Self::get_verify_program_commitment(&self.app_committed_exe, &self.app_pk, false);

        tracing::debug!(name: "generate_root_verifier_input", ?task_id);
        let app_prover = AppProver::<_, BabyBearPoseidon2Engine>::new(
            self.app_pk.app_vm_pk.clone(),
            self.app_committed_exe.clone(),
        );
        // TODO: should we cache the app_proof?
        let app_proof = app_prover.generate_app_proof(stdin);
        tracing::info!("app proof generated for {} task {task_id}", Type::NAME);
        let agg_prover = AggStarkProver::<BabyBearPoseidon2Engine>::new(
            AGG_STARK_PROVING_KEY.clone(),
            self.app_pk.leaf_committed_exe.clone(),
            Default::default(),
        );
        let proof = agg_prover.generate_root_verifier_input(app_proof);
        Ok(proof)
    }

    /// Generate an [evm proof][evm_proof].
    ///
    /// [evm_proof][openvm_native_recursion::halo2::EvmProof]
    fn gen_proof_snark(&self, task: &Type::ProvingTask) -> Result<RawEvmProof, Error> {
        let stdin = task
            .build_guest_input()
            .map_err(|e| Error::GenProof(e.to_string()))?;

        let evm_proof: RawEvmProof = self
            .evm_prover
            .as_ref()
            .expect("Prover::gen_proof_snark expects EVM-prover setup")
            .halo2_prover
            .generate_proof_for_evm(stdin)
            .try_into()
            .map_err(|e| Error::GenProof(format!("{}", e)))?;

        // sanity check
        assert_eq!(
            evm_proof.instances[12],
            crate::utils::compress_commitment(&Type::EXE_COMMIT),
            "commitment is not match in generate evm proof",
        );
        assert_eq!(
            evm_proof.instances[13],
            crate::utils::compress_commitment(&Type::LEAF_COMMIT),
            "commitment is not match in generate evm proof",
        );

        Ok(evm_proof)
    }

    fn get_verify_program_commitment(
        app_committed_exe: &NonRootCommittedExe,
        app_pk: &AppProvingKey<SdkVmConfig>,
        debug_out: bool,
    ) -> (AppExecutionCommit<F>, [[u32; 8]; 2]) {
        use openvm_stark_sdk::openvm_stark_backend::p3_field::PrimeField32;
        let commits = AppExecutionCommit::compute(
            &app_pk.app_vm_pk.vm_config,
            app_committed_exe,
            &app_pk.leaf_committed_exe,
        );

        let exe_commit = commits.exe_commit.map(|x| x.as_canonical_u32());
        let leaf_commit = commits
            .leaf_vm_verifier_commit
            .map(|x| x.as_canonical_u32());

        // print the 2 exe commitments
        if debug_out {
            debug!(name: "exe-commitment", prover_name = Type::NAME, raw = ?exe_commit, as_bn254 = ?commits.exe_commit_to_bn254());
            debug!(name: "leaf-commitment", prover_name = Type::NAME, raw = ?leaf_commit, as_bn254 = ?commits.app_config_commit_to_bn254());
        }

        assert_eq!(
            exe_commit,
            Type::EXE_COMMIT,
            "read unmatched exe commitment from app"
        );
        assert_eq!(
            leaf_commit,
            Type::LEAF_COMMIT,
            "read unmatched app commitment from app"
        );
        (commits, [exe_commit, leaf_commit])
    }
}

pub trait ProverType {
    /// The name given to the prover, this is also used as a prefix while storing generated proofs
    /// to disc.
    const NAME: &'static str;

    /// Whether this prover generates SNARKs that are EVM-verifiable. In our context, only the
    /// [`BundleProver`] has the EVM set to `true`.
    const EVM: bool;

    /// The size of a segment, i.e. the max height of its chips.
    const SEGMENT_SIZE: usize;

    /// The app program's exe commitment.
    const EXE_COMMIT: [u32; 8];

    /// The app program's leaf commitment.
    const LEAF_COMMIT: [u32; 8];

    /// The task provided as argument during proof generation process.
    type ProvingTask: ProvingTask;

    /// The proof type, i.e. whether [root proof][root_proof] or [evm proof][evm_proof].
    ///
    /// [root_proof][openvm_sdk::verifier::root::types::RootVmVerifierInput]
    /// [evm_proof][openvm_native_recursion::halo2::EvmProof]
    type ProofType: Serialize + DeserializeOwned;

    /// The metadata accompanying the wrapper proof generated by this prover.
    type ProofMetadata: ProofMetadata;

    /// Provided the proving task, computes the proof metadata.
    fn metadata_with_prechecks(task: &Self::ProvingTask) -> Result<Self::ProofMetadata, Error>;
}
