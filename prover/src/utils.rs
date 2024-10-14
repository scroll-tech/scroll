use env_logger::Env;
use std::{borrow::Cow, fs::OpenOptions};

use crate::types::{ProverType, TaskType};

pub fn build_logger(log_file: Option<String>) -> env_logger::Logger {
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
    builder.build()
}

pub fn log_init(log_file: Option<String>) {
    let logger = build_logger(log_file);
    let max_level = logger.filter();
    let boxed_logger = Box::new(logger);
    alterable_logger::configure(max_level, boxed_logger);
}

pub fn set_logger_with_sentry(log_file: Option<String>) {
    let logger = build_logger(log_file);
    let max_level = logger.filter();
    let boxed_logger = Box::new(sentry_log::SentryLogger::with_dest(logger));
    alterable_logger::configure(max_level, boxed_logger);
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
