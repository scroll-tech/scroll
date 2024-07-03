use super::errors::ErrorCode;
use crate::types::{ProofFailureType, ProofStatus, ProverType, TaskType};
use rlp::RlpStream;
use serde::{Deserialize, Serialize};

#[derive(Deserialize)]
pub struct Response<T> {
    pub errcode: ErrorCode,
    pub errmsg: String,
    pub data: Option<T>,
}

#[derive(Serialize, Deserialize)]
pub struct LoginMessage {
    pub challenge: String,
    pub prover_name: String,
    pub prover_version: String,
    pub prover_types: Vec<ProverType>,
    pub vks: Vec<String>,
}

impl LoginMessage {
    pub fn rlp(&self) -> Vec<u8> {
        let mut rlp = RlpStream::new();
        let num_fields = 5;
        rlp.begin_list(num_fields);
        rlp.append(&self.prover_name);
        rlp.append(&self.prover_version);
        rlp.append(&self.challenge);
        rlp.out().freeze().into()
    }
}

#[derive(Serialize, Deserialize)]
pub struct LoginRequest {
    pub message: LoginMessage,
    pub public_key: String,
    pub signature: String,
}

#[derive(Serialize, Deserialize)]
pub struct LoginResponseData {
    pub time: String,
    pub token: String,
}

pub type ChallengeResponseData = LoginResponseData;

#[derive(Default, Serialize, Deserialize)]
pub struct GetTaskRequest {
    pub task_types: Vec<TaskType>,
    pub prover_height: Option<u64>,
}

#[derive(Serialize, Deserialize)]
pub struct GetTaskResponseData {
    pub uuid: String,
    pub task_id: String,
    pub task_type: TaskType,
    pub task_data: String,
    pub hard_fork_name: String,
}

#[derive(Serialize, Deserialize, Default)]
pub struct SubmitProofRequest {
    pub uuid: String,
    pub task_id: String,
    pub task_type: TaskType,
    pub status: ProofStatus,
    pub proof: String,
    pub failure_type: Option<ProofFailureType>,
    pub failure_msg: Option<String>,
}

#[derive(Serialize, Deserialize)]
pub struct SubmitProofResponseData {}
