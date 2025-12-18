use async_trait::async_trait;
use libzkp::ProvingTaskExt;
use scroll_proving_sdk::prover::{
    proving_service::{
        GetVkRequest, GetVkResponse, ProveRequest, ProveResponse, QueryTaskRequest,
        QueryTaskResponse, TaskStatus,
    },
    ProvingService,
};
use scroll_zkvm_types::ProvingTask;

#[derive(Default)]
pub struct Dumper {
    #[allow(dead_code)]
    target_path: String,
}

impl Dumper {
    fn dump(&self, input_string: &str) -> eyre::Result<()> {
        let task: ProvingTaskExt = serde_json::from_str(input_string)?;
        let task = ProvingTask::from(task);

        // stream-encode serialized_witness to input_task.bin using bincode 2.0
        let input_file = std::fs::File::create("input_task.bin")?;
        let mut input_writer = std::io::BufWriter::new(input_file);
        bincode::encode_into_std_write(
            &task.serialized_witness,
            &mut input_writer,
            bincode::config::standard(),
        )?;

        // stream-encode aggregated_proofs to agg_proofs.bin using bincode 2.0
        let agg_file = std::fs::File::create("agg_proofs.bin")?;
        let mut agg_writer = std::io::BufWriter::new(agg_file);
        for proof in &task.aggregated_proofs {
            bincode::serde::encode_into_std_write(
                &proof.proofs,
                &mut agg_writer,
                bincode::config::standard(),
            )?;
        }

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
        let error = if let Err(e) = self.dump(&req.input) {
            Some(format!("failed to dump: {}", e))
        } else {
            None
        };

        ProveResponse {
            status: TaskStatus::Failed,
            error,
            ..Default::default()
        }
    }

    async fn query_task(&mut self, req: QueryTaskRequest) -> QueryTaskResponse {
        QueryTaskResponse {
            task_id: req.task_id,
            status: TaskStatus::Failed,
            error: Some("dump file finished but need a fail return to exit".to_string()),
            ..Default::default()
        }
    }
}
