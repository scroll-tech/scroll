use anyhow::{anyhow, bail, Result};
use std::any::Any;
use std::collections::HashMap;
use std::sync::mpsc::{channel, sync_channel, Sender, SyncSender};
use std::sync::Arc;
use std::time::Duration;
use std::time::Instant;
use tokio::task::JoinHandle;

type Key = String;
type RawValue = Arc<dyn Any + Send + Sync>;
type Ttl = u64;

const CHANNEL_BOUND: usize = 200;

pub fn run() -> Result<Cache> {
    let mut cache = Cache::new();
    cache.run();

    Ok(cache)
}

#[derive(Debug)]
pub struct Cache {
    handle: Option<JoinHandle<()>>,
    sender: Option<SyncSender<Request>>,
}

// TODO: Add a `Request::AutoClear` to delete expired values automatically.
impl Cache {
    fn new() -> Self {
        Self {
            handle: None,
            sender: None,
        }
    }

    fn run(&mut self) {
        let (sender, receiver) = sync_channel(CHANNEL_BOUND);
        let handle = tokio::spawn(async move {
            let mut kvs: HashMap<String, Value> = HashMap::new();
            loop {
                match receiver.recv() {
                    Ok(req) => {
                        match req {
                            Request::Get(sender, key) => {
                                let raw = match kvs.get(&key) {
                                    Some(val) => {
                                        if val.expired_at > Instant::now() {
                                            Some(val.raw.clone())
                                        } else {
                                            kvs.remove(&key);
                                            None
                                        }
                                    }
                                    None => None,
                                };
                                if let Err(error) = sender.send(Response::Get(raw)) {
                                    log::error!(
                                        "Cache - Failed to send Response of Get({key}): {error}"
                                    );
                                    break;
                                }
                            }
                            Request::Set(sender, key, raw, ttl) => {
                                let expired_at = if let Some(expired_at) =
                                    Instant::now().checked_add(Duration::new(ttl, 0))
                                {
                                    expired_at
                                } else {
                                    log::error!(
                                        "Cache - Failed to calculate expired time by TTL: {ttl}"
                                    );
                                    break;
                                };
                                if let Err(error) = sender.send(Response::Set) {
                                    log::error!("Cache - Failed to send Response of Set({key}, ..): {error}");
                                    break;
                                }
                                kvs.insert(key, Value { raw, expired_at });
                            }
                        }
                    }
                    Err(error) => {
                        log::error!("Cache - Failed to receive Request: {error}");
                        break;
                    }
                }
            }
        });

        self.handle = Some(handle);
        self.sender = Some(sender);
    }

    pub async fn stop(&mut self) -> Result<()> {
        if let Some(sender) = self.sender.take() {
            drop(sender);
        }
        if let Some(handle) = self.handle.take() {
            tokio::try_join!(handle)?;
        }

        Ok(())
    }

    pub async fn get(&self, key: &str) -> Result<Option<Arc<dyn Any + Send + Sync>>> {
        if let Some(sender) = self.sender.as_ref() {
            let (req_sender, res_receiver) = channel();
            sender
                .send(Request::Get(req_sender, key.to_string()))
                .map_err(|error| anyhow!("Cache - Failed to send a Get request: {error}"))?;
            match res_receiver.recv()? {
                Response::Get(raw) => Ok(raw),
                _ => unreachable!(),
            }
        } else {
            bail!("Cache is not running");
        }
    }

    pub async fn set(&self, key: &str, val: Arc<dyn Any + Send + Sync>, ttl: Ttl) -> Result<()> {
        if let Some(sender) = self.sender.as_ref() {
            let (req_sender, res_receiver) = channel();
            sender
                .send(Request::Set(req_sender, key.to_string(), val, ttl))
                .map_err(|error| anyhow!("Cache - Failed to send a Set request: {error}"))?;
            match res_receiver.recv()? {
                Response::Set => Ok(()),
                _ => unreachable!(),
            }
        } else {
            bail!("Cache is not running");
        }
    }
}

#[derive(Debug)]
enum Request {
    Get(Sender<Response>, Key),
    Set(Sender<Response>, Key, RawValue, Ttl),
}

#[derive(Debug)]
enum Response {
    Get(Option<RawValue>),
    Set,
}

#[derive(Clone, Debug)]
struct Value {
    expired_at: Instant,
    raw: RawValue,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test(flavor = "multi_thread", worker_threads = 1)]
    async fn cache_tests() {
        let mut cache = run().unwrap();

        // Return None for non-existing key.
        assert!(cache.get("non-existing-key").await.unwrap().is_none());

        // Set a key-value to cache and get the cached value.
        cache
            .set("key1", Arc::new("value1".to_string()), 10)
            .await
            .unwrap();
        assert_eq!(
            cache
                .get("key1")
                .await
                .unwrap()
                .unwrap()
                .downcast_ref::<String>()
                .unwrap(),
            "value1"
        );

        // Value should be expired directly for zero TTL.
        cache
            .set("key2", Arc::new("value2".to_string()), 0)
            .await
            .unwrap();
        assert!(cache.get("key2").await.unwrap().is_none());

        cache.stop().await.unwrap();
    }
}
