use libzkp::{gen_universal_task, tasks::chunk_interpreter::ChunkInterpreter, TaskType};
use scroll_zkvm_types::ProvingTask;
use std::{fs, path::Path};

// Global constant for testdata directory
const TESTDATA_DIR: &str = "./testdata";

// Mock interpreter for testing chunk tasks
#[derive(Clone)]
struct MockChunkInterpreter;

impl ChunkInterpreter for MockChunkInterpreter {}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_gen_universal_task(task_file: &str, task_type: TaskType) -> eyre::Result<ProvingTask> {
        // Load chunk task JSON from testdata file
        let testdata_path = Path::new(TESTDATA_DIR).join(task_file);
        let task_json = fs::read_to_string(&testdata_path)?;

        // Parse task_json as raw JSON object and extract fork_name field
        let raw_json: serde_json::Value = serde_json::from_str(&task_json)?;
        let fork_name = raw_json["fork_name"]
            .as_str()
            .ok_or_else(|| eyre::eyre!("fork_name field not found or not a string"))?
            .to_lowercase();

        let mocked_vk = vec![0u8; 32]; // Mock verification key
        let interpreter = Some(MockChunkInterpreter);

        let (pi_hash, metadata, task) = gen_universal_task(
            task_type as i32,
            &task_json,
            &fork_name,
            &mocked_vk,
            interpreter,
        )?;

        assert!(!pi_hash.is_zero(), "PI hash should not be zero");
        assert!(!metadata.is_empty(), "Metadata should not be empty");
        assert!(!task.is_empty(), "Task should not be empty");

        // Dump the task content to testdata directory
        let dump_path = Path::new(TESTDATA_DIR).join("dump_univ_task.json");
        std::fs::write(&dump_path, &task).ok(); // Dump task content

        Ok(serde_json::from_str(&task)?)
    }

    #[ignore = "need testing stuff"]
    #[test]
    fn test_gen_universal_task_chunk() {
        let _ = test_gen_universal_task("chunk_proving_task.json", TaskType::Chunk).unwrap();
    }

    #[ignore = "need testing stuff"]
    #[test]
    fn test_gen_universal_task_batch() {
        let _ = test_gen_universal_task("batch_proving_task.json", TaskType::Batch).unwrap();
    }

    #[ignore = "need testing stuff"]
    #[test]
    fn test_gen_universal_task_bundle() {
        let _ = test_gen_universal_task("bundle_proving_task.json", TaskType::Bundle).unwrap();
    }
}
