use std::path::Path;

use super::CircuitsHandler;
use async_trait::async_trait;
use base64::{prelude::BASE64_STANDARD, Engine};
use eyre::Result;
use scroll_proving_sdk::prover::ProofType;
use scroll_zkvm_prover::{Prover, ProverConfig};
use scroll_zkvm_types::ProvingTask;
use tokio::sync::Mutex;
pub struct UniversalHandler {
    prover: Prover,
}

unsafe impl Send for UniversalHandler {}

impl UniversalHandler {
    pub fn new(workspace_path: impl AsRef<Path>, proof_type: ProofType) -> Result<Self> {
        let path_app_exe = workspace_path.as_ref().join("app.vmexe");
        let path_app_config = workspace_path.as_ref().join("openvm.toml");
        let segment_len = Some((1 << 22) - 100);
        let config = ProverConfig {
            path_app_config,
            path_app_exe,
            segment_len,
        };

        let use_evm = proof_type == ProofType::Bundle;

        let prover = Prover::setup(config, use_evm, None)?;
        Ok(Self { prover })
    }

    /// get_prover get the inner prover, later we would replace chunk/batch/bundle_prover with
    /// universal prover, before that, use bundle_prover as the represent one
    pub fn get_prover(&self) -> &Prover {
        &self.prover
    }

    pub fn get_task_from_input(input: &str) -> Result<ProvingTask> {
        Ok(serde_json::from_str(input)?)
    }
}

#[async_trait]
impl CircuitsHandler for Mutex<UniversalHandler> {

    async fn get_proof_data(&self, u_task: &ProvingTask, need_snark: bool) -> Result<String> {
        let handler_self = self.lock().await;
        // let u_task: ProvingTask = serde_json::from_str(&prove_request.input)?;
        // let expected_vk = handler_self.get_prover().get_app_vk();
        // if u_task.vk != expected_vk {
        //     eyre::bail!(
        //         "vk is not match!, prove type {:?}, expected {}, get {}",
        //         prove_request.proof_type,
        //         BASE64_STANDARD.encode(expected_vk),
        //         BASE64_STANDARD.encode(u_task.vk),
        //     );
        // }
        if need_snark && handler_self.prover.evm_prover.is_none() {
            eyre::bail!(
                "do not init prover for evm (vk: {})",
                BASE64_STANDARD.encode(handler_self.get_prover().get_app_vk())
            )
        }

        let proof = handler_self
            .get_prover()
            .gen_proof_universal(u_task, need_snark)?;

        //TODO: check expected PI
        Ok(serde_json::to_string(&proof)?)
    }
}
