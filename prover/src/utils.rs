use crate::types::ProverType;
use scroll_proving_sdk::prover::types::CircuitType;

pub fn get_circuit_types(prover_type: ProverType) -> Vec<CircuitType> {
    match prover_type {
        ProverType::Chunk => vec![CircuitType::Chunk],
        ProverType::Batch => vec![CircuitType::Batch, CircuitType::Bundle],
    }
}

pub fn get_prover_type(task_type: CircuitType) -> Option<ProverType> {
    match task_type {
        CircuitType::Undefined => None,
        CircuitType::Chunk => Some(ProverType::Chunk),
        CircuitType::Batch => Some(ProverType::Batch),
        CircuitType::Bundle => Some(ProverType::Batch),
    }
}
