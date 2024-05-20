use env_logger::Env;
use std::sync::Once;


static LOG_INIT: Once = Once::new();

/// Initialize log
pub fn log_init() {
    LOG_INIT.call_once(|| {
        env_logger::Builder::from_env(Env::default().default_filter_or("info")).init();
    });
}