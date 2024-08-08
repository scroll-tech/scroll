use env_logger::Env;
use std::{fs::OpenOptions, sync::Once};

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
