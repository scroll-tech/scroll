use std::path::Path;

use super::CircuitsHandler;
use async_trait::async_trait;
use eyre::Result;
use libzkp::ProvintTaskExt;
use scroll_zkvm_prover::{Prover, ProverConfig};
use scroll_zkvm_types::ProvingTask;
use tokio::sync::Mutex;
pub struct UniversalHandler {
    prover: Prover,
}

// additional config dispatched with proving task
#[derive(Debug, Default)]
pub(crate) struct TaskConfig {
    pub is_openvm_v13: bool,
}

/// Safe for current usage as `CircuitsHandler` trait (protected inside of Mutex and NEVER extract
/// the instance out by `into_inner`)
unsafe impl Send for UniversalHandler {}

impl UniversalHandler {
    pub fn new(workspace_path: impl AsRef<Path>, cfg: &TaskConfig) -> Result<Self> {
        let path_app_exe = workspace_path.as_ref().join("app.vmexe");
        let path_app_config = workspace_path.as_ref().join("openvm.toml");
        let segment_len = Some((1 << 22) - 100);
        let config = ProverConfig {
            path_app_config,
            path_app_exe,
            segment_len,
            is_openvm_v13: cfg.is_openvm_v13,
        };

        let prover = Prover::setup(config, None)?;
        Ok(Self { prover })
    }

    /// get_prover get the inner prover, later we would replace chunk/batch/bundle_prover with
    /// universal prover, before that, use bundle_prover as the represent one
    pub fn get_prover(&mut self) -> &mut Prover {
        &mut self.prover
    }

    pub fn get_task_from_input(input: &str) -> Result<(ProvingTask, TaskConfig)> {
        let task_ext: ProvintTaskExt = serde_json::from_str(input)?;
        let cfg = TaskConfig {
            is_openvm_v13: task_ext.use_openvm_13,
        };

        Ok((task_ext.into(), cfg))
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
