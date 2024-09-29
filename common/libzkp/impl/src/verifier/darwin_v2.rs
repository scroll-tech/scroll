use super::{ProofVerifier, TaskType};

use anyhow::Result;
use halo2_proofs::{halo2curves::bn256::Bn256, poly::kzg::commitment::ParamsKZG};

use crate::utils::panic_catch;
use prover_v5::{
    aggregator::Verifier as AggVerifier, zkevm::Verifier, BatchProof, BundleProof, ChunkProof,
};
use std::{collections::BTreeMap, env};

pub struct DarwinV2Verifier<'params> {
    verifier: Verifier<'params>,
    agg_verifier: AggVerifier<'params>,
}

impl<'params> DarwinV2Verifier<'params> {
    pub fn new(params_map: &'params BTreeMap<u32, ParamsKZG<Bn256>>, assets_dir: &str) -> Self {
        env::set_var("SCROLL_PROVER_ASSETS_DIR", assets_dir);
        let verifier = Verifier::from_params_and_assets(params_map, assets_dir);
        let agg_verifier = AggVerifier::from_params_and_assets(params_map, assets_dir);

        Self {
            verifier,
            agg_verifier,
        }
    }
}

impl<'params> ProofVerifier for DarwinV2Verifier<'params> {
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
