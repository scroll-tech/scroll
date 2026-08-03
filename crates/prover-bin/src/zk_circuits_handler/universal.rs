use std::path::Path;

use eyre::Result;
use libzkp::ProvingTaskExt;
use openvm_circuit::arch::deferral::DeferralState;
use openvm_sdk::DeferralInput;
use scroll_zkvm_prover::{task::ProvingTask as ProvingTaskTrait, Prover, ProverConfig};
use scroll_zkvm_types::ProvingTask;
pub struct UniversalHandler {
    prover: Prover,
}

/// Safe for current usage as `CircuitsHandler` trait (protected inside of Mutex and NEVER extract
/// the instance out by `into_inner`)
unsafe impl Send for UniversalHandler {}

impl UniversalHandler {
    pub fn new(workspace_path: impl AsRef<Path>) -> Result<Self> {
        let path_app_exe = workspace_path.as_ref().join("app.vmexe");
        let path_app_config = workspace_path.as_ref().join("openvm.toml");
        let config = ProverConfig {
            path_app_config,
            path_app_exe,
        };

        let prover = Prover::setup(config, None)?;
        Ok(Self { prover })
    }

    /// Enable OpenVM deferral using `child_prover` as the child circuit prover.
    /// Required for batch (child=chunk) and bundle (child=batch) aggregation proofs.
    pub fn enable_deferral(&mut self, child: &UniversalHandler) -> Result<()> {
        self.prover
            .enable_deferral(&child.prover)
            .map_err(|e| eyre::eyre!("failed to enable deferral: {}", e))?;
        Ok(())
    }

    /// Release the lazily-built SDK (GPU proving keys) of this circuit.
    ///
    /// The openvm VPMM pool never returns physical pages to the OS, but freed
    /// allocations become reusable pool regions; dropping a child circuit's SDK
    /// once deferral is configured lowers the live GPU set (and thus the pool
    /// high-water mark) before the parent's STARK/SNARK phase. Mirrors the
    /// zkvm integration tester, which calls `Prover::reset()` on child provers
    /// after `enable_deferral` for the same reason. The SDK is rebuilt lazily
    /// on the next task that needs this circuit.
    pub fn reset(&mut self) {
        self.prover.reset();
    }

    /// Return the child aggregation VK needed to build deferral data.
    /// Reads the pre-built agg_vk.bin asset instead of sdk.agg_vk(): building
    /// the (GPU) aggregation prover just to obtain the VK uploads ~5.7 GiB of
    /// proving keys into the VPMM pool, which is never reclaimed and starves
    /// the halo2-gpu SNARK phase of VRAM (quotient.cu cudaErrorInvalidConfiguration).
    pub fn agg_vk(&self) -> Result<openvm_stark_sdk::openvm_stark_backend::keygen::types::MultiStarkVerifyingKey<openvm_sdk::SC>> {
        self.prover
            .load_agg_vk()
            .map(|mvk| mvk.as_ref().clone())
            .map_err(|e| eyre::eyre!("failed to load agg vk: {e}"))
    }

    /// Return the cached commit of the verify-stark deferral circuit (def_idx 0).
    pub fn deferral_cached_commit(&self) -> Result<openvm_continuations::CommitBytes> {
        let sdk = self.prover.sdk().map_err(|e| eyre::eyre!("failed to get sdk: {e}"))?;
        let mut commits = sdk
            .deferral_circuit_cached_commits(0)
            .map_err(|e| eyre::eyre!("failed to get deferral cached commits: {e}"))?;
        eyre::ensure!(
            commits.len() == 1,
            "expected one deferral circuit, got {}",
            commits.len()
        );
        Ok(commits.pop().unwrap())
    }

    pub fn get_task_from_input(input: &str) -> Result<ProvingTaskExt> {
        Ok(serde_json::from_str(input)?)
    }

    /// Generate a proof for `u_task`.
    pub fn get_proof_data(&mut self, u_task: &ProvingTask, need_snark: bool) -> Result<String> {
        let proof = self.prover.gen_proof_universal(u_task, need_snark)?;
        Ok(serde_json::to_string(&proof)?)
    }

    /// Generate a proof for `u_task` using deferred STARK verification data.
    ///
    /// For batch/bundle tasks this writes `input_commits` into stdin, attaches the deferral
    /// states and passes the deferral inputs to the SDK prover.
    pub fn get_proof_data_with_deferral(
        &mut self,
        u_task: &ProvingTask,
        need_snark: bool,
        def_inputs: &[DeferralInput],
        def_states: &[DeferralState],
    ) -> Result<String> {
        let mut stdin = u_task.build_guest_input();
        stdin.deferrals = def_states.to_vec();

        let proof = if need_snark {
            scroll_zkvm_types::proof::ProofEnum::from(scroll_zkvm_types::proof::EvmProof::from(
                self.prover.gen_proof_snark(stdin, def_inputs)?,
            ))
        } else {
            scroll_zkvm_types::proof::ProofEnum::from(self.prover.gen_proof_stark(
                stdin, def_inputs,
            )?)
        };

        Ok(serde_json::to_string(&proof)?)
    }
}
