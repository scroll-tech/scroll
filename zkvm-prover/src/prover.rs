use crate::zk_circuits_handler::{
    euclid::EuclidHandler, euclidV2::EuclidV2Handler, CircuitsHandler,
};
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
    collections::HashMap,
    fs::File,
    sync::Arc,
    time::{SystemTime, UNIX_EPOCH},
};
use tokio::{runtime::Handle, sync::Mutex, task::JoinHandle};

#[derive(Clone, Serialize, Deserialize)]
pub struct LocalProverConfig {
    pub sdk_config: SdkConfig,
    pub circuits: HashMap<String, CircuitConfig>,
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
    pub workspace_path: String,
}

pub struct LocalProver {
    config: LocalProverConfig,
    next_task_id: u64,
    current_task: Option<JoinHandle<Result<String>>>,

    active_handler: Option<(String, Arc<dyn CircuitsHandler>)>,
}

#[async_trait]
impl ProvingService for LocalProver {
    fn is_local(&self) -> bool {
        true
    }
    async fn get_vks(&self, req: GetVkRequest) -> GetVkResponse {
        let mut vks = vec![];
        for hard_fork_name in self.config.circuits.keys() {
            let handler = self.new_handler(hard_fork_name);
            for proof_type in &req.proof_types {
                let vk = handler.get_vk(*proof_type).await;

                if let Some(vk) = vk {
                    vks.push(base64::encode(vk));
                }
            }
        }

        GetVkResponse { vks, error: None }
    }
    async fn prove(&mut self, req: ProveRequest) -> ProveResponse {
        self.set_active_handler(&req.hard_fork_name);
        match self
            .do_prove(req, self.active_handler.as_ref().unwrap().1.clone())
            .await
        {
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
        Self {
            config,
            next_task_id: 0,
            current_task: None,
            active_handler: None,
        }
    }

    async fn do_prove(
        &mut self,
        req: ProveRequest,
        handler: Arc<dyn CircuitsHandler>,
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

    fn set_active_handler(&mut self, hard_fork_name: &str) {
        if let Some(handler) = &self.active_handler {
            if handler.0 == hard_fork_name {
                return;
            }
        }
        self.active_handler = Some((hard_fork_name.to_string(), self.new_handler(hard_fork_name)));
    }

    fn new_handler(&self, hard_fork_name: &str) -> Arc<dyn CircuitsHandler> {
        // if we got assigned a task for an unknown hard fork, there is something wrong in the
        // coordinator
        let config = self.config.circuits.get(hard_fork_name).unwrap();

        match hard_fork_name {
            "euclid" => Arc::new(Arc::new(Mutex::new(EuclidHandler::new(
                &config.workspace_path,
            )))) as Arc<dyn CircuitsHandler>,
            "euclidV2" => Arc::new(Arc::new(Mutex::new(EuclidV2Handler::new(
                &config.workspace_path,
            )))) as Arc<dyn CircuitsHandler>,
            _ => unreachable!(),
        }
    }
}
