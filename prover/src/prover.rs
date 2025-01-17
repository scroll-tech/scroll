use crate::{
    types::ProverType,
    utils::get_prover_type,
    zk_circuits_handler::{CircuitsHandler, CircuitsHandlerProvider},
};
use anyhow::{Context, Result};
use async_trait::async_trait;
use scroll_proving_sdk::{
    config::LocalProverConfig,
    prover::{
        proving_service::{
            GetVkRequest, GetVkResponse, ProveRequest, ProveResponse, QueryTaskRequest,
            QueryTaskResponse, TaskStatus,
        },
        CircuitType, ProvingService,
    },
};
use std::{
    sync::{Arc, Mutex},
    time::{SystemTime, UNIX_EPOCH},
};
use tokio::sync::RwLock;

pub struct LocalProver {
    config: LocalProverConfig,
    prover_types: Vec<ProverType>,
    circuits_handler_provider: RwLock<CircuitsHandlerProvider>,
    next_task_id: Arc<Mutex<u64>>,
    result: Arc<Mutex<Result<String>>>,
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

        let local_prover_config = self.config.clone();
        let vks = self
            .circuits_handler_provider
            .read()
            .await
            .init_vks(&local_prover_config, prover_types)
            .await;
        GetVkResponse { vks, error: None }
    }
    async fn prove(&self, req: ProveRequest) -> ProveResponse {
        let handler = self
            .circuits_handler_provider
            .write()
            .await
            .get_circuits_handler(&req.hard_fork_name, self.prover_types.clone())
            .context("failed to get circuit handler")
            .unwrap();

        match self.do_prove(req.clone(), handler).await {
            Ok(resp) => resp,
            Err(e) => build_prove_error_response(
                String::new(),
                TaskStatus::Failed,
                None,
                String::from(&format!("failed to request proof: {}", e)),
            ),
        }
    }

    async fn query_task(&self, req: QueryTaskRequest) -> QueryTaskResponse {
        let mut result_guard = self.result.lock().unwrap();
        let resp = match result_guard.as_ref() {
            Ok(proof) => build_query_task_response(
                req.task_id,
                TaskStatus::Success,
                Some(proof.clone()),
                None,
            ),
            Err(e) => build_query_task_response(
                req.task_id,
                TaskStatus::Failed,
                None,
                Some(e.to_string()),
            ),
        };
        *result_guard = Err(anyhow::Error::msg("prover not started"));
        resp
    }
}

impl LocalProver {
    pub fn new(config: LocalProverConfig, prover_types: Vec<ProverType>) -> Self {
        let circuits_handler_provider = CircuitsHandlerProvider::new(config.clone())
            .context("failed to create circuits handler provider")
            .unwrap();

        Self {
            config,
            prover_types,
            circuits_handler_provider: RwLock::new(circuits_handler_provider),
            next_task_id: Arc::new(Mutex::new(0)),
            result: Arc::new(Mutex::new(Err(anyhow::Error::msg("prover not started")))),
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
        let result = tokio::task::spawn(async move {
            handler.get_proof_data(req_clone).await
        })
        .await
        .unwrap();

        *self.result.lock().unwrap() = result;

        Ok(ProveResponse {
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

fn build_prove_error_response(
    task_id: String,
    status: TaskStatus,
    proof: Option<String>,
    error_msg: String,
) -> ProveResponse {
    ProveResponse {
        task_id,
        circuit_type: CircuitType::Undefined, // TODO
        circuit_version: "".to_string(),
        hard_fork_name: "".to_string(),
        status,
        created_at: 0.0,
        started_at: None,
        finished_at: None,
        compute_time_sec: None,
        input: None,
        proof,
        vk: None,
        error: Some(error_msg),
    }
}

fn build_query_task_response(
    task_id: String,
    status: TaskStatus,
    proof: Option<String>,
    error_msg: Option<String>,
) -> QueryTaskResponse {
    QueryTaskResponse {
        task_id,
        circuit_type: CircuitType::Undefined, // TODO
        circuit_version: "".to_string(),
        hard_fork_name: "".to_string(),
        status,
        created_at: 0.0,
        started_at: None,
        finished_at: None,
        compute_time_sec: None,
        input: None,
        proof,
        vk: None,
        error: error_msg,
    }
}
