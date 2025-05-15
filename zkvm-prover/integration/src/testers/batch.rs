use std::path::Path;

use scroll_zkvm_prover::{
    BatchProverType, ChunkProof, ProverType, task::ProvingTask, utils::read_json_deep,
};

use crate::{
    ProverTester,
    testers::{PATH_TESTDATA, chunk::ChunkProverTester},
    utils::{build_batch_task, phase_base_directory},
};

pub struct BatchProverTester;

impl ProverTester for BatchProverTester {
    type Prover = BatchProverType;

    const PATH_PROJECT_ROOT: &str = "./../circuits/batch-circuit";

    const DIR_ASSETS: &str = "batch";

    fn gen_proving_task() -> eyre::Result<<Self::Prover as ProverType>::ProvingTask> {
        Ok(read_json_deep(
            Path::new(PATH_TESTDATA)
                .join(phase_base_directory())
                .join("tasks")
                .join("batch-task.json"),
        )?)
    }
}

pub struct BatchTaskBuildingTester;

impl ProverTester for BatchTaskBuildingTester {
    type Prover = BatchProverType;

    const PATH_PROJECT_ROOT: &str = "./../circuits/batch-circuit";

    const DIR_ASSETS: &str = "batch";

    fn gen_proving_task() -> eyre::Result<<Self::Prover as ProverType>::ProvingTask> {
        let chunk_task = ChunkProverTester::gen_proving_task()?;

        let proof_path = Path::new(PATH_TESTDATA)
            .join(phase_base_directory())
            .join("proofs")
            .join(format!("chunk-{}.json", chunk_task.identifier()));
        println!("proof_path: {:?}", proof_path);

        let chunk_proof = read_json_deep::<_, ChunkProof>(&proof_path)?;

        let task = build_batch_task(&[chunk_task], &[chunk_proof], Default::default());
        Ok(task)
    }
}

#[test]
fn batch_task_parsing() {
    use scroll_zkvm_prover::task::ProvingTask;

    let task = BatchProverTester::gen_proving_task().unwrap();

    let _ = task.build_guest_input().unwrap();
}
