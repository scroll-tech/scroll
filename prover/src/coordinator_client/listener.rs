use super::SubmitProofRequest;

pub trait Listener {
    fn on_proof_submitted(&self, req: &SubmitProofRequest);
}
