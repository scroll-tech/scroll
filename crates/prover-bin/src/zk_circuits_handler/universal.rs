use std::{
    path::Path,
    sync::Arc,
};

use super::CircuitsHandler;
use crate::prover::CircuitConfig;
use async_trait::async_trait;
use base64::{prelude::BASE64_STANDARD, Engine};
use eyre::Result;
use scroll_proving_sdk::prover::{proving_service::ProveRequest, ProofType};
use scroll_zkvm_prover::{Prover, ProverConfig};
use scroll_zkvm_types::ProvingTask;
use tokio::sync::Mutex;
pub struct UniversalHandler {
    prover: Prover,
}

unsafe impl Send for UniversalHandler {}

impl UniversalHandler {
    pub fn new(cfg: &CircuitConfig, proof_type: ProofType) -> Result<Self> {
        let workspace_path = Path::new(&cfg.workspace_path);
        let dir_cache = Some(workspace_path.join("cache"));
        let path_app_exe = workspace_path.join("app.vmexe");
        let path_app_config = workspace_path.join("openvm.toml");
        let segment_len = Some((1 << 22) - 100);
        let config = ProverConfig {
            dir_cache,
            path_app_config,
            path_app_exe,
            segment_len,
            ..Default::default()
        };

        let use_evm = proof_type == ProofType::Bundle;

        let prover = Prover::setup(config, use_evm, None)?;
        Ok(Self {
            prover,
        })
    }

    /// get_prover get the inner prover, later we would replace chunk/batch/bundle_prover with
    /// universal prover, before that, use bundle_prover as the represent one
    pub fn get_prover(&self) -> &Prover {
        &self.prover
    }

}

#[async_trait]
impl CircuitsHandler for Mutex<UniversalHandler> {
    async fn get_vk(&self) -> String {
        BASE64_STANDARD.encode(self.lock().await.get_prover().get_app_vk())
    }

    async fn get_proof_data(&self, prove_request: ProveRequest) -> Result<String> {
        let handler_self = self.lock().await;
        let u_task: ProvingTask = serde_json::from_str(&prove_request.input)?;
        let expected_vk = handler_self.get_prover().get_app_vk();
        if u_task.vk != expected_vk {
            eyre::bail!(
                "vk is not match!, prove type {:?}, expected {}, get {}",
                prove_request.proof_type,
                BASE64_STANDARD.encode(expected_vk),
                BASE64_STANDARD.encode(u_task.vk),
            );
        }

        let use_evm = prove_request.proof_type == ProofType::Bundle;
        let proof = handler_self.get_prover().gen_proof_universal(&u_task, use_evm)?;

        //TODO: check expected PI
        Ok(serde_json::to_string(&proof)?)
    }
}
