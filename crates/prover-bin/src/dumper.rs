use async_trait::async_trait;
use libzkp::ProvingTaskExt;
use scroll_zkvm_types::ProvingTask;
use scroll_proving_sdk::{
    prover::{
        proving_service::{
            GetVkRequest, GetVkResponse, ProveRequest, ProveResponse, QueryTaskRequest,
            QueryTaskResponse, TaskStatus,
        },
        ProvingService,
    },
};

#[derive(Default)]
pub struct Dumper {
    target_path: String
}

impl Dumper {
    fn dump(input_string: &str) -> eyre::Result<()> {
        let task : ProvingTaskExt = serde_json::from_str(input_string)?;
        let task = ProvingTask::from(task);
        

        Ok(())
    }
}

#[async_trait]
impl ProvingService for Dumper {
    fn is_local(&self) -> bool {
        true
    }
    async fn get_vks(&self, _: GetVkRequest) -> GetVkResponse {
        // get vk has been deprecated in new prover with dynamic asset loading scheme
        GetVkResponse {
            vks: vec![],
            error: None,
        }
    }
    async fn prove(&mut self, req: ProveRequest) -> ProveResponse {
        

        // match self.do_prove(req).await {
        //     Ok(resp) => resp,
        //     Err(e) => ProveResponse {
        //         status: TaskStatus::Failed,
        //         error: Some(format!("failed to request proof: {}", e)),
        //         ..Default::default()
        //     },
        // }
        ProveResponse {
            status: TaskStatus::Failed,
            error: Some(format!("failed to request proof: {}", e)),
            ..Default::default()
        }  
    }

    async fn query_task(&mut self, req: QueryTaskRequest) -> QueryTaskResponse {
        unreachable!("");
    }
}
