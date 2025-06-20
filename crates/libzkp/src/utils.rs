use std::{
    panic::{catch_unwind, AssertUnwindSafe},
    path::Path,
};

use git_version::git_version;
use serde::{
    de::{Deserialize, DeserializeOwned},
    Serialize,
};

use eyre::Result;

const GIT_VERSION: &str = git_version!(args = ["--abbrev=7", "--always"]);

/// Shortened git commit ref from [`scroll_zkvm_prover`].
pub(crate) fn short_git_version() -> String {
    let commit_version = GIT_VERSION.split('-').next_back().unwrap();

    // Check if use commit object as fallback.
    if commit_version.len() < 8 {
        commit_version.to_string()
    } else {
        commit_version[1..8].to_string()
    }
}

/// Wrapper to read JSON that might be deeply nested.
pub(crate) fn read_json_deep<P: AsRef<Path>, T: DeserializeOwned>(path: P) -> Result<T> {
    let fd = std::fs::File::open(path)?;
    let mut deserializer = serde_json::Deserializer::from_reader(fd);
    deserializer.disable_recursion_limit();
    let deserializer = serde_stacker::Deserializer::new(&mut deserializer);
    Ok(Deserialize::deserialize(deserializer)?)
}

/// Serialize the provided type to JSON format and write to the given path.
pub(crate) fn write_json<P: AsRef<Path>, T: Serialize>(path: P, value: &T) -> Result<()> {
    let mut writer = std::fs::File::create(path)?;
    Ok(serde_json::to_writer(&mut writer, value)?)
}

pub(crate) fn panic_catch<F: FnOnce() -> R, R>(f: F) -> Result<R, String> {
    catch_unwind(AssertUnwindSafe(f)).map_err(|err| {
        if let Some(s) = err.downcast_ref::<String>() {
            s.to_string()
        } else if let Some(s) = err.downcast_ref::<&str>() {
            s.to_string()
        } else {
            format!("unable to get panic info {err:?}")
        }
    })
}
