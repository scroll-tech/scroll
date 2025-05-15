use sbv_primitives::{
    B256, U256,
    types::{BlockWitness, Transaction, eips::Encodable2718, reth::TransactionSigned},
};
use scroll_zkvm_circuit_input_types::{
    batch::{BatchHeader, BatchHeaderV6, BatchHeaderV7},
    utils::keccak256,
};
use scroll_zkvm_prover::{
    ChunkProof,
    task::{
        batch::{BatchHeaderV, BatchProvingTask},
        chunk::ChunkProvingTask,
    },
    utils::point_eval,
};
use vm_zstd::zstd_encode;

fn is_l1_tx(tx: &Transaction) -> bool {
    // 0x7e is l1 tx
    tx.transaction_type == 0x7e
}

#[allow(dead_code)]
fn final_l1_index(blk: &BlockWitness) -> u64 {
    // Get number of l1 txs. L1 txs can be skipped, so just counting is not enough
    // (The comment copied from scroll-prover, but why the max l1 queue index is always
    // the last one for a chunk, or, is the last l1 never being skipped?)
    blk.transaction
        .iter()
        .filter(|tx| is_l1_tx(tx))
        .map(|tx| tx.queue_index.expect("l1 msg should has queue index"))
        .max()
        .unwrap_or_default()
}

fn blks_tx_bytes<'a>(blks: impl Iterator<Item = &'a BlockWitness>) -> Vec<u8> {
    blks.flat_map(|blk| &blk.transaction)
        .filter(|tx| !is_l1_tx(tx))
        .fold(Vec::new(), |mut tx_bytes, tx| {
            TransactionSigned::try_from(tx)
                .unwrap()
                .encode_2718(&mut tx_bytes);
            tx_bytes
        })
}

pub fn phase_base_directory() -> &'static str {
    if cfg!(feature = "euclidv2") {
        "phase2"
    } else {
        "phase1"
    }
}

#[derive(Debug)]
pub struct LastHeader {
    pub batch_index: u64,
    pub batch_hash: B256,
    pub version: u8,
    /// legacy field
    pub l1_message_index: u64,
}

impl Default for LastHeader {
    fn default() -> Self {
        // create a default LastHeader according to the dummy value
        // being set in the e2e test in scroll-prover:
        // https://github.com/scroll-tech/scroll-prover/blob/82f8ed3fabee5c3001b0b900cda1608413e621f8/integration/tests/e2e_tests.rs#L203C1-L207C8

        Self {
            batch_index: 123,
            version: if cfg!(feature = "euclidv2") { 7 } else { 6 },
            batch_hash: B256::new([
                0xab, 0xac, 0xad, 0xae, 0xaf, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                0, 0, 0, 0, 0, 0, 0, 0, 0,
            ]),
            l1_message_index: 0,
        }
    }
}

impl From<&BatchHeaderV> for LastHeader {
    fn from(value: &BatchHeaderV) -> Self {
        match value {
            BatchHeaderV::V6(h) => h.into(),
            BatchHeaderV::V7(h) => h.into(),
        }
    }
}

impl From<&BatchHeaderV6> for LastHeader {
    fn from(h: &BatchHeaderV6) -> Self {
        Self {
            batch_index: h.batch_index,
            version: h.version,
            batch_hash: h.batch_hash(),
            l1_message_index: h.total_l1_message_popped,
        }
    }
}

impl From<&BatchHeaderV7> for LastHeader {
    fn from(h: &BatchHeaderV7) -> Self {
        Self {
            batch_index: h.batch_index,
            version: h.version,
            batch_hash: h.batch_hash(),
            l1_message_index: 0,
        }
    }
}

pub fn build_batch_task(
    chunk_tasks: &[ChunkProvingTask],
    chunk_proofs: &[ChunkProof],
    last_header: LastHeader,
) -> BatchProvingTask {
    // Sanity check.
    assert_eq!(chunk_tasks.len(), chunk_proofs.len());

    // collect tx bytes from chunk tasks
    let (meta_chunk_sizes, chunk_digests, chunk_tx_bytes) = chunk_tasks.iter().fold(
        (Vec::new(), Vec::new(), Vec::new()),
        |(mut meta_chunk_sizes, mut chunk_digests, mut payload_bytes), task| {
            let tx_bytes = blks_tx_bytes(task.block_witnesses.iter());
            meta_chunk_sizes.push(tx_bytes.len());
            chunk_digests.push(keccak256(&tx_bytes));
            payload_bytes.extend(tx_bytes);
            (meta_chunk_sizes, chunk_digests, payload_bytes)
        },
    );

    // sanity check
    for (digest, proof) in chunk_digests.iter().zip(chunk_proofs.iter()) {
        assert_eq!(digest, &proof.metadata.chunk_info.tx_data_digest);
    }

    const LEGACY_MAX_CHUNKS: usize = 45;

    let meta_chunk_bytes = {
        let valid_chunk_size = chunk_proofs.len() as u16;
        meta_chunk_sizes
            .into_iter()
            .chain(std::iter::repeat(0))
            .take(LEGACY_MAX_CHUNKS)
            .fold(
                Vec::from(valid_chunk_size.to_be_bytes()),
                |mut bytes, len| {
                    bytes.extend_from_slice(&(len as u32).to_be_bytes());
                    bytes
                },
            )
    };

    // collect all data together for payload
    let mut payload = if cfg!(feature = "euclidv2") {
        Vec::new()
    } else {
        meta_chunk_bytes.clone()
    };
    #[cfg(feature = "euclidv2")]
    {
        let num_blocks = chunk_tasks
            .iter()
            .map(|t| t.block_witnesses.len())
            .sum::<usize>() as u16;
        let (prev_msg_queue_hash, initial_block_number) = {
            let first_chunk = &chunk_proofs
                .first()
                .expect("at least one chunk")
                .metadata
                .chunk_info;
            (
                first_chunk.prev_msg_queue_hash,
                first_chunk.initial_block_number,
            )
        };

        let post_msg_queue_hash = chunk_proofs
            .last()
            .expect("at least one chunk")
            .metadata
            .chunk_info
            .post_msg_queue_hash;
        payload.extend_from_slice(prev_msg_queue_hash.as_slice());
        payload.extend_from_slice(post_msg_queue_hash.as_slice());
        payload.extend(initial_block_number.to_be_bytes());
        payload.extend(num_blocks.to_be_bytes());
        assert_eq!(payload.len(), 74);
        for proof in chunk_proofs {
            for ctx in &proof.metadata.chunk_info.block_ctxs {
                payload.extend(ctx.to_bytes());
            }
        }
        assert_eq!(payload.len(), 74 + 52 * num_blocks as usize);
    }
    payload.extend(chunk_tx_bytes);
    // compress ...
    let compressed_payload = zstd_encode(&payload);

    let version = 7u32;
    let heading = compressed_payload.len() as u32 + (version << 24);

    let blob_bytes = if cfg!(feature = "euclidv2") {
        let mut blob_bytes = Vec::from(heading.to_be_bytes());
        blob_bytes.push(1u8); // compressed flag
        blob_bytes.extend(compressed_payload);
        blob_bytes.resize(4096 * 31, 0);
        blob_bytes
    } else {
        let mut blob_bytes = vec![1];
        blob_bytes.extend(compressed_payload);
        blob_bytes
    };

    let kzg_blob = point_eval::to_blob(&blob_bytes);
    let kzg_commitment = point_eval::blob_to_kzg_commitment(&kzg_blob);
    let blob_versioned_hash = point_eval::get_versioned_hash(&kzg_commitment);

    // primage = keccak(payload) + blob_versioned_hash
    let chg_preimage = if cfg!(feature = "euclidv2") {
        let mut chg_preimage = keccak256(&blob_bytes).to_vec();
        chg_preimage.extend(blob_versioned_hash.0);
        chg_preimage
    } else {
        let mut chg_preimage = Vec::from(keccak256(&meta_chunk_bytes).0);
        let last_digest = chunk_digests.last().expect("at least we have one");
        chg_preimage.extend(
            chunk_digests
                .iter()
                .chain(std::iter::repeat(last_digest))
                .take(LEGACY_MAX_CHUNKS)
                .fold(Vec::new(), |mut ret, digest| {
                    ret.extend_from_slice(&digest.0);
                    ret
                }),
        );
        chg_preimage.extend_from_slice(blob_versioned_hash.as_slice());
        chg_preimage
    };
    let challenge_digest = keccak256(&chg_preimage);

    let x = point_eval::get_x_from_challenge(challenge_digest);
    let (kzg_proof, z) = point_eval::get_kzg_proof(&kzg_blob, challenge_digest);

    #[cfg(feature = "euclidv2")]
    let batch_header = {
        // avoid unused variant warning
        let _ = x + z;
        BatchHeaderV::V7(BatchHeaderV7 {
            version: last_header.version,
            batch_index: last_header.batch_index + 1,
            parent_batch_hash: last_header.batch_hash,
            blob_versioned_hash,
        })
    };

    #[cfg(not(feature = "euclidv2"))]
    let batch_header = {
        // collect required fields for batch header
        let last_l1_message_index: u64 = chunk_tasks
            .iter()
            .flat_map(|t| &t.block_witnesses)
            .map(final_l1_index)
            .reduce(|last, cur| if cur == 0 { last } else { cur })
            .expect("at least one chunk");
        let last_l1_message_index = if last_l1_message_index == 0 {
            last_header.l1_message_index
        } else {
            last_l1_message_index
        };

        let last_block_timestamp = chunk_tasks.last().map_or(0u64, |t| {
            t.block_witnesses
                .last()
                .map_or(0, |trace| trace.header.timestamp)
        });

        let point_evaluations = [x, z];

        let data_hash = keccak256(
            chunk_proofs
                .iter()
                .map(|proof| &proof.metadata.chunk_info.data_hash)
                .fold(Vec::new(), |mut bytes, h| {
                    bytes.extend_from_slice(&h.0);
                    bytes
                }),
        );

        BatchHeaderV::V6(BatchHeaderV6 {
            version: last_header.version,
            batch_index: last_header.batch_index + 1,
            l1_message_popped: last_l1_message_index - last_header.l1_message_index,
            last_block_timestamp,
            total_l1_message_popped: last_l1_message_index,
            parent_batch_hash: last_header.batch_hash,
            data_hash,
            blob_versioned_hash,
            blob_data_proof: point_evaluations.map(|u| B256::new(u.to_be_bytes())),
        })
    };

    BatchProvingTask {
        chunk_proofs: Vec::from(chunk_proofs),
        batch_header,
        blob_bytes,
        challenge_digest: Some(U256::from_be_bytes(challenge_digest.0)),
        kzg_commitment: Some(kzg_commitment.to_bytes()),
        kzg_proof: Some(kzg_proof.to_bytes()),
        fork_name: if cfg!(feature = "euclidv2") {
            String::from("euclidv2")
        } else {
            String::from("euclidv1")
        },
    }
}

#[test]
fn test_build_and_parse_batch_task() -> eyre::Result<()> {
    #[cfg(not(feature = "euclidv2"))]
    use scroll_zkvm_circuit_input_types::batch::{EnvelopeV6 as Envelope, PayloadV6 as Payload};
    #[cfg(feature = "euclidv2")]
    use scroll_zkvm_circuit_input_types::batch::{EnvelopeV7 as Envelope, PayloadV7 as Payload};
    use scroll_zkvm_prover::utils::{read_json, read_json_deep, write_json};

    // ./testdata/
    let path_testdata = std::path::Path::new("testdata");

    // read block witnesses.
    let paths_block_witnesses = if cfg!(feature = "euclidv2") {
        [
            path_testdata.join("1.json"),
            path_testdata.join("2.json"),
            path_testdata.join("3.json"),
            path_testdata.join("4.json"),
        ]
    } else {
        [
            path_testdata.join("12508460.json"),
            path_testdata.join("12508461.json"),
            path_testdata.join("12508462.json"),
            path_testdata.join("12508463.json"),
        ]
    };
    let read_block_witness = |path| Ok(read_json::<_, BlockWitness>(path)?);
    let chunk_task = ChunkProvingTask {
        block_witnesses: paths_block_witnesses
            .iter()
            .map(read_block_witness)
            .collect::<eyre::Result<Vec<BlockWitness>>>()?,
        prev_msg_queue_hash: Default::default(),
        fork_name: if cfg!(feature = "euclidv2") {
            String::from("euclidv2")
        } else {
            String::from("euclidv1")
        },
    };

    // read chunk proof.
    let path_chunk_proof = path_testdata
        .join("proofs")
        .join(if cfg!(feature = "euclidv2") {
            "chunk-1-4.json"
        } else {
            "chunk-12508460-12508463.json"
        });
    let chunk_proof = read_json_deep::<_, ChunkProof>(&path_chunk_proof)?;

    let task = build_batch_task(&[chunk_task], &[chunk_proof], Default::default());

    let chunk_infos = task
        .chunk_proofs
        .iter()
        .map(|proof| proof.metadata.chunk_info.clone())
        .collect::<Vec<_>>();

    let enveloped = Envelope::from(task.blob_bytes.as_slice());

    #[cfg(feature = "euclidv2")]
    let header = task.batch_header.must_v7_header();
    #[cfg(not(feature = "euclidv2"))]
    let header = task.batch_header.must_v6_header();
    Payload::from(&enveloped).validate(header, &chunk_infos);

    // depressed task output for pre-v2
    #[cfg(feature = "euclidv2")]
    write_json(path_testdata.join("batch-task-test-out.json"), &task).unwrap();
    #[cfg(not(feature = "euclidv2"))]
    write_json(path_testdata.join("batch-task-legacy-test-out.json"), &task).unwrap();
    Ok(())
}

#[cfg(feature = "euclidv2")]
#[test]
fn test_batch_task_payload() -> eyre::Result<()> {
    use scroll_zkvm_circuit_input_types::batch::{EnvelopeV7, PayloadV7};
    use scroll_zkvm_prover::utils::read_json_deep;

    // ./testdata/
    let path_testdata = std::path::Path::new("testdata");

    let task =
        read_json_deep::<_, BatchProvingTask>(path_testdata.join("batch-task-test-out.json"))
            .unwrap();

    println!("blob {:?}", &task.blob_bytes[..32]);
    let enveloped = EnvelopeV7::from(task.blob_bytes.as_slice());

    let chunk_infos = task
        .chunk_proofs
        .iter()
        .map(|proof| proof.metadata.chunk_info.clone())
        .collect::<Vec<_>>();

    PayloadV7::from(&enveloped).validate(task.batch_header.must_v7_header(), &chunk_infos);

    Ok(())
}
