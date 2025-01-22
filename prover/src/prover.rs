use crate::{
    types::ProverType,
    utils::get_prover_type,
    zk_circuits_handler::{CircuitsHandler, CircuitsHandlerProvider},
};
use anyhow::Result;
use async_trait::async_trait;
use scroll_proving_sdk::{
    config::LocalProverConfig,
    prover::{
        proving_service::{
            GetVkRequest, GetVkResponse, ProveRequest, ProveResponse, QueryTaskRequest,
            QueryTaskResponse, TaskStatus,
        },
        ProvingService,
    },
};
use std::{
    sync::{Arc, Mutex},
    time::{SystemTime, UNIX_EPOCH},
};
use tokio::{runtime::Handle, sync::RwLock, task::JoinHandle};

pub struct LocalProver {
    config: LocalProverConfig,
    prover_types: Vec<ProverType>,
    circuits_handler_provider: RwLock<CircuitsHandlerProvider>,
    next_task_id: Arc<Mutex<u64>>,
    current_task: Arc<Mutex<Option<JoinHandle<Result<String>>>>>,
}

#[async_trait]
impl ProvingService for LocalProver {
    fn is_local(&self) -> bool {
        true
    }
    async fn get_vks(&self, req: GetVkRequest) -> GetVkResponse {
        let mut prover_types = vec![];
        req.circuit_types.iter().for_each(|circuit_type| {
            if let Some(pt) = get_prover_type(*circuit_type) {
                if !prover_types.contains(&pt) {
                    prover_types.push(pt);
                }
            }
        });

        let vks = self
            .circuits_handler_provider
            .read()
            .await
            .init_vks(&self.config, prover_types)
            .await;
        GetVkResponse { vks, error: None }
    }
    async fn prove(&self, req: ProveRequest) -> ProveResponse {
        let handler = self
            .circuits_handler_provider
            .write()
            .await
            .get_circuits_handler(&req.hard_fork_name, self.prover_types.clone())
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

    async fn query_task(&self, req: QueryTaskRequest) -> QueryTaskResponse {
        let handle = self.current_task.lock().unwrap().take();
        if let Some(handle) = handle {
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
                *self.current_task.lock().unwrap() = Some(handle);
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
    pub fn new(config: LocalProverConfig, prover_types: Vec<ProverType>) -> Self {
        let circuits_handler_provider = CircuitsHandlerProvider::new(config.clone())
            .expect("failed to create circuits handler provider");

        Self {
            config,
            prover_types,
            circuits_handler_provider: RwLock::new(circuits_handler_provider),
            next_task_id: Arc::new(Mutex::new(0)),
            current_task: Arc::new(Mutex::new(None)),
        }
    }

    async fn do_prove(
        &self,
        req: ProveRequest,
        handler: Arc<Box<dyn CircuitsHandler>>,
    ) -> Result<ProveResponse> {
        let task_id = {
            let mut next_task_id = self.next_task_id.lock().unwrap();
            *next_task_id += 1;
            *next_task_id
        };

        let duration = SystemTime::now().duration_since(UNIX_EPOCH).unwrap();
        let created_at = duration.as_secs() as f64 + duration.subsec_nanos() as f64 * 1e-9;

        let req_clone = req.clone();
        let handle = Handle::current();
        let task_handle =
            tokio::task::spawn_blocking(move || handle.block_on(handler.get_proof_data(req_clone)));

        *self.current_task.lock().unwrap() = Some(task_handle);

        Ok(ProveResponse {
            task_id: task_id.to_string(),
            circuit_type: req.circuit_type,
            circuit_version: req.circuit_version,
            hard_fork_name: req.hard_fork_name,
            status: TaskStatus::Proving,
            created_at,
            input: Some(req.input),
            ..Default::default()
        })
    }
}
