use anyhow::{Ok, Result};

use crate::types::TaskWrapper;
use sled::{Config, Db};

pub struct TaskCache {
    db: Db,
}

impl TaskCache {
    pub fn new(db_path: &String) -> Result<Self> {
        let config = Config::new().path(db_path);
        let db = config.open()?;
        Ok(Self{db})
    }

    pub fn put_task(&self, task_wrapper: &TaskWrapper) -> Result<()> {
        let k = task_wrapper.task.id.clone().into_bytes();
        let v = serde_json::to_vec(task_wrapper)?;
        self.db.insert(k, v)?;
        Ok(())
    }

    pub fn get_last_task(&self) -> Result<Option<TaskWrapper>> {
        let last = self.db.last()?;
        if let Some((k, v)) = last {
            let kk = std::str::from_utf8(k.as_ref())?;
            log::info!("get last task, task_id: {kk}");
            let task_wrapper: TaskWrapper = serde_json::from_slice(v.as_ref())?;
            return Ok(Some(task_wrapper));
        }
        Ok(None)
    }

    pub fn delete_task(&self, task_id: String) -> Result<()> {
        let k = task_id.into_bytes();
        self.db.remove(k)?;
        Ok(())
    }
}
