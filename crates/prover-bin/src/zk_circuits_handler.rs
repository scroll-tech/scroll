//pub mod euclid;

#[allow(non_snake_case)]
pub mod euclidV2;

use async_trait::async_trait;
use eyre::Result;
use scroll_proving_sdk::prover::{proving_service::ProveRequest, ProofType};
use scroll_zkvm_prover_euclid::ProverConfig;
use std::path::Path;

#[async_trait]
pub trait CircuitsHandler: Sync + Send {
    fn get_vk(&self, task_type: ProofType) -> String;

    async fn get_proof_data(&self, prove_request: ProveRequest) -> Result<String>;
}

#[derive(Clone, Copy)]
pub(crate) enum Phase {
    EuclidV2,
}

impl Phase {
    pub fn phase_spec_chunk(&self, workspace_path: &Path) -> ProverConfig {
        let dir_cache = Some(workspace_path.join("cache"));
        let path_app_exe = workspace_path.join("chunk/app.vmexe");
        let path_app_config = workspace_path.join("chunk/openvm.toml");
        let segment_len = Some((1 << 22) - 100);
        ProverConfig {
            dir_cache,
            path_app_config,
            path_app_exe,
            segment_len,
            ..Default::default()
        }
    }

    pub fn phase_spec_batch(&self, workspace_path: &Path) -> ProverConfig {
        let dir_cache = Some(workspace_path.join("cache"));
        let path_app_exe = workspace_path.join("batch/app.vmexe");
        let path_app_config = workspace_path.join("batch/openvm.toml");
        let segment_len = Some((1 << 22) - 100);
        ProverConfig {
            dir_cache,
            path_app_config,
            path_app_exe,
            segment_len,
            ..Default::default()
        }
    }

    pub fn phase_spec_bundle(&self, workspace_path: &Path) -> ProverConfig {
        let dir_cache = Some(workspace_path.join("cache"));
        let path_app_config = workspace_path.join("bundle/openvm.toml");
        let segment_len = Some((1 << 22) - 100);
        ProverConfig {
            dir_cache,
            path_app_config,
            segment_len,
            path_app_exe: workspace_path.join("bundle/app.vmexe"),
            ..Default::default()
        }
    }
}
