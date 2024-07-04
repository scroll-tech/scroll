use super::errors::ErrorCode;
use crate::types::{ProofFailureType, ProofStatus, ProverType, TaskType};
use rlp::{Encodable, RlpStream};
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

impl Encodable for LoginMessage {
    fn rlp_append(&self, s: &mut RlpStream) {
        let num_fields = 5;
        s.begin_list(num_fields);
        s.append(&self.challenge);
        s.append(&self.prover_version);
        s.append(&self.prover_name);
        // The ProverType in go side is an type alias of uint8
        // A uint8 slice is treated as a string when doing the rlp encoding
        let prover_types = self
            .prover_types
            .iter()
            .map(|prover_type: &ProverType| prover_type.to_u8())
            .collect::<Vec<u8>>();
        s.append(&prover_types);
        s.begin_list(self.vks.len());
        for vk in &self.vks {
            s.append(vk);
        }
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
