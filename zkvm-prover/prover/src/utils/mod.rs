use std::path::Path;

use git_version::git_version;
use serde::{
    Serialize,
    de::{Deserialize, DeserializeOwned},
};

use crate::Error;

pub mod vm;

pub const GIT_VERSION: &str = git_version!(args = ["--abbrev=7", "--always"]);

/// Shortened git commit ref from [`scroll_zkvm_prover`].
pub fn short_git_version() -> String {
    let commit_version = GIT_VERSION.split('-').next_back().unwrap();

    // Check if use commit object as fallback.
    if commit_version.len() < 8 {
        commit_version.to_string()
    } else {
        commit_version[1..8].to_string()
    }
}

/// Wrapper to read JSON file.
pub fn read_json<P: AsRef<Path>, T: DeserializeOwned>(path: P) -> Result<T, Error> {
    let path = path.as_ref();
    let bytes = read(path)?;
    serde_json::from_slice(&bytes).map_err(|source| Error::JsonReadWrite {
        source,
        path: path.to_path_buf(),
    })
}

/// Wrapper to read JSON that might be deeply nested.
pub fn read_json_deep<P: AsRef<Path>, T: DeserializeOwned>(path: P) -> Result<T, Error> {
    let fd = std::fs::File::open(path)?;
    let mut deserializer = serde_json::Deserializer::from_reader(fd);
    deserializer.disable_recursion_limit();
    let deserializer = serde_stacker::Deserializer::new(&mut deserializer);
    Ok(Deserialize::deserialize(deserializer)?)
}

/// Read bytes from a file.
pub fn read<P: AsRef<Path>>(path: P) -> Result<Vec<u8>, Error> {
    let path = path.as_ref();
    std::fs::read(path).map_err(|source| Error::IoReadWrite {
        source,
        path: path.into(),
    })
}

/// Serialize the provided type to JSON format and write to the given path.
pub fn write_json<P: AsRef<Path>, T: Serialize>(path: P, value: &T) -> Result<(), Error> {
    let mut writer = std::fs::File::create(path)?;
    Ok(serde_json::to_writer(&mut writer, value)?)
}

/// Serialize the provided type with bincode and write to the given path.
pub fn write_bin<P: AsRef<Path>, T: Serialize>(path: P, value: &T) -> Result<(), Error> {
    let data = bincode_v1::serialize(value).map_err(|e| Error::Custom(e.to_string()))?;
    write(path, &data)
}

/// Wrapper functionality to write bytes to a file.
pub fn write<P: AsRef<Path>>(path: P, data: &[u8]) -> Result<(), Error> {
    let path = path.as_ref();
    Ok(std::fs::write(path, data)?)
}

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

pub mod as_base64 {
    use base64::prelude::*;
    use serde::{Deserialize, Deserializer, Serialize, Serializer, de::DeserializeOwned};

    pub fn serialize<S: Serializer, T: Serialize>(v: &T, s: S) -> Result<S::Ok, S::Error> {
        let v_bytes = bincode_v1::serialize(v).map_err(serde::ser::Error::custom)?;
        let v_base64 = BASE64_STANDARD.encode(&v_bytes);
        String::serialize(&v_base64, s)
    }

    pub fn deserialize<'de, D: Deserializer<'de>, T: DeserializeOwned>(
        d: D,
    ) -> Result<T, D::Error> {
        let v_base64 = String::deserialize(d)?;
        let v_bytes = BASE64_STANDARD
            .decode(v_base64.as_bytes())
            .map_err(serde::de::Error::custom)?;
        bincode_v1::deserialize(&v_bytes).map_err(serde::de::Error::custom)
    }
}

pub mod point_eval {
    use c_kzg;
    use sbv_primitives::{B256 as H256, U256, types::eips::eip4844::BLS_MODULUS};

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

    /// Get the EIP-4844 versioned hash from the KZG commitment.
    pub fn get_versioned_hash(commitment: &c_kzg::KzgCommitment) -> H256 {
        H256::new(
            revm::precompile::kzg_point_evaluation::kzg_to_versioned_hash(commitment.as_slice()),
        )
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

use snark_verifier_sdk::snark_verifier::halo2_base::halo2_proofs::halo2curves::bn256::Fr;

/// use the openvm's implement for compress 8xbabybear into bn254
pub fn compress_commitment(commitment: &[u32; 8]) -> Fr {
    use openvm_stark_sdk::{openvm_stark_backend::p3_field::PrimeField32, p3_baby_bear::BabyBear};
    let order = Fr::from(BabyBear::ORDER_U32 as u64);
    let mut base = Fr::one();
    let mut ret = Fr::zero();

    for v in commitment {
        ret += Fr::from(*v as u64) * base;
        base *= order;
    }

    ret
}
