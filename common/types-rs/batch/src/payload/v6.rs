use alloy_primitives::B256;
use itertools::Itertools;

use crate::BatchHeaderV6;
use types_base::{public_inputs::chunk::ChunkInfo, utils::keccak256};

/// The default max chunks for v6 payload
pub const N_MAX_CHUNKS: usize = 45;

/// The number of bytes to encode number of chunks in a batch.
const N_BYTES_NUM_CHUNKS: usize = 2;

/// The number of rows to encode chunk size (u32).
const N_BYTES_CHUNK_SIZE: usize = 4;

impl From<&[u8]> for EnvelopeV6 {
    fn from(blob_bytes: &[u8]) -> Self {
        let is_encoded = blob_bytes[0] & 1 == 1;
        Self {
            is_encoded,
            envelope_bytes: if blob_bytes[0] & 1 == 1 {
                vm_zstd::process(&blob_bytes[1..]).unwrap().decoded_data
            } else {
                Vec::from(&blob_bytes[1..])
            },
        }
    }
}

#[derive(Debug, Clone)]
pub struct EnvelopeV6 {
    /// The original envelope bytes supplied.
    ///
    /// Caching just for re-use later in challenge digest computation.
    pub envelope_bytes: Vec<u8>,
    /// If the enveloped bytes is encoded (compressed) in envelop
    pub is_encoded: bool,
}

impl EnvelopeV6 {
    /// Parse payload bytes and obtain challenge digest
    pub fn challenge_digest(&self, versioned_hash: B256) -> B256 {
        let payload = Payload::from(self);
        payload.get_challenge_digest(versioned_hash)
    }
}

impl From<&EnvelopeV6> for Payload {
    fn from(envelope: &EnvelopeV6) -> Self {
        Self::from_payload(&envelope.envelope_bytes)
    }
}

/// Payload that describes a batch.
#[derive(Clone, Debug, Default)]
pub struct Payload {
    /// Metadata that encodes the sizes of every chunk in the batch.
    pub metadata_digest: B256,
    /// The Keccak digests of transaction bytes for every chunk in the batch.
    ///
    /// The `chunk_data_digest` is a part of the chunk-circuit's public input and hence used to
    /// verify that the transaction bytes included in the chunk-circuit indeed match the
    /// transaction bytes made available in the batch.
    pub chunk_data_digests: Vec<B256>,
}

pub type PayloadV6 = Payload;

impl Payload {
    /// For raw payload data (read from decompressed enveloped data), which is raw batch bytes
    /// with metadata, this function segments the byte stream into chunk segments.
    ///
    /// This method is used INSIDE OF zkvm since we can not generate (compress) batch data within
    /// the vm program
    ///
    /// The structure of batch bytes is as follows:
    ///
    /// | Byte Index                                                   | Size                          | Hint                                |
    /// |--------------------------------------------------------------|-------------------------------|-------------------------------------|
    /// | 0                                                            | N_BYTES_NUM_CHUNKS            | Number of chunks                    |
    /// | N_BYTES_NUM_CHUNKS                                           | N_BYTES_CHUNK_SIZE            | Size of chunks[0]                   |
    /// | N_BYTES_NUM_CHUNKS + N_BYTES_CHUNK_SIZE                      | N_BYTES_CHUNK_SIZE            | Size of chunks[1]                   |
    /// | N_BYTES_NUM_CHUNKS + (i * N_BYTES_CHUNK_SIZE)                | N_BYTES_CHUNK_SIZE            | Size of chunks[i]                   |
    /// | N_BYTES_NUM_CHUNKS + ((N_MAX_CHUNKS-1) * N_BYTES_CHUNK_SIZE) | N_BYTES_CHUNK_SIZE            | Size of chunks[N_MAX_CHUNKS-1]      |
    /// | N_BYTES_NUM_CHUNKS + (N_MAX_CHUNKS * N_BYTES_CHUNK_SIZE)     | Size of chunks[0]             | L2 tx bytes of chunks[0]            |
    /// | "" + Size_of_chunks[0]                                       | Size of chunks[1]             | L2 tx bytes of chunks[1]            |
    /// | "" + Size_of_chunks[i-1]                                     | Size of chunks[i]             | L2 tx bytes of chunks[i]            |
    /// | "" + Size_of_chunks[Num_chunks-1]                            | Size of chunks[Num_chunks-1]  | L2 tx bytes of chunks[Num_chunks-1] |
    pub fn from_payload(batch_bytes_with_metadata: &[u8]) -> Self {
        // Get the metadata bytes and metadata digest.
        let n_bytes_metadata = Self::n_bytes_metadata();
        let metadata_bytes = &batch_bytes_with_metadata[..n_bytes_metadata];
        let metadata_digest = keccak256(metadata_bytes);

        // The remaining bytes represent the chunk data (L2 tx bytes) segmented as chunks.
        let batch_bytes = &batch_bytes_with_metadata[n_bytes_metadata..];

        // The number of chunks in the batch.
        let valid_chunks = metadata_bytes[..N_BYTES_NUM_CHUNKS]
            .iter()
            .fold(0usize, |acc, &d| acc * 256usize + d as usize);

        // The size of each chunk in the batch.
        let chunk_sizes = metadata_bytes[N_BYTES_NUM_CHUNKS..]
            .iter()
            .chunks(N_BYTES_CHUNK_SIZE)
            .into_iter()
            .map(|bytes| bytes.fold(0usize, |acc, &d| acc * 256usize + d as usize))
            .collect::<Vec<usize>>();

        // For every unused chunk, the chunk size should be set to 0.
        for &unused_chunk_size in chunk_sizes.iter().skip(valid_chunks) {
            assert_eq!(unused_chunk_size, 0, "unused chunk has size 0");
        }

        // Segment the batch bytes based on the chunk sizes.
        let (segmented_batch_data, remaining_bytes) =
            chunk_sizes.into_iter().take(valid_chunks).fold(
                (Vec::new(), batch_bytes),
                |(mut datas, rest_bytes), size| {
                    datas.push(Vec::from(&rest_bytes[..size]));
                    (datas, &rest_bytes[size..])
                },
            );

        // After segmenting the batch data into chunks, no bytes should be left.
        assert!(
            remaining_bytes.is_empty(),
            "chunk segmentation len must add up to the correct value"
        );

        // Compute the chunk data digests based on the segmented data.
        let chunk_data_digests = segmented_batch_data
            .iter()
            .map(|bytes| B256::from(keccak256(bytes)))
            .collect();

        Self {
            metadata_digest,
            chunk_data_digests,
        }
    }

    /// Compute the challenge digest from blob bytes. which is the combination of
    /// digest for bytes in each chunk
    pub fn get_challenge_digest(&self, versioned_hash: B256) -> B256 {
        keccak256(self.get_challenge_digest_preimage(versioned_hash))
    }

    /// The number of bytes in payload Data to represent the "payload metadata" section: a u16 to
    /// represent the size of chunks and max_chunks * u32 to represent chunk sizes
    const fn n_bytes_metadata() -> usize {
        N_BYTES_NUM_CHUNKS + (N_MAX_CHUNKS * N_BYTES_CHUNK_SIZE)
    }

    /// Validate the payload contents.
    pub fn validate<'a>(
        &self,
        header: &BatchHeaderV6,
        chunk_infos: &'a [ChunkInfo],
    ) -> (&'a ChunkInfo, &'a ChunkInfo) {
        // There should be at least 1 chunk info.
        assert!(!chunk_infos.is_empty(), "at least 1 chunk info");

        // Get the first and last chunks' info, to construct the batch info.
        let (first_chunk, last_chunk) = (
            chunk_infos.first().expect("at least one chunk in batch"),
            chunk_infos.last().expect("at least one chunk in batch"),
        );

        for (&chunk_data_digest, chunk_info) in self.chunk_data_digests.iter().zip_eq(chunk_infos) {
            assert_eq!(chunk_data_digest, chunk_info.tx_data_digest)
        }

        // Validate the l1-msg identifier data_hash for the batch.
        let batch_data_hash_preimage = chunk_infos
            .iter()
            .flat_map(|chunk_info| chunk_info.data_hash.0)
            .collect::<Vec<_>>();
        let batch_data_hash = keccak256(batch_data_hash_preimage);
        assert_eq!(batch_data_hash, header.data_hash);

        (first_chunk, last_chunk)
    }

    /// Get the preimage for the challenge digest.
    pub(crate) fn get_challenge_digest_preimage(&self, versioned_hash: B256) -> Vec<u8> {
        // preimage =
        //     metadata_digest ||
        //     chunk[0].chunk_data_digest || ...
        //     chunk[N_SNARKS-1].chunk_data_digest ||
        //     blob_versioned_hash
        //
        // where chunk_data_digest for a padded chunk is set equal to the "last valid chunk"'s
        // chunk_data_digest.
        let mut preimage = self.metadata_digest.to_vec();
        let last_digest = self
            .chunk_data_digests
            .last()
            .expect("at least we have one");
        for chunk_digest in self
            .chunk_data_digests
            .iter()
            .chain(std::iter::repeat(last_digest))
            .take(N_MAX_CHUNKS)
        {
            preimage.extend_from_slice(chunk_digest.as_slice());
        }
        preimage.extend_from_slice(versioned_hash.as_slice());
        preimage
    }
}
