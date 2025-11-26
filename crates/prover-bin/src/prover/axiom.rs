use crate::zk_circuits_handler::universal::UniversalHandler;
use async_trait::async_trait;
use axiom_sdk::{
    build::BuildSdk,
    input::Input as AxiomInput,
    prove::{ProveArgs, ProveSdk},
    AxiomConfig, AxiomSdk, ProofType as AxiomProofType, SaveOption,
};
use eyre::Context;
use jiff::Timestamp;
use scroll_proving_sdk::{
    config::Config as SdkConfig,
    prover::{
        proving_service::{
            GetVkRequest, GetVkResponse, ProveRequest, ProveResponse, QueryTaskRequest,
            QueryTaskResponse, TaskStatus,
        },
        ProofType, ProvingService,
    },
};
use scroll_zkvm_types::{
    proof::{OpenVmEvmProof, OpenVmVersionedVmStarkProof, ProofEnum},
    ProvingTask,
};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, fs::File, path::Path};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AxiomProverConfig {
    #[serde(rename = "axiom_api_key")]
    pub api_key: String,
    pub sdk_config: SdkConfig,
    // vk to program mapping
    #[serde(rename = "axiom_programs")]
    pub programs: HashMap<String, AxiomProgram>,
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
    #[instrument(skip(self), ret)]
    fn is_local(&self) -> bool {
        false
    }

    #[instrument(skip(self), ret)]
    async fn get_vks(&self, _: GetVkRequest) -> GetVkResponse {
        // get vk has been deprecated in new prover with dynamic asset loading scheme
        GetVkResponse {
            vks: vec![],
            error: None,
        }
    }

    #[instrument(skip(self), ret)]
    async fn prove(&mut self, req: ProveRequest) -> ProveResponse {
        self.prove_inner(req)
            .await
            .unwrap_or_else(|e| ProveResponse {
                status: TaskStatus::Failed,
                error: Some(format!("failed to submit proof task to axiom: {}", e)),
                ..Default::default()
            })
    }

    #[instrument(skip(self), ret)]
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
        req: impl FnOnce(AxiomSdk) -> eyre::Result<R> + Send + 'static,
    ) -> eyre::Result<R> {
        let api_key = self.config.api_key.clone();
        tokio::task::spawn_blocking(move || {
            let config = AxiomConfig {
                api_key: Some(api_key),
                config_id,
                ..Default::default()
            };
            let sdk = AxiomSdk::new(config);
            req(sdk)
        })
        .await
        .context("failed to join axiom request")
        .flatten()
    }

    #[instrument(skip_all, ret, err)]
    fn get_program(&self, vk: &[u8]) -> eyre::Result<AxiomProgram> {
        let vk = hex::encode(vk);
        info!(vk = %vk, "looking up axiom program for vk");
        self.config
            .programs
            .get(vk.as_str())
            .cloned()
            .ok_or_else(|| eyre::eyre!("no axiom program configured for vk: {vk}"))
    }

    #[instrument(skip_all, fields(proof_type = ?req.proof_type), err)]
    async fn prove_inner(&mut self, req: ProveRequest) -> eyre::Result<ProveResponse> {
        let prover_task = UniversalHandler::get_task_from_input(&req.input)?;
        if prover_task.use_openvm_13 {
            eyre::bail!("axiom prover does not support openvm v1.3 tasks");
        }

        let prover_task: ProvingTask = prover_task.into();

        let program = self.get_program(&prover_task.vk)?;

        let input = serde_json::to_value(prover_task.build_openvm_input())?;
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
                    input: Some(AxiomInput::Value(input)),
                    proof_type: Some(proof_type),
                    num_gpus: None,
                    priority: None,
                })
            })
            .await?;
        info!(task_id = %response.task_id, "submitted axiom proving task");

        Ok(response)
    }

    #[instrument(skip_all, fields(task_id = %req.task_id), err)]
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
                    Some(sdk.get_generated_proof(
                        &status.id,
                        &axiom_proof_type,
                        SaveOption::DoNotSave,
                    )?)
                } else {
                    None
                };

                Ok((status, proof_type, proof))
            })
            .await?;

        // Queued, Executing, Executed, AppProving, AppProvingDone, PostProcessing, Failed,
        // Succeeded
        response.status = match status.state.as_str() {
            "Queued" => TaskStatus::Proving,
            "Executing" | "Executed" | "AppProving" | "AppProvingDone" | "PostProcessing" => {
                TaskStatus::Proving
            }
            "Succeeded" => TaskStatus::Success,
            "Failed" => TaskStatus::Failed,
            other => {
                return Err(eyre::eyre!("unrecognized axiom task status: {other}"));
            }
        };
        info!(status = ?response.status, "mapped axiom task status");

        response.proof_type = proof_type;

        let created_at: Timestamp = status.created_at.parse()?;
        response.created_at = created_at.as_duration().as_secs_f64();
        if let Some(launched_at) = status.launched_at {
            let started_at: Timestamp = launched_at.parse()?;
            let started_at = started_at.as_duration();
            response.started_at = Some(started_at.as_secs_f64());

            if let Some(terminated_at) = status.terminated_at {
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
            }
        }

        if let Some(proof_bytes) = proof {
            let proof = match proof_type {
                ProofType::Bundle => {
                    let proof: OpenVmEvmProof = serde_json::from_slice(&proof_bytes)?;
                    ProofEnum::Evm(proof.into())
                }
                _ => {
                    let proof: OpenVmVersionedVmStarkProof = serde_json::from_slice(&proof_bytes)?;
                    ProofEnum::Stark(proof.try_into()?)
                }
            };

            response.proof = Some(serde_json::to_string(&proof)?);
        }

        Ok(response)
    }
}
