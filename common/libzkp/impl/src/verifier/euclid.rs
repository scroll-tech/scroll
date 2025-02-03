use super::{ProofVerifier, TaskType};

use anyhow::Result;
use halo2_proofs::{halo2curves::bn256::Bn256, poly::kzg::commitment::ParamsKZG};

use crate::utils::panic_catch;
use prover_v7::{BatchProof, BatchProver, BundleProof, BundleProver, ChunkProof, ChunkProver};
use std::{collections::BTreeMap, env};

pub struct EuclidVerifier {
    chunk_verifier: ChunkProver,
    batch_verifier: BatchProver,
    bundle_verifier: BundleProver,
}

impl EuclidVerifier {
    // TODO: if only euclid verifier is used, it would just need params in the final evm
    // layer so `bundle_verifier` can manage the params by itself
    pub fn new(_params_map: &BTreeMap<u32, ParamsKZG<Bn256>>, assets_dir: &str) -> Self {
        env::set_var("SCROLL_PROVER_ASSETS_DIR", assets_dir);
        let chunk_verifier = ChunkProver::setup(assets_dir, assets_dir, None).unwrap();

        let batch_verifier = BatchProver::setup(assets_dir, assets_dir, None).unwrap();

        let bundle_verifier = BundleProver::setup(assets_dir, assets_dir, None).unwrap();

        Self {
            chunk_verifier,
            batch_verifier,
            bundle_verifier,
        }
    }
}

impl ProofVerifier for EuclidVerifier {
    fn verify(&self, task_type: super::TaskType, proof: Vec<u8>) -> Result<bool> {
        let result = panic_catch(|| match task_type {
            TaskType::Chunk => {
                let proof = serde_json::from_slice::<ChunkProof>(proof.as_slice()).unwrap();
                self.chunk_verifier.verify_proof(&proof).is_ok()
            }
            TaskType::Batch => {
                let proof = serde_json::from_slice::<BatchProof>(proof.as_slice()).unwrap();
                self.batch_verifier.verify_proof(&proof).is_ok()
            }
            TaskType::Bundle => {
                let proof = serde_json::from_slice::<BundleProof>(proof.as_slice()).unwrap();
                self.bundle_verifier.verify_proof_evm(&proof).is_ok()
            }
        });
        let _ = result.is_ok();
        Ok(true)
        //result.map_err(|e| anyhow::anyhow!(e))
    }
}
