use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize)]
pub struct Response<T> {
    pub errcode: i32,
    pub errmsg: String,
    pub data: Option<T>,
}

#[derive(Serialize, Deserialize)]
pub struct LoginMessage {
    pub challenge: String,
    pub prover_name: String,
    pub prover_version: String,
    pub hard_fork_name: String,
}

impl LoginMessage {
    pub fn hash() -> Result<Vec<u8>> {

    }

    pub fn sign_with_key() -> Result<String> {

    }
}

#[derive(Serialize, Deserialize)]
pub struct LoginRequest {
    pub message: LoginMessage,
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
    pub task_type: crate::types::ProofType,
    pub prover_height: Option<u64>,
    pub vks: Vec<String>,
}

#[derive(Serialize, Deserialize)]
pub struct GetTaskResponseData {
    pub uuid: String,
    pub task_id: String,
    pub task_type: crate::types::ProofType,
    pub task_data: String,
}

#[derive(Serialize, Deserialize)]
pub struct SubmitProofRequest {
    pub uuid: String,
    pub task_id: String,
    pub task_type: i32,
    pub status: i32,
    pub proof: String,
    pub failure_type: Option<i32>,
    pub failure_msg: Option<String>,
}

#[derive(Serialize, Deserialize)]
pub struct SubmitProofResponseData {}