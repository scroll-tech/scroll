//pub mod euclid;

#[allow(non_snake_case)]
pub mod universal;
// keep an old handler for utilities on assets
pub mod assets;

use async_trait::async_trait;
use eyre::Result;
use scroll_zkvm_prover::ProverConfig;
use scroll_zkvm_types::ProvingTask;
use std::path::Path;

#[async_trait]
pub trait CircuitsHandler: Sync + Send {
    #[allow(dead_code)]
    async fn get_vk(&self) -> String;

    async fn get_proof_data(&self, u_task: &ProvingTask, need_snark: bool) -> Result<String>;
}

#[derive(Clone, Copy)]
pub(crate) enum Phase {
    EuclidV2,
}

impl Phase {
    pub fn phase_spec_chunk(&self, workspace_path: &Path) -> ProverConfig {
        let path_app_exe = workspace_path.join("chunk/app.vmexe");
        let path_app_config = workspace_path.join("chunk/openvm.toml");
        let segment_len = Some((1 << 22) - 100);
        ProverConfig {
            path_app_config,
            path_app_exe,
            segment_len,
            ..Default::default()
        }
    }

    pub fn phase_spec_batch(&self, workspace_path: &Path) -> ProverConfig {
        let path_app_exe = workspace_path.join("batch/app.vmexe");
        let path_app_config = workspace_path.join("batch/openvm.toml");
        let segment_len = Some((1 << 22) - 100);
        ProverConfig {
            path_app_config,
            path_app_exe,
            segment_len,
            ..Default::default()
        }
    }

    pub fn phase_spec_bundle(&self, workspace_path: &Path) -> ProverConfig {
        let path_app_config = workspace_path.join("bundle/openvm.toml");
        let segment_len = Some((1 << 22) - 100);
        ProverConfig {
            path_app_config,
            segment_len,
            path_app_exe: workspace_path.join("bundle/app.vmexe"),
            ..Default::default()
        }
    }
}
