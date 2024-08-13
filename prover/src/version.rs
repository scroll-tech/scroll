use std::{borrow::Cow, cell::OnceCell};

static DEFAULT_COMMIT: &str = "unknown";
static mut VERSION: OnceCell<String> = OnceCell::new();

pub const DEFAULT_TAG: &str = "v0.0.0";
pub const DEFAULT_ZK_VERSION: &str = "000000-000000";

static TAG: Option<&str> = option_env!("GO_TAG");

fn init_version() -> String {
    let commit = option_env!("GIT_REV").unwrap_or(DEFAULT_COMMIT);
    let tag = TAG.unwrap_or(DEFAULT_TAG);
    let zk_version = option_env!("ZK_VERSION").unwrap_or(DEFAULT_ZK_VERSION);
    format!("{tag}-{commit}-{zk_version}")
}

pub fn get_version() -> String {
    unsafe { VERSION.get_or_init(init_version).clone() }
}

pub fn get_version_cow() -> Cow<'static, str> {

    let v = TAG.unwrap_or(DEFAULT_TAG);
    std::borrow::Cow::Borrowed(v)
}

// =================================== tests module ========================================

#[cfg(test)]
mod tests {
    use super::*;
    use anyhow::{Ok, Result};

    #[ctor::ctor]
    fn init() {
        crate::utils::log_init(None, false);
        log::info!("logger initialized");
    }

    #[test]
    fn test_get_version_cow() -> Result<()> {
        let version = get_version_cow();

        assert_eq!(get_version(), "v0.0.0-unknown-000000-000000");
        assert_eq!(&version, "v0.0.0");
        Ok(())
    }
}