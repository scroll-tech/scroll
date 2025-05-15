use std::path::PathBuf;

/// Errors encountered by the prover.
#[derive(thiserror::Error, Debug)]
pub enum Error {
    /// Error occurred while doing i/o operations.
    #[error(transparent)]
    Io(#[from] std::io::Error),
    /// Error encountered while reading from or writing to files.
    #[error("error during read/write! path={path}, e={source}")]
    IoReadWrite {
        /// The path we tried to read from or write to.
        path: PathBuf,
        /// The source error.
        source: std::io::Error,
    },
    /// Error occurred while doing serde operations.
    #[error(transparent)]
    Serde(#[from] serde_json::Error),
    /// Error encountered during JSON serde.
    #[error("error during read/write json! path={path}, e={source}")]
    JsonReadWrite {
        /// The path of the file we tried to serialize/deserialize.
        path: PathBuf,
        /// The source error.
        source: serde_json::Error,
    },
    /// Covers various errors encountered during the setup phase.
    #[error("failed to read or deserialize {path}: {src}")]
    Setup { path: PathBuf, src: String },
    /// An error encountered while generating commitments to the app exe.
    #[error("failed to commit app exe: {0}")]
    Commit(String),
    /// An error encountered while performing the STARK aggregation keygen process.
    #[error("failed to generate STARK aggregation proving key: {0}")]
    Keygen(String),
    /// An error encountered during proof generation.
    #[error("failed to generate proof: {0}")]
    GenProof(String),
    /// An error encountered during proof verification.
    #[error("failed to verify proof: {0}")]
    VerifyProof(String),
    /// A custom error not covered by above variants.
    #[error("custom error: {0}")]
    Custom(String),
}
