use crate::zk_circuits_handler::universal::UniversalHandler;
use async_trait::async_trait;
use axiom_sdk::{
    AxiomSdk, ProofType as AxiomProofType,
    build::BuildSdk,
    input::Input as AxiomInput,
    prove::{ProveArgs, ProveSdk},
};
use backon::{BlockingRetryable, ExponentialBuilder};
use eyre::Context;
use jiff::Timestamp;
use scroll_proving_sdk::{
    config::Config as SdkConfig,
    prover::{
        ProofType, ProvingService,
        proving_service::{
            GetVkRequest, GetVkResponse, ProveRequest, ProveResponse, QueryTaskRequest,
            QueryTaskResponse, TaskStatus,
        },
    },
};
use scroll_zkvm_types::{
    ProvingTask,
    proof::{OpenVmEvmProof, OpenVmVersionedVmStarkProof, ProofEnum},
};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, fs::File, io::Write, path::Path, time::Duration};
use tempfile::NamedTempFile;
use tracing::Level;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AxiomProverConfig {
    pub axiom: AxiomConfig,
    pub sdk_config: SdkConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AxiomConfig {
    pub api_key: String,
    // vk to program mapping
    pub programs: HashMap<String, AxiomProgram>,
    pub num_gpus: Option<usize>,
    #[serde(default)]
    pub retry: RetryConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AxiomProgram {
    pub program_id: String,
    pub config_id: String,
}

#[derive(Debug)]
pub struct AxiomProver {
    config: AxiomProverConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RetryConfig {
    jitter: bool,
    factor: f32,
    min_delay: Duration,
    max_delay: Option<Duration>,
    max_times: Option<usize>,
    total_delay: Option<Duration>,
}

impl Default for RetryConfig {
    fn default() -> Self {
        Self {
            jitter: false,
            factor: 2.0,
            min_delay: Duration::from_secs(1),
            max_delay: Some(Duration::from_secs(60)),
            max_times: Some(10),
            total_delay: None,
        }
    }
}

impl From<&RetryConfig> for ExponentialBuilder {
    fn from(cfg: &RetryConfig) -> Self {
        let mut builder = ExponentialBuilder::default()
            .with_factor(cfg.factor)
            .with_total_delay(cfg.total_delay);
        if cfg.jitter {
            builder = builder.with_jitter();
        }
        if let Some(max_delay) = cfg.max_delay {
            builder = builder.with_max_delay(max_delay);
        }
        if let Some(max_times) = cfg.max_times {
            builder = builder.with_max_times(max_times);
        }
        builder
    }
}

impl AxiomProverConfig {
    pub fn from_reader<R>(reader: R) -> eyre::Result<Self>
    where
        R: std::io::Read,
    {
        serde_json::from_reader(reader).map_err(|e| eyre::eyre!(e))
    }

    pub fn from_file<P: AsRef<Path>>(file_name: P) -> eyre::Result<Self> {
        let file = File::open(file_name)?;
        Self::from_reader(&file)
    }
}

#[async_trait]
impl ProvingService for AxiomProver {
    fn is_local(&self) -> bool {
        false
    }

    async fn get_vks(&self, _: GetVkRequest) -> GetVkResponse {
        // get vk has been deprecated in new prover with dynamic asset loading scheme
        GetVkResponse {
            vks: vec![],
            error: None,
        }
    }

    #[instrument(skip(self), ret, level = Level::DEBUG)]
    async fn prove(&mut self, req: ProveRequest) -> ProveResponse {
        self.prove_inner(req)
            .await
            .unwrap_or_else(|e| ProveResponse {
                status: TaskStatus::Failed,
                error: Some(format!("failed to submit proof task to axiom: {}", e)),
                ..Default::default()
            })
    }

    #[instrument(skip(self), ret, level = Level::DEBUG)]
    async fn query_task(&mut self, req: QueryTaskRequest) -> QueryTaskResponse {
        let task_id = req.task_id.clone();
        self.query_task_inner(req)
            .await
            .unwrap_or_else(|e| QueryTaskResponse {
                task_id,
                status: TaskStatus::Failed,
                error: Some(format!("failed to query axiom task: {}", e)),
                ..Default::default()
            })
    }
}

impl AxiomProver {
    pub fn new(config: AxiomProverConfig) -> Self {
        Self { config }
    }

    async fn make_axiom_request<R: Send + 'static>(
        &self,
        config_id: Option<String>,
        req: impl Fn(&AxiomSdk) -> eyre::Result<R> + Send + 'static,
    ) -> eyre::Result<R> {
        let api_key = self.config.axiom.api_key.clone();
        let retry_config = ExponentialBuilder::from(&self.config.axiom.retry);
        tokio::task::spawn_blocking(move || {
            let config = axiom_sdk::AxiomConfig {
                api_key: Some(api_key),
                config_id,
                ..Default::default()
            };
            let sdk = AxiomSdk::new(config);
            let req = || req(&sdk);
            req.retry(retry_config)
                .when(|e| e.to_string().contains("502"))
                .notify(|e, duration| {
                    tracing::warn!("request failed: {e}, retrying in {duration:?}")
                })
                .call()
        })
        .await
        .context("failed to join axiom request")
        .flatten()
    }

    #[instrument(skip_all, ret, err, level = Level::DEBUG)]
    fn get_program(&self, vk: &[u8]) -> eyre::Result<AxiomProgram> {
        let vk = hex::encode(vk);
        debug!(vk = %vk);
        self.config
            .axiom
            .programs
            .get(vk.as_str())
            .cloned()
            .ok_or_else(|| eyre::eyre!("no axiom program configured for vk: {vk}"))
    }

    #[instrument(skip_all, err, level = Level::DEBUG)]
    async fn prove_inner(&mut self, req: ProveRequest) -> eyre::Result<ProveResponse> {
        let prover_task = UniversalHandler::get_task_from_input(&req.input)?;
        if prover_task.use_openvm_13 {
            eyre::bail!("axiom prover does not support openvm v1.3 tasks");
        }

        let prover_task: ProvingTask = prover_task.into();

        let program = self.get_program(&prover_task.vk)?;
        let num_gpus = self.config.axiom.num_gpus;

        let mut input_file = NamedTempFile::new()?;
        let input = prover_task.build_openvm_input();
        serde_json::to_writer(&mut input_file, &input)?;
        input_file.flush()?;

        let proof_type = if req.proof_type == ProofType::Bundle {
            AxiomProofType::Evm
        } else {
            AxiomProofType::Stark
        };

        let mut response = ProveResponse {
            proof_type: req.proof_type,
            created_at: Timestamp::now().as_duration().as_secs_f64(),
            status: TaskStatus::Proving,
            ..Default::default()
        };

        response.task_id = self
            .make_axiom_request(Some(program.config_id), move |sdk| {
                sdk.generate_new_proof(ProveArgs {
                    program_id: Some(program.program_id.clone()),
                    input: Some(AxiomInput::FilePath(input_file.path().to_path_buf())),
                    proof_type: Some(proof_type),
                    num_gpus,
                    priority: None,
                })
            })
            .await?;
        info!(
            proof_type = ?req.proof_type,
            identifier = %prover_task.identifier,
            task_id = %response.task_id,
            "submitted axiom proving task"
        );

        Ok(response)
    }

    #[instrument(skip_all, err, level = Level::DEBUG)]
    async fn query_task_inner(&mut self, req: QueryTaskRequest) -> eyre::Result<QueryTaskResponse> {
        let mut response = QueryTaskResponse {
            task_id: req.task_id.clone(),
            ..Default::default()
        };

        let task_id = req.task_id.clone();

        let (status, proof_type, proof) = self
            .make_axiom_request(None, move |sdk| {
                let status = sdk.get_proof_status(&task_id)?;
                debug!(status = ?status, "fetched axiom task status");

                let program_status = sdk.get_build_status(&status.program_uuid)?;
                let proof_type = match program_status.name.as_str() {
                    "chunk" => ProofType::Chunk,
                    "batch" => ProofType::Batch,
                    "bundle" => ProofType::Bundle,
                    _ => {
                        return Err(eyre::eyre!("unrecognized program in: {program_status:#?}",));
                    }
                };

                let axiom_proof_type: AxiomProofType = status.proof_type.parse()?;
                let proof = if status.state == "Succeeded" {
                    let file = NamedTempFile::new()?;
                    sdk.get_generated_proof(
                        &status.id,
                        &axiom_proof_type,
                        Some(file.path().to_path_buf()),
                    )?;
                    Some(file)
                } else {
                    None
                };

                Ok((status, proof_type, proof))
            })
            .await?;

        // Queued, Executing, Executed, AppProving, AppProvingDone, PostProcessing, Failed,
        // Succeeded
        response.status = match status.state.as_str() {
            "Queued" => TaskStatus::Queued,
            "Executing" | "Executed" | "AppProving" | "AppProvingDone" | "PostProcessing" => {
                TaskStatus::Proving
            }
            "Succeeded" => TaskStatus::Success,
            "Failed" => TaskStatus::Failed,
            other => {
                return Err(eyre::eyre!("unrecognized axiom task status: {other}"));
            }
        };
        debug!(status = ?response.status, "mapped axiom task status");

        if response.status == TaskStatus::Failed {
            response.error = Some(
                status
                    .error_message
                    .unwrap_or_else(|| "unknown error".to_string()),
            );
        }

        response.proof_type = proof_type;

        let created_at: Timestamp = status.created_at.parse()?;
        response.created_at = created_at.as_duration().as_secs_f64();
        if let Some(launched_at) = status.launched_at
            && !launched_at.is_empty()
        {
            let started_at: Timestamp = launched_at.parse()?;
            let started_at = started_at.as_duration();
            response.started_at = Some(started_at.as_secs_f64());

            if let Some(terminated_at) = status.terminated_at
                && !terminated_at.is_empty()
            {
                let finished_at: Timestamp = terminated_at.parse()?;
                let finished_at = finished_at.as_duration();
                response.finished_at = Some(finished_at.as_secs_f64());

                let duration = finished_at.checked_sub(started_at).ok_or_else(|| {
                    eyre::eyre!(
                        "invalid timestamps: started_at={:?}, finished_at={:?}",
                        started_at,
                        finished_at
                    )
                })?;
                response.compute_time_sec = Some(duration.as_secs_f64());
                info!(
                    task_id = %req.task_id,
                    launched_at = %format_args!("{launched_at:#}"),
                    terminated_at = %format_args!("{terminated_at:#}"),
                    duration = %format_args!("{duration:#}"),
                    priority = %status.priority,
                    "completed"
                );
                info!(
                    task_id = %req.task_id,
                    cells_used = %status.cells_used,
                    num_gpus = %status.num_gpus,
                    "resource usage"
                );
                if let Some(num_instructions) = status.num_instructions {
                    let mhz = num_instructions as f64 / (duration.as_secs_f64() * 1_000_000.0);
                    info!(
                        task_id = %req.task_id,
                        cycles = %num_instructions,
                        MHz = %format_args!("{mhz:.2}"),
                        "performance"
                    );
                }
            }
        }

        if let Some(proof_file) = proof {
            let proof = match proof_type {
                ProofType::Bundle => {
                    let proof: OpenVmEvmProof = serde_json::from_reader(proof_file)?;
                    ProofEnum::Evm(proof.into())
                }
                _ => {
                    let proof: OpenVmVersionedVmStarkProof = serde_json::from_reader(proof_file)?;
                    ProofEnum::Stark(proof.try_into()?)
                }
            };

            response.proof = Some(serde_json::to_string(&proof)?);
        }

        Ok(response)
    }
}
