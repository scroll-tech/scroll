use anyhow::Result;
pub use prover::{BatchProof, BlockTrace, ChunkHash, ChunkProof, Proof};

pub use prover_next::{
    BatchProof as NextBatchProof,
    BlockTrace as NextBlockTrace,
    ChunkHash as NextChunkHash,
    ChunkProof as NextChunkProof,
    Proof as NextProof
};

pub fn chunk_proof_next_to_base(next: NextChunkProof) -> Result<ChunkProof> {
    let proof_bytes = serde_json::to_string(&next.proof)?;
    let proof: Proof = serde_json::from_str(&proof_bytes)?;

    let chunk_hash = next.chunk_hash.map(|hash| {
        ChunkHash {
            chain_id: hash.chain_id,
            prev_state_root: hash.prev_state_root,
            post_state_root: hash.post_state_root,
            withdraw_root: hash.withdraw_root,
            data_hash: hash.data_hash,
            tx_bytes: hash.tx_bytes,
            is_padding: hash.is_padding,
        }
    });
    Ok(ChunkProof {
        protocol: next.protocol,
        proof,
        chunk_hash,
    })
}

pub fn batch_proof_next_to_base(next: NextBatchProof) -> Result<BatchProof> {
    let proof_bytes = serde_json::to_string(&next)?;
    serde_json::from_str(&proof_bytes).map_err(|err| anyhow::anyhow!(err))
}

pub fn chunk_proof_base_to_next(base: &ChunkProof) -> Result<NextChunkProof> {
    let proof_bytes = serde_json::to_string(&base.proof)?;
    let proof: NextProof = serde_json::from_str(&proof_bytes)?;

    let chunk_hash = base.chunk_hash.clone().map(|hash| {
        NextChunkHash {
            chain_id: hash.chain_id,
            prev_state_root: hash.prev_state_root,
            post_state_root: hash.post_state_root,
            withdraw_root: hash.withdraw_root,
            data_hash: hash.data_hash,
            tx_bytes: hash.tx_bytes,
            is_padding: hash.is_padding,
        }
    });
    Ok(NextChunkProof {
        protocol: base.protocol.clone(),
        proof,
        chunk_hash,
    })
}

pub fn chunk_hash_base_to_next(base: ChunkHash) -> NextChunkHash {
    NextChunkHash {
        chain_id: base.chain_id,
        prev_state_root: base.prev_state_root,
        post_state_root: base.post_state_root,
        withdraw_root: base.withdraw_root,
        data_hash: base.data_hash,
        tx_bytes: base.tx_bytes,
        is_padding: base.is_padding,
    }
}

pub fn block_trace_base_to_next(base: BlockTrace) -> Result<NextBlockTrace> {
    let trace_bytes = serde_json::to_string(&base)?;
    serde_json::from_str(&trace_bytes).map_err(|err| anyhow::anyhow!(err))
}