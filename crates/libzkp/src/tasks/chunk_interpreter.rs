use eyre::Result;
use sbv_primitives::{types::BlockWitness, Bytes, B256};

/// An interpreter which is cirtical in translating chunk data
/// since we need to grep block witness and storage node data
/// (in rare case) from external
pub trait ChunkInterpreter {
    fn try_fetch_block_witness(
        &self,
        _block_hash: B256,
        _prev_witness: Option<&BlockWitness>,
    ) -> Result<BlockWitness> {
        Err(eyre::eyre!("no implement"))
    }
    fn try_fetch_storage_node(&self, _node_hash: B256) -> Result<Bytes> {
        Err(eyre::eyre!("no implement"))
    }
}

pub trait TryFromWithInterpreter<T>: Sized {
    fn try_from_with_interpret(value: T, intepreter: impl ChunkInterpreter) -> Result<Self>;
}

pub struct DummyInterpreter {}

impl ChunkInterpreter for DummyInterpreter {}
