use std::path::Path;

use super::CircuitsHandler;
use async_trait::async_trait;
use eyre::Result;
use libzkp::ProvingTaskExt;
use scroll_zkvm_prover::{Prover, ProverConfig};
use scroll_zkvm_types::ProvingTask;
use tokio::sync::Mutex;
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

    /// get_prover get the inner prover, later we would replace chunk/batch/bundle_prover with
    /// universal prover, before that, use bundle_prover as the represent one
    pub fn get_prover(&mut self) -> &mut Prover {
        &mut self.prover
    }

    pub fn get_task_from_input(input: &str) -> Result<ProvingTaskExt> {
        Ok(serde_json::from_str(input)?)
    }
}

#[async_trait]
impl CircuitsHandler for Mutex<UniversalHandler> {
    async fn get_proof_data(&self, u_task: &ProvingTask, need_snark: bool) -> Result<String> {
        let mut handler_self = self.lock().await;

        let proof = handler_self
            .get_prover()
            .gen_proof_universal(u_task, need_snark)?;

        Ok(serde_json::to_string(&proof)?)
    }
}
