use serde::{Deserialize, Serialize};

// Represents the result of a chunk proof checking operation.
// `ok` indicates whether the proof checking was successful.
// `error` provides additional details in case the check failed.
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct CheckChunkProofsResponse {
    ok: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}

// Encapsulates the result from generating a proof.
// `message` holds the generated proof in byte slice format.
// `error` provides additional details in case the proof generation failed.
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ProofResult {
    #[serde(skip_serializing_if = "Option::is_none")]
    message: Option<Vec<u8>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}
