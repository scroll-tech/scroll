use std::path::Path;

use anyhow::Result;
use ethers_core::{
    k256::{
        ecdsa::{signature::hazmat::PrehashSigner, RecoveryId, Signature, SigningKey},
        elliptic_curve::{sec1::ToEncodedPoint, FieldBytes},
        PublicKey, Secp256k1, SecretKey,
    },
    types::Signature as EthSignature,
};

use ethers_core::types::{H256, U256};
use hex::ToHex;
use tiny_keccak::{Hasher, Keccak};

pub struct KeySigner {
    public_key: PublicKey,
    signer: SigningKey,
}

impl KeySigner {
    pub fn new(key_path: &str, passwd: &str) -> Result<Self> {
        let p = Path::new(key_path);

        let secret = if !p.exists() {
            log::info!("[key_signer] key_path not exists, create one");
            let dir = p.parent().unwrap();
            let name = p.file_name().and_then(|s| s.to_str());
            let mut rng = rand::thread_rng();
            let (secret, _) = eth_keystore::new(dir, &mut rng, passwd, name)?;
            secret
        } else {
            log::info!("[key_signer] key_path already exists, load it");
            eth_keystore::decrypt_key(key_path, passwd).map_err(|e| anyhow::anyhow!(e))?
        };

        let secret_key = SecretKey::from_bytes(secret.as_slice().into())?;

        let signer = SigningKey::from(secret_key.clone());

        Ok(Self {
            public_key: secret_key.public_key(),
            signer,
        })
    }

    pub fn get_public_key(&self) -> String {
        let v: Vec<u8> = Vec::from(self.public_key.to_encoded_point(true).as_bytes());
        buffer_to_hex(&v, false)
    }

    /// Signs the provided hash.
    pub fn sign_hash(&self, hash: H256) -> Result<EthSignature> {
        let signer = &self.signer as &dyn PrehashSigner<(Signature, RecoveryId)>;
        let (recoverable_sig, recovery_id) = signer.sign_prehash(hash.as_ref())?;

        let v = u8::from(recovery_id) as u64;

        let r_bytes: FieldBytes<Secp256k1> = recoverable_sig.r().into();
        let s_bytes: FieldBytes<Secp256k1> = recoverable_sig.s().into();
        let r = U256::from_big_endian(r_bytes.as_slice());
        let s = U256::from_big_endian(s_bytes.as_slice());

        Ok(EthSignature { r, s, v })
    }

    pub fn sign_buffer<T>(&self, buffer: &T) -> Result<String>
    where
        T: AsRef<[u8]>,
    {
        let pre_hash = keccak256(buffer);

        let hash = H256::from(pre_hash);
        let sig = self.sign_hash(hash)?;

        Ok(buffer_to_hex(&sig.to_vec(), true))
    }
}

fn buffer_to_hex<T>(buffer: &T, has_prefix: bool) -> String
where
    T: AsRef<[u8]>,
{
    if has_prefix {
        format!("0x{}", buffer.encode_hex::<String>())
    } else {
        buffer.encode_hex::<String>()
    }
}

/// Compute the Keccak-256 hash of input bytes.
///
/// Note that strings are interpreted as UTF-8 bytes,
pub fn keccak256<T: AsRef<[u8]>>(bytes: T) -> [u8; 32] {
    let mut output = [0u8; 32];

    let mut hasher = Keccak::v256();
    hasher.update(bytes.as_ref());
    hasher.finalize(&mut output);

    output
}
