use std::borrow::Cow;

static RELEASE_VERSION: Option<&str> = option_env!("RELEASE_VERSION");
const DEFAULT_VERSION: &str = "v0.0.0-unknown-000000-000000";

pub fn get_version() -> String {
    RELEASE_VERSION.unwrap_or(DEFAULT_VERSION).to_string()
}

pub fn get_version_cow() -> Cow<'static, str> {
    let v = RELEASE_VERSION.unwrap_or(DEFAULT_VERSION);
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

        assert_eq!(get_version(), DEFAULT_VERSION);
        assert_eq!(&version, DEFAULT_VERSION);
        Ok(())
    }
}
