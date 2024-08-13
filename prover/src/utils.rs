use env_logger::Env;
use std::{borrow::Cow, fs::OpenOptions, sync::Once};

use crate::types::{ProverType, TaskType};

static LOG_INIT: Once = Once::new();

/// Initialize log
pub fn log_init(log_file: Option<String>, sentry_enabled: bool) {
    LOG_INIT.call_once(|| {
        let mut builder = env_logger::Builder::from_env(Env::default().default_filter_or("info"));
        if let Some(file_path) = log_file {
            let target = Box::new(
                OpenOptions::new()
                    .write(true)
                    .create(true)
                    .truncate(false)
                    .open(file_path)
                    .expect("Can't create log file"),
            );
            builder.target(env_logger::Target::Pipe(target));
        }
        let logger = builder.build();
        let max_level = logger.filter();

        let boxed_logger: Box<dyn log::Log> = if sentry_enabled {
            Box::new(sentry_log::SentryLogger::with_dest(logger))
        } else {
            Box::new(logger)
        };

        log::set_boxed_logger(boxed_logger)
            .map(|()| log::set_max_level(max_level))
            .unwrap();
    });
}

pub fn get_task_types(prover_type: ProverType) -> Vec<TaskType> {
    match prover_type {
        ProverType::Chunk => vec![TaskType::Chunk],
        ProverType::Batch => vec![TaskType::Batch, TaskType::Bundle],
    }
}

static ENV_UNKNOWN: &str = "unknown";
static ENV_DEVNET: &str = "devnet";
static ENV_SEPOLIA: &str = "sepolia";
static ENV_MAINNET: &str = "mainnet";

pub fn get_environment() -> Cow<'static, str> {
    let env: &'static str = match std::env::var("CHAIN_ID") {
        Ok(chain_id) => match chain_id.as_str() {
            "534352" => ENV_MAINNET,
            "534351" => ENV_SEPOLIA,
            _ => ENV_DEVNET,
        },
        Err(_) => ENV_UNKNOWN,
    };
    std::borrow::Cow::Borrowed(env)
}
