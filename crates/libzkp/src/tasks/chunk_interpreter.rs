use eyre::Result;
use sbv_core::BlockWitness;
use sbv_primitives::{B256, Bytes, types::consensus::TxL1Message};

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

    fn try_fetch_l1_msgs(&self, _block_number: u64) -> Result<Vec<TxL1Message>> {
        Err(eyre::eyre!("no implement"))
    }
}

pub trait TryFromWithInterpreter<T>: Sized {
    fn try_from_with_interpret(
        value: T,
        decryption_key: Option<&[u8]>,
        intepreter: impl ChunkInterpreter,
    ) -> Result<Self>;
}

pub struct DummyInterpreter {}

impl ChunkInterpreter for DummyInterpreter {}
