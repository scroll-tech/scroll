use super::{ProofVerifier, TaskType};

use anyhow::Result;

use crate::utils::panic_catch;
use prover_v4::{
    aggregator::Verifier as AggVerifier, zkevm::Verifier, BatchProof, BundleProof, ChunkProof,
};
use std::env;

pub struct DarwinVerifier {
    verifier: Verifier,
    agg_verifier: AggVerifier,
}

impl DarwinVerifier {
    #[allow(dead_code)]
    pub fn new(params_dir: &str, assets_dir: &str) -> Self {
        env::set_var("SCROLL_PROVER_ASSETS_DIR", assets_dir);
        let verifier = Verifier::from_dirs(params_dir, assets_dir);

        let agg_verifier = AggVerifier::from_dirs(params_dir, assets_dir);

        Self {
            verifier,
            agg_verifier,
        }
    }
}

impl ProofVerifier for DarwinVerifier {
    fn verify(&self, task_type: super::TaskType, proof: Vec<u8>) -> Result<bool> {
        let result = panic_catch(|| match task_type {
            TaskType::Chunk => {
                let proof = serde_json::from_slice::<ChunkProof>(proof.as_slice()).unwrap();
                self.verifier.verify_chunk_proof(proof)
            }
            TaskType::Batch => {
                let proof = serde_json::from_slice::<BatchProof>(proof.as_slice()).unwrap();
                self.agg_verifier.verify_batch_proof(&proof)
            }
            TaskType::Bundle => {
                let proof = serde_json::from_slice::<BundleProof>(proof.as_slice()).unwrap();
                self.agg_verifier.verify_bundle_proof(proof)
            }
        });
        result.map_err(|e| anyhow::anyhow!(e))
    }
}
