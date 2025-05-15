use std::{
    path::{Path, PathBuf},
    process,
};

use once_cell::sync::OnceCell;
use openvm_sdk::{
    F, Sdk,
    config::{AppConfig, SdkVmConfig},
};
use scroll_zkvm_prover::{
    ProverType, WrappedProof,
    setup::{read_app_config, read_app_exe},
    task::ProvingTask,
};
use tracing::instrument;
use tracing_subscriber::{fmt::format::FmtSpan, layer::SubscriberExt, util::SubscriberInitExt};

pub mod testers;

pub mod utils;

/// Path to store release assets, root directory of zkvm-prover repository.
const DIR_OUTPUT: &str = "./../../.output";

/// Directory to store proofs on disc.
const DIR_PROOFS: &str = "proofs";

/// File descriptor for app openvm config.
const FD_APP_CONFIG: &str = "openvm.toml";

/// File descriptor for app exe.
const FD_APP_EXE: &str = "app.vmexe";

/// Environment variable used to set the test-run's output directory for assets.
const ENV_OUTPUT_DIR: &str = "OUTPUT_DIR";

/// Every test run will write assets to a new directory.
///
/// Possibly one of the following:
/// - <DIR_OUTPUT>/chunk-tests-{timestamp}
/// - <DIR_OUTPUT>/batch-tests-{timestamp}
/// - <DIR_OUTPUT>/bundle-tests-{timestamp}
static DIR_TESTRUN: OnceCell<PathBuf> = OnceCell::new();

/// Circuit that implements functionality required to run e2e tests.
pub trait ProverTester {
    /// Prover type that is being tested.
    type Prover: ProverType;

    /// Path to the corresponding circuit's project directory.
    const PATH_PROJECT_ROOT: &str;

    /// Prefix to use while naming app-specific data like app exe, app pk, etc.
    const DIR_ASSETS: &str;

    /// Setup directory structure for the test suite.
    fn setup() -> eyre::Result<()> {
        // Setup tracing subscriber.
        setup_logger()?;

        // If user has set an output directory, use it.
        let dir_testrun = if let Ok(env_dir) = std::env::var(ENV_OUTPUT_DIR) {
            let dir = Path::new(&env_dir);
            if std::fs::exists(dir).is_err() {
                tracing::error!("OUTPUT_DIR={dir:?} not found");
                process::exit(1);
            }
            dir.into()
        } else {
            // Create the <OUTPUT>/{DIR_ASSETS}-test-{timestamp} for test-run.
            let testrun = format!(
                "{}-tests-{}",
                Self::DIR_ASSETS,
                chrono::Utc::now().format("%Y%m%d_%H%M%S"),
            );
            Path::new(DIR_OUTPUT).join(testrun)
        };

        // Set the path for the current test-run.
        DIR_TESTRUN
            .set(dir_testrun)
            .map_err(|dir| eyre::eyre!("could not set test-run dir: {dir:?}"))?;

        Ok(())
    }

    /// Load the app config.
    fn load_with_exe_fd(
        app_exe_fd: &str,
    ) -> eyre::Result<(PathBuf, AppConfig<SdkVmConfig>, PathBuf)> {
        let path_app_config = Path::new(Self::PATH_PROJECT_ROOT).join(FD_APP_CONFIG);
        let app_config = read_app_config(&path_app_config)?;
        let path_assets = Path::new(Self::PATH_PROJECT_ROOT).join("openvm");
        let path_app_exe = path_assets.join(app_exe_fd);
        Ok((path_app_config, app_config, path_app_exe))
    }

    /// Load the app config.
    fn load() -> eyre::Result<(PathBuf, AppConfig<SdkVmConfig>, PathBuf)> {
        Self::load_with_exe_fd(&Self::fd_app_exe())
    }

    /// Get the path to the app exe.
    fn fd_app_exe() -> String {
        FD_APP_EXE.to_string()
    }

    /// Generate proving task for test purposes.
    fn gen_proving_task() -> eyre::Result<<Self::Prover as ProverType>::ProvingTask>;

    /// Generate multiple proving tasks for test purposes.
    fn gen_multi_proving_tasks() -> eyre::Result<Vec<<Self::Prover as ProverType>::ProvingTask>> {
        unimplemented!("must be implemented by MultiTester");
    }

    /// Light weight testing to simply execute the vm program for test
    #[instrument("ProverTester::execute", skip_all, fields(task_id))]
    fn execute(
        app_config: AppConfig<SdkVmConfig>,
        task: &<Self::Prover as ProverType>::ProvingTask,
        exe_path: impl AsRef<Path>,
    ) -> eyre::Result<Vec<F>> {
        let stdin = task.build_guest_input()?;

        Ok(Sdk::new().execute(read_app_exe(exe_path)?, app_config.app_vm_config, stdin)?)
    }

    fn execute_with_proving_task(
        app_config: AppConfig<SdkVmConfig>,
        exe_path: impl AsRef<Path>,
    ) -> eyre::Result<Vec<F>> {
        Self::execute(app_config, &Self::gen_proving_task()?, exe_path)
    }
}

/// The outcome of a successful prove-verify run.
pub struct ProveVerifyOutcome<T, P> {
    /// Single or multiple proving tasks.
    pub tasks: Vec<T>,
    /// Verified proofs for the proving tasks.
    pub proofs: Vec<P>,
}

impl<T: Clone, P: Clone> ProveVerifyOutcome<T, P> {
    pub fn single(task: T, proof: P) -> Self {
        Self {
            tasks: vec![task],
            proofs: vec![proof],
        }
    }

    pub fn multi(tasks: &[T], proofs: &[P]) -> Self {
        Self {
            tasks: tasks.to_vec(),
            proofs: proofs.to_vec(),
        }
    }
}

/// Setup test environment
fn setup_logger() -> eyre::Result<()> {
    let fmt_layer = tracing_subscriber::fmt::layer()
        .pretty()
        .with_span_events(FmtSpan::CLOSE);

    #[cfg(feature = "limit-logs")]
    {
        let filters = tracing_subscriber::filter::Targets::new()
            .with_target("scroll_zkvm_prover", tracing::Level::INFO)
            .with_target("scroll_zkvm_integration", tracing::Level::DEBUG);

        tracing_subscriber::registry()
            .with(fmt_layer)
            .with(filters)
            .try_init()?;
    }

    #[cfg(not(feature = "limit-logs"))]
    {
        tracing_subscriber::registry()
            .with(tracing_subscriber::EnvFilter::from_default_env())
            .with(fmt_layer)
            .with(metrics_tracing_context::MetricsLayer::new())
            .try_init()?;
    }

    Ok(())
}

/// Alias for convenience.
type ProveVerifyRes<T> = eyre::Result<
    ProveVerifyOutcome<
        <<T as ProverTester>::Prover as ProverType>::ProvingTask,
        WrappedProof<<<T as ProverTester>::Prover as ProverType>::ProofMetadata>,
    >,
>;

/// Alias for convenience.
type ProveVerifyEvmRes<T> = eyre::Result<(
    ProveVerifyOutcome<
        <<T as ProverTester>::Prover as ProverType>::ProvingTask,
        WrappedProof<<<T as ProverTester>::Prover as ProverType>::ProofMetadata>,
    >,
    scroll_zkvm_verifier::verifier::Verifier<scroll_zkvm_verifier::verifier::AnyVerifier>,
    PathBuf,
)>;

/// End-to-end test for a single proving task.
#[instrument(name = "prove_verify_single", skip_all)]
pub fn prove_verify_single<T>(
    task: Option<<T::Prover as ProverType>::ProvingTask>,
) -> ProveVerifyRes<T>
where
    T: ProverTester,
    <T::Prover as ProverType>::ProvingTask: Clone,
    <T::Prover as ProverType>::ProofMetadata: Clone,
    <T::Prover as ProverType>::ProofType: Clone,
{
    let (path_app_config, _, path_app_exe) = T::load()?;

    let cache_dir = DIR_TESTRUN
        .get()
        .ok_or(eyre::eyre!("missing assets dir"))?
        .join(T::DIR_ASSETS)
        .join(DIR_PROOFS);
    std::fs::create_dir_all(&cache_dir)?;

    // Generate proving task for the circuit.
    let task = if let Some(t) = task {
        t
    } else {
        T::gen_proving_task()?
    };

    // Setup prover.
    let config = scroll_zkvm_prover::ProverConfig {
        path_app_exe,
        path_app_config,
        dir_cache: Some(cache_dir),
        ..Default::default()
    };
    let prover = scroll_zkvm_prover::Prover::<T::Prover>::setup(config)?;

    // Construct root proof for the circuit.
    let proof = prover.gen_proof(&task)?;

    // Verify proof.
    prover.verify_proof(&proof)?;

    Ok(ProveVerifyOutcome::single(task, proof))
}

/// End-to-end test for multiple proving tasks of the same prover.
#[instrument(name = "prove_verify_multi", skip_all)]
pub fn prove_verify_multi<T>(
    tasks: Option<&[<T::Prover as ProverType>::ProvingTask]>,
) -> ProveVerifyRes<T>
where
    T: ProverTester,
    <T::Prover as ProverType>::ProvingTask: Clone,
    <T::Prover as ProverType>::ProofMetadata: Clone,
    <T::Prover as ProverType>::ProofType: Clone,
{
    let (path_app_config, _, path_app_exe) = T::load()?;

    // Setup prover.
    let cache_dir = DIR_TESTRUN
        .get()
        .ok_or(eyre::eyre!("missing assets dir"))?
        .join(T::DIR_ASSETS)
        .join(DIR_PROOFS);
    std::fs::create_dir_all(&cache_dir)?;
    let config = scroll_zkvm_prover::ProverConfig {
        path_app_exe,
        path_app_config,
        dir_cache: Some(cache_dir),
        ..Default::default()
    };
    let prover = scroll_zkvm_prover::Prover::<T::Prover>::setup(config)?;

    // Generate proving task for the circuit.
    let tasks = tasks.map_or_else(|| T::gen_multi_proving_tasks(), |tasks| Ok(tasks.to_vec()))?;

    // For each of the tasks, generate and verify proof.
    let proofs = tasks
        .iter()
        .map(|task| {
            let proof = prover.gen_proof(task)?;
            prover.verify_proof(&proof)?;
            Ok(proof)
        })
        .collect::<eyre::Result<Vec<WrappedProof<<T::Prover as ProverType>::ProofMetadata>>>>()?;

    Ok(ProveVerifyOutcome::multi(&tasks, &proofs))
}

/// End-to-end test for a single proving task to generate an EVM-verifiable SNARK proof.
#[instrument(name = "prove_verify_single_evm", skip_all)]
pub fn prove_verify_single_evm<T>(
    task: Option<<T::Prover as ProverType>::ProvingTask>,
) -> ProveVerifyEvmRes<T>
where
    T: ProverTester,
    <T::Prover as ProverType>::ProvingTask: Clone,
    <T::Prover as ProverType>::ProofMetadata: Clone,
    <T::Prover as ProverType>::ProofType: Clone,
{
    let (path_app_config, _, path_app_exe) = T::load()?;

    // Setup prover.
    let path_assets = DIR_TESTRUN
        .get()
        .ok_or(eyre::eyre!("missing testrun dir"))?
        .join(T::DIR_ASSETS);
    let cache_dir = path_assets.join(DIR_PROOFS);
    std::fs::create_dir_all(&cache_dir)?;
    let config = scroll_zkvm_prover::ProverConfig {
        path_app_exe,
        path_app_config,
        dir_cache: Some(cache_dir),
        ..Default::default()
    };
    let prover = scroll_zkvm_prover::Prover::<T::Prover>::setup(config)?;

    // Dump verifier-only assets to disk.
    let (path_vm_config, path_root_committed_exe) = prover.dump_verifier(&path_assets)?;
    let path_verifier_code = Path::new(T::PATH_PROJECT_ROOT)
        .join("openvm")
        .join("verifier.bin");
    let verifier = scroll_zkvm_verifier::verifier::Verifier::setup(
        &path_vm_config,
        &path_root_committed_exe,
        &path_verifier_code,
    )?;

    // Generate proving task for the circuit.
    let task = task.map_or_else(|| T::gen_proving_task(), Ok)?;

    // Construct root proof for the circuit.
    let proof = prover.gen_proof_evm(&task)?;

    // Verify proof.
    prover.verify_proof_evm(&proof)?;

    Ok((
        ProveVerifyOutcome::single(task, proof),
        verifier,
        path_assets,
    ))
}
