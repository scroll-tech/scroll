use async_trait::async_trait;
use anyhow::{Context, Result};
use tokio::{task::JoinHandle, runtime::Runtime};
use std::{sync::{Arc, Mutex}, cell::RefCell};
use scroll_proving_sdk::{
    prover::{
        proving_service::{
            GetVkRequest, GetVkResponse, ProveRequest, ProveResponse, QueryTaskRequest,
            QueryTaskResponse, TaskStatus,
        },
        CircuitType, ProvingService,
    }, config::{ProverConfig, LocalProverConfig},
};
use crate::{
    utils::get_prover_type,
    zk_circuits_handler::{CircuitsHandler, CircuitsHandlerProvider}, types::ProverType,
};
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::sync::RwLock;


pub struct LocalProver {
    config: LocalProverConfig,
    circuits_handler_provider: RwLock<CircuitsHandlerProvider>,
    current_task: Arc<Mutex<Option<JoinHandle<Result<String>>>>>,
    next_task_id: Arc<Mutex<u64>>,
}

#[async_trait]
impl ProvingService for LocalProver {
    fn is_local(&self) -> bool {
        true
    }
    async fn get_vks(&self, req: GetVkRequest) -> GetVkResponse {
        let mut prover_types = vec![];
        req.circuit_types.iter().for_each(|circuit_type| {
            match get_prover_type(*circuit_type) {
                Some(pt) => {
                    if !prover_types.contains(&pt) {
                        prover_types.push(pt);
                    }
                }
                None => {}
            }
        });

        let local_prover_config = self.config.clone();
        let vks = self.circuits_handler_provider.read().await.init_vks(&local_prover_config, prover_types).await;
        GetVkResponse {
            vks: vks,
            error: None
        }
    }
    async fn prove(&self, req: ProveRequest) -> ProveResponse {
        let prover_type = match get_prover_type(req.circuit_type) {
            Some(pt) => pt,
            None => {
                return build_prove_error_response(&req, "unsupported prover_type")
            }
        };
        let handler = self
            .circuits_handler_provider
            .write()
            .await
            .get_circuits_handler(&req.hard_fork_name, prover_type)
            .context("failed to get circuit handler").unwrap();
        
        match self
            .do_prove(req.clone(), handler) 
        {
            Ok(resp) => resp,
            Err(e) => {
                return build_prove_error_response(&req, &format!("Failed to request proof: {}", e))
            }
        }

    }

    async fn query_task(&self, req: QueryTaskRequest) -> QueryTaskResponse {
        let mut current_task = self.current_task.lock().unwrap();

        if let Some(handle) = current_task.take() {
            if handle.is_finished() {
                let result = Runtime::new().unwrap().block_on(handle).unwrap();
                match result {
                    Ok(proof) => {
                        return QueryTaskResponse {
                            task_id: "".to_string(),
                            circuit_type: CircuitType::Undefined,
                            circuit_version: "".to_string(),
                            hard_fork_name: "".to_string(),
                            status: TaskStatus::Success,
                            created_at: 0.0,
                            started_at: None,
                            finished_at: None,
                            compute_time_sec: None,
                            input: None,
                            proof: Some(proof),
                            vk: None,
                            error: None,
                        };
                    }
                    Err(e) => {
                        return QueryTaskResponse {
                            task_id: "".to_string(),
                            circuit_type: CircuitType::Undefined,
                            circuit_version: "".to_string(),
                            hard_fork_name: "".to_string(),
                            status: TaskStatus::Failed,
                            created_at: 0.0,
                            started_at: None,
                            finished_at: None,
                            compute_time_sec: None,
                            input: None,
                            proof: None,
                            vk: None,
                            error: Some(e.to_string()),
                        };
                    } 
                }

            } else {
                *current_task = Some(handle);
                return QueryTaskResponse {
                    task_id: "".to_string(),
                    circuit_type: CircuitType::Undefined,
                    circuit_version: "".to_string(),
                    hard_fork_name: "".to_string(),
                    status: TaskStatus::Proving,
                    created_at: 0.0,
                    started_at: None,
                    finished_at: None,
                    compute_time_sec: None,
                    input: None,
                    proof: None,
                    vk: None,
                    error: None,
                };
            }
        } else {
            return QueryTaskResponse {
                    task_id: "".to_string(),
                    circuit_type: CircuitType::Undefined,
                    circuit_version: "".to_string(),
                    hard_fork_name: "".to_string(),
                    status: TaskStatus::Failed,
                    created_at: 0.0,
                    started_at: None,
                    finished_at: None,
                    compute_time_sec: None,
                    input: None,
                    proof: None,
                    vk: None,
                    error: None,
            };
        }

    }
}

impl LocalProver {
    pub fn new(config: LocalProverConfig) -> Self {

        let circuits_handler_provider = CircuitsHandlerProvider::new(config.clone())
        .context("failed to create circuits handler provider").unwrap();

        Self {
            config,
            circuits_handler_provider: RwLock::new(circuits_handler_provider),
            current_task: Arc::new(Mutex::new(None)),
            next_task_id: Arc::new(Mutex::new(0)),
        }
    }

    fn do_prove(&self, req: ProveRequest, handler: Arc<Box<dyn CircuitsHandler>>) -> Result<ProveResponse> {
        let mut current_task = self.current_task.lock().unwrap();
        if current_task.is_some() {
            return Err(anyhow::Error::msg("prover working on previous task"));
        }

        let task_id = {
            let mut next_task_id = self.next_task_id.lock().unwrap();
            *next_task_id += 1;
            *next_task_id
        };

        let req_clone = req.clone();
        let handle = tokio::spawn(async move {
            handler.get_proof_data(req_clone).await

        });
        *current_task = Some(handle);

        let duration = SystemTime::now().duration_since(UNIX_EPOCH).unwrap();
        let created_at = duration.as_secs() as f64 + duration.subsec_nanos() as f64 * 1e-9;

        Ok(
            ProveResponse {
            task_id: task_id.to_string(),
            circuit_type: req.circuit_type,
            circuit_version: req.circuit_version.clone(),
            hard_fork_name: req.hard_fork_name.clone(),
            status: TaskStatus::Proving,
            created_at,
            started_at: None,
            finished_at: None,
            compute_time_sec: None,
            input: Some(req.input),
            proof: None,
            vk: None,
            error: None,
        })
    }
}

fn build_prove_error_response(req: &ProveRequest, error_msg: &str) -> ProveResponse {
    ProveResponse {
        task_id: String::new(),
        circuit_type: req.circuit_type,
        circuit_version: req.circuit_version.clone(),
        hard_fork_name: req.hard_fork_name.clone(),
        status: TaskStatus::Failed,
        created_at: 0.0,
        started_at: None,
        finished_at: None,
        compute_time_sec: None,
        input: Some(req.input.clone()),
        proof: None,
        vk: None,
        error: Some(error_msg.to_string()),
    }
}