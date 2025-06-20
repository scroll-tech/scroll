pub mod base64 {
    use base64::prelude::*;
    use serde::{Deserialize, Deserializer, Serialize, Serializer};

    pub fn serialize<S: Serializer>(v: &Vec<u8>, s: S) -> Result<S::Ok, S::Error> {
        let base64 = BASE64_STANDARD.encode(v);
        String::serialize(&base64, s)
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(d: D) -> Result<Vec<u8>, D::Error> {
        let base64 = String::deserialize(d)?;
        BASE64_STANDARD
            .decode(base64.as_bytes())
            .map_err(serde::de::Error::custom)
    }
}

pub mod point_eval {
    use c_kzg;
    use sbv_primitives::{types::eips::eip4844::BLS_MODULUS, B256 as H256, U256};
    use scroll_zkvm_types::util::sha256_rv32;

    /// Given the blob-envelope, translate it to a fixed size EIP-4844 blob.
    ///
    /// For every 32-bytes chunk in the blob, the most-significant byte is set to 0 while the other
    /// 31 bytes are copied from the provided blob-envelope.
    pub fn to_blob(envelope_bytes: &[u8]) -> c_kzg::Blob {
        let mut blob_bytes = [0u8; c_kzg::BYTES_PER_BLOB];

        assert!(
            envelope_bytes.len()
                <= c_kzg::FIELD_ELEMENTS_PER_BLOB * (c_kzg::BYTES_PER_FIELD_ELEMENT - 1),
            "too many bytes in blob envelope",
        );

        for (i, &byte) in envelope_bytes.iter().enumerate() {
            blob_bytes[(i / 31) * 32 + 1 + (i % 31)] = byte;
        }

        c_kzg::Blob::new(blob_bytes)
    }

    /// Get the KZG commitment from an EIP-4844 blob.
    pub fn blob_to_kzg_commitment(blob: &c_kzg::Blob) -> c_kzg::KzgCommitment {
        c_kzg::KzgCommitment::blob_to_kzg_commitment(blob, c_kzg::ethereum_kzg_settings())
            .expect("blob to kzg commitment should succeed")
    }

    /// The version for KZG as per EIP-4844.
    const VERSIONED_HASH_VERSION_KZG: u8 = 1;

    /// Get the EIP-4844 versioned hash from the KZG commitment.
    pub fn get_versioned_hash(commitment: &c_kzg::KzgCommitment) -> H256 {
        let mut hash: [u8; 32] = sha256_rv32(commitment.to_bytes().as_slice()).into();
        hash[0] = VERSIONED_HASH_VERSION_KZG;
        H256::new(hash)
    }

    /// Get x for kzg proof from challenge hash
    pub fn get_x_from_challenge(challenge: H256) -> U256 {
        U256::from_be_bytes(challenge.0) % BLS_MODULUS
    }

    /// Generate KZG proof and evaluation given the blob (polynomial) and a random challenge.
    pub fn get_kzg_proof(blob: &c_kzg::Blob, challenge: H256) -> (c_kzg::KzgProof, U256) {
        let challenge = get_x_from_challenge(challenge);

        let (proof, y) = c_kzg::KzgProof::compute_kzg_proof(
            blob,
            &c_kzg::Bytes32::new(challenge.to_be_bytes()),
            c_kzg::ethereum_kzg_settings(),
        )
        .expect("kzg proof should succeed");

        (proof, U256::from_be_slice(y.as_slice()))
    }
}
