use crate::zk_circuits_handler::{CircuitsHandler, CircuitsHandlerProvider};
use anyhow::{anyhow, Result};
use async_trait::async_trait;
use scroll_proving_sdk::{
    config::Config as SdkConfig,
    prover::{
        proving_service::{
            GetVkRequest, GetVkResponse, ProveRequest, ProveResponse, QueryTaskRequest,
            QueryTaskResponse, TaskStatus,
        },
        ProvingService,
    },
};
use serde::{Deserialize, Serialize};
use std::{
    fs::File,
    sync::Arc,
    time::{SystemTime, UNIX_EPOCH},
};
use tokio::{runtime::Handle, task::JoinHandle};

#[derive(Clone, Serialize, Deserialize)]
pub struct LocalProverConfig {
    pub sdk_config: SdkConfig,
    pub high_version_circuit: CircuitConfig,
    pub low_version_circuit: CircuitConfig,
}

impl LocalProverConfig {
    pub fn from_reader<R>(reader: R) -> Result<Self>
    where
        R: std::io::Read,
    {
        serde_json::from_reader(reader).map_err(|e| anyhow!(e))
    }

    pub fn from_file(file_name: String) -> Result<Self> {
        let file = File::open(file_name)?;
        Self::from_reader(&file)
    }
}

#[derive(Clone, Serialize, Deserialize)]
pub struct CircuitConfig {
    pub hard_fork_name: String,
    pub params_path: String,
    pub assets_path: String,
}

pub struct LocalProver {
    config: LocalProverConfig,
    circuits_handler_provider: CircuitsHandlerProvider,
    next_task_id: u64,
    current_task: Option<JoinHandle<Result<String>>>,
}

#[async_trait]
impl ProvingService for LocalProver {
    fn is_local(&self) -> bool {
        true
    }
    async fn get_vks(&self, req: GetVkRequest) -> GetVkResponse {
        let vks = self
            .circuits_handler_provider
            .init_vks(&self.config, req.proof_types)
            .await;
        GetVkResponse { vks, error: None }
    }
    async fn prove(&mut self, req: ProveRequest) -> ProveResponse {
        let handler = self
            .circuits_handler_provider
            .get_circuits_handler(&req.hard_fork_name)
            .expect("failed to get circuit handler");

        match self.do_prove(req, handler).await {
            Ok(resp) => resp,
            Err(e) => ProveResponse {
                status: TaskStatus::Failed,
                error: Some(format!("failed to request proof: {}", e)),
                ..Default::default()
            },
        }
    }

    async fn query_task(&mut self, req: QueryTaskRequest) -> QueryTaskResponse {
        if let Some(handle) = &mut self.current_task {
            if handle.is_finished() {
                return match handle.await {
                    Ok(Ok(proof)) => QueryTaskResponse {
                        task_id: req.task_id,
                        status: TaskStatus::Success,
                        proof: Some(proof),
                        ..Default::default()
                    },
                    Ok(Err(e)) => QueryTaskResponse {
                        task_id: req.task_id,
                        status: TaskStatus::Failed,
                        error: Some(format!("proving task failed: {}", e)),
                        ..Default::default()
                    },
                    Err(e) => QueryTaskResponse {
                        task_id: req.task_id,
                        status: TaskStatus::Failed,
                        error: Some(format!("proving task panicked: {}", e)),
                        ..Default::default()
                    },
                };
            } else {
                return QueryTaskResponse {
                    task_id: req.task_id,
                    status: TaskStatus::Proving,
                    ..Default::default()
                };
            }
        }
        // If no handle is found
        QueryTaskResponse {
            task_id: req.task_id,
            status: TaskStatus::Failed,
            error: Some("no proving task is running".to_string()),
            ..Default::default()
        }
    }
}

impl LocalProver {
    pub fn new(config: LocalProverConfig) -> Self {
        let circuits_handler_provider = CircuitsHandlerProvider::new(config.clone())
            .expect("failed to create circuits handler provider");

        Self {
            config,
            circuits_handler_provider,
            next_task_id: 0,
            current_task: None,
        }
    }

    async fn do_prove(
        &mut self,
        req: ProveRequest,
        handler: Arc<Box<dyn CircuitsHandler>>,
    ) -> Result<ProveResponse> {
        self.next_task_id += 1;
        let duration = SystemTime::now().duration_since(UNIX_EPOCH).unwrap();
        let created_at = duration.as_secs() as f64 + duration.subsec_nanos() as f64 * 1e-9;

        let req_clone = req.clone();
        let handle = Handle::current();
        let task_handle =
            tokio::task::spawn_blocking(move || handle.block_on(handler.get_proof_data(req_clone)));

        self.current_task = Some(task_handle);

        Ok(ProveResponse {
            task_id: self.next_task_id.to_string(),
            proof_type: req.proof_type,
            circuit_version: req.circuit_version,
            hard_fork_name: req.hard_fork_name,
            status: TaskStatus::Proving,
            created_at,
            input: Some(req.input),
            ..Default::default()
        })
    }
}
