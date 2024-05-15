

pub const TAG: &str = "v4.4.3";
pub const COMMIT: &str = "test";
pub const ZK_VERSION: &str = "000000-000000";
pub const VERSION: String = format!("{TAG}-{COMMIT}-{ZK_VERSION}");

pub fn get_version() -> String {
    VERSION
}