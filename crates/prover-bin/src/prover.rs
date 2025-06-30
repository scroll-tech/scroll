use crate::zk_circuits_handler::{euclidV2::EuclidV2Handler, CircuitsHandler};
use async_trait::async_trait;
use eyre::Result;
use scroll_proving_sdk::{
    config::Config as SdkConfig,
    prover::{
        proving_service::{
            GetVkRequest, GetVkResponse, ProveRequest, ProveResponse, QueryTaskRequest,
            QueryTaskResponse, TaskStatus,
        },
        types::ProofType,
        ProvingService,
    },
};
use serde::{Deserialize, Serialize};
use std::{
    collections::HashMap,
    fs::File,
    sync::{Arc, OnceLock},
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
        serde_json::from_reader(reader).map_err(|e| eyre::eyre!(e))
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
    /// cached vk value to save some initial cost, for debugging only
    #[serde(default)]
    pub vks: HashMap<ProofType, String>,
}

pub struct LocalProver {
    config: LocalProverConfig,
    next_task_id: u64,
    current_task: Option<JoinHandle<Result<String>>>,

    handlers: HashMap<String, OnceLock<Arc<dyn CircuitsHandler>>>,
}

#[async_trait]
impl ProvingService for LocalProver {
    fn is_local(&self) -> bool {
        true
    }
    async fn get_vks(&self, req: GetVkRequest) -> GetVkResponse {
        let mut vks = vec![];
        for (hard_fork_name, cfg) in self.config.circuits.iter() {
            for proof_type in &req.proof_types {
                if let Some(vk) = cfg.vks.get(proof_type) {
                    vks.push(vk.clone())
                } else {
                    let handler = self.get_or_init_handler(hard_fork_name);
                    vks.push(handler.get_vk(*proof_type));
                }
            }
        }

        GetVkResponse { vks, error: None }
    }
    async fn prove(&mut self, req: ProveRequest) -> ProveResponse {
        let handler = self.get_or_init_handler(&req.hard_fork_name);
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
        let handlers = config
            .circuits
            .keys()
            .map(|k| (k.clone(), OnceLock::new()))
            .collect();
        Self {
            config,
            next_task_id: 0,
            current_task: None,
            handlers,
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

    fn get_or_init_handler(&self, hard_fork_name: &str) -> Arc<dyn CircuitsHandler> {
        let lk = self
            .handlers
            .get(hard_fork_name)
            .expect("coordinator should never sent unexpected forkname");
        lk.get_or_init(|| self.new_handler(hard_fork_name)).clone()
    }

    pub fn new_handler(&self, hard_fork_name: &str) -> Arc<dyn CircuitsHandler> {
        // if we got assigned a task for an unknown hard fork, there is something wrong in the
        // coordinator
        let config = self.config.circuits.get(hard_fork_name).unwrap();

        match hard_fork_name {
            // The new EuclidV2Handler is a universal handler
            // We can add other handler implements if needed
            "some future forkname" => unreachable!(),
            _ => Arc::new(Arc::new(Mutex::new(EuclidV2Handler::new(config))))
                as Arc<dyn CircuitsHandler>,
        }
    }
}
