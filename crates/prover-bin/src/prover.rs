use crate::zk_circuits_handler::{universal::UniversalHandler, CircuitsHandler};
use async_trait::async_trait;
use eyre::Result;
use scroll_proving_sdk::{
    config::Config as SdkConfig,
    prover::{
        proving_service::{
            GetVkRequest, GetVkResponse, ProveRequest, ProveResponse, QueryTaskRequest,
            QueryTaskResponse, TaskStatus,
        },
        types::ProofType,
        ProvingService,
    },
};
use scroll_zkvm_types::ProvingTask;
use serde::{Deserialize, Serialize};
use std::{
    collections::HashMap,
    fs::File,
    path::{Path, PathBuf},
    sync::{Arc, LazyLock},
    time::{SystemTime, UNIX_EPOCH},
};
use tokio::{runtime::Handle, sync::Mutex, task::JoinHandle};

#[derive(Clone, Serialize, Deserialize)]
pub struct AssetsLocationData {
    /// the base url to form a general downloading url for an asset, MUST HAVE A TRAILING SLASH
    pub base_url: url::Url,
    #[serde(default)]
    /// a altered url for specififed vk
    pub asset_detours: HashMap<String, url::Url>,
    /// when asset file existed, do not verify from network, help for debugging stuffs
    #[serde(default)]
    pub debug_mode: bool,
}

impl AssetsLocationData {
    pub fn gen_asset_url(&self, vk_as_path: &str, proof_type: ProofType) -> Result<url::Url> {
        Ok(self.base_url.join(
            match proof_type {
                ProofType::Chunk => format!("chunk/{vk_as_path}/"),
                ProofType::Batch => format!("batch/{vk_as_path}/"),
                ProofType::Bundle => format!("bundle/{vk_as_path}/"),
                t => eyre::bail!("unrecognized proof type: {}", t as u8),
            }
            .as_str(),
        )?)
    }

    pub fn validate(&self) -> Result<()> {
        if !self.base_url.path().ends_with('/') {
            eyre::bail!(
                "base_url must have a trailing slash, got: {}",
                self.base_url
            );
        }
        Ok(())
    }

    /// Pre-flight check: verify the asset base URL is reachable before any proving.
    /// S3 buckets typically return 403 for directory listings, which is expected and OK.
    /// A connection error (DNS failure, timeout) indicates a bad URL.
    pub async fn preflight_check(&self) -> Result<()> {
        let client = reqwest::Client::new();
        match client.head(self.base_url.clone()).send().await {
            Ok(resp) => {
                let status = resp.status();
                if status.is_success() || status.as_u16() == 403 {
                    // 403 on S3 directory listing is normal — bucket listing is disabled
                    Ok(())
                } else {
                    eyre::bail!(
                        "Asset URL returned unexpected status {}: {}",
                        status,
                        self.base_url
                    )
                }
            }
            Err(e) => {
                eyre::bail!(
                    "Asset URL unreachable: {}\n  Caused by: {}\n  Check the base_url in your prover config (common issue: extra 'releases/' path segment).",
                    self.base_url,
                    e
                )
            }
        }
    }

    pub async fn get_asset(
        &self,
        vk: &str,
        url_base: &url::Url,
        base_path: impl AsRef<Path>,
    ) -> Result<PathBuf> {
        let download_files = ["app.vmexe", "openvm.toml"];

        // Step 1: Create a local path for storage
        let storage_path = base_path.as_ref().join(vk);
        std::fs::create_dir_all(&storage_path)?;

        // Step 2 & 3: Download each file if needed
        let client = reqwest::Client::new();

        for filename in download_files.iter() {
            let local_file_path = storage_path.join(filename);
            let download_url = url_base.join(filename)?;

            // Check if file already exists
            if local_file_path.exists() {
                // Get file metadata to check size
                if let Ok(metadata) = std::fs::metadata(&local_file_path) {
                    // Make a HEAD request to get remote file size
                    if self.debug_mode {
                        println!(
                            "File {} already exists, skipping download under debugmode",
                            filename
                        );
                        continue;
                    }

                    if let Ok(head_resp) = client.head(download_url.clone()).send().await {
                        if let Some(content_length) = head_resp.headers().get("content-length") {
                            if let Ok(remote_size) =
                                content_length.to_str().unwrap_or("0").parse::<u64>()
                            {
                                // If sizes match, skip download
                                if metadata.len() == remote_size {
                                    println!("File {} already exists with matching size, skipping download", filename);
                                    continue;
                                }
                            }
                        }
                    }
                }
            }

            println!("Downloading {} from {}", filename, download_url);

            let response = client.get(download_url).send().await?;
            if !response.status().is_success() {
                eyre::bail!(
                    "Failed to download {}: HTTP status {}",
                    filename,
                    response.status()
                );
            }

            // Stream the content directly to file instead of loading into memory
            let mut file = std::fs::File::create(&local_file_path)?;
            let mut stream = response.bytes_stream();

            use futures_util::StreamExt;
            while let Some(chunk) = stream.next().await {
                std::io::Write::write_all(&mut file, &chunk?)?;
            }
        }

        // Step 4: Return the storage path
        Ok(storage_path)
    }
}

#[derive(Clone, Serialize, Deserialize)]
pub struct LocalProverConfig {
    pub sdk_config: SdkConfig,
    pub circuits: HashMap<String, CircuitConfig>,
}

impl LocalProverConfig {
    pub fn from_reader<R>(reader: R) -> Result<Self>
    where
        R: std::io::Read,
    {
        serde_json::from_reader(reader).map_err(|e| eyre::eyre!(e))
    }

    pub fn from_file(file_name: String) -> Result<Self> {
        let file = File::open(file_name)?;
        Self::from_reader(&file)
    }
}

#[derive(Clone, Serialize, Deserialize)]
pub struct CircuitConfig {
    /// The path to save assets for a specified hard fork phase
    pub workspace_path: String,
    #[serde(flatten)]
    /// The location data for dynamic loading
    pub location_data: AssetsLocationData,
    /// cached vk value to save some initial cost, for debugging only
    #[serde(default)]
    pub vks: HashMap<ProofType, String>,
    /// Child circuit VKs used to enable OpenVM deferral for aggregation tasks.
    /// Required for batch (child=chunk) and bundle (child=batch) proving in v0.9.0+.
    #[serde(default)]
    pub child_circuit_vks: HashMap<ProofType, String>,
}

pub struct LocalProver {
    config: LocalProverConfig,
    next_task_id: u64,
    current_task: Option<JoinHandle<Result<String>>>,

    handlers: HashMap<String, Arc<Mutex<UniversalHandler>>>,
}

#[async_trait]
impl ProvingService for LocalProver {
    fn is_local(&self) -> bool {
        true
    }
    async fn get_vks(&self, _: GetVkRequest) -> GetVkResponse {
        // get vk has been deprecated in new prover with dynamic asset loading scheme
        GetVkResponse {
            vks: vec![],
            error: None,
        }
    }
    async fn prove(&mut self, req: ProveRequest) -> ProveResponse {
        match self.do_prove(req).await {
            Ok(resp) => resp,
            Err(e) => ProveResponse {
                status: TaskStatus::Failed,
                error: Some(format!("failed to request proof: {}", e)),
                ..Default::default()
            },
        }
    }

    async fn query_task(&mut self, req: QueryTaskRequest) -> QueryTaskResponse {
        if let Some(handle) = &mut self.current_task {
            if handle.is_finished() {
                return match handle.await {
                    Ok(Ok(proof)) => QueryTaskResponse {
                        task_id: req.task_id,
                        status: TaskStatus::Success,
                        proof: Some(proof),
                        ..Default::default()
                    },
                    Ok(Err(e)) => QueryTaskResponse {
                        task_id: req.task_id,
                        status: TaskStatus::Failed,
                        error: Some(format!("proving task failed: {}", e)),
                        ..Default::default()
                    },
                    Err(e) => {
                        if e.is_panic() {
                            // simply re-throw panic for any panicking in proving process,
                            // cause worker loop and the whole prover exit
                            std::panic::resume_unwind(e.into_panic());
                        }

                        QueryTaskResponse {
                            task_id: req.task_id,
                            status: TaskStatus::Failed,
                            error: Some(format!("proving task failed: {}", e)),
                            ..Default::default()
                        }
                    }
                };
            } else {
                return QueryTaskResponse {
                    task_id: req.task_id,
                    status: TaskStatus::Proving,
                    ..Default::default()
                };
            }
        }
        // If no handle is found
        QueryTaskResponse {
            task_id: req.task_id,
            status: TaskStatus::Failed,
            error: Some("no proving task is running".to_string()),
            ..Default::default()
        }
    }
}

static GLOBAL_ASSET_URLS: LazyLock<HashMap<String, HashMap<String, url::Url>>> =
    LazyLock::new(|| {
        const ASSETS_JSON: &str = include_str!("../assets_url_preset.json");
        serde_json::from_str(ASSETS_JSON).expect("Failed to parse assets_url_preset.json")
    });

impl LocalProver {
    pub fn new(mut config: LocalProverConfig) -> Self {
        for (fork_name, circuit_config) in config.circuits.iter_mut() {
            // validate each base url
            circuit_config.location_data.validate().unwrap();
            let mut template_url_mapping = GLOBAL_ASSET_URLS
                .get(&fork_name.to_lowercase())
                .cloned()
                .unwrap_or_default();

            // apply default settings in template
            for (key, url) in circuit_config.location_data.asset_detours.drain() {
                template_url_mapping.insert(key, url);
            }

            circuit_config.location_data.asset_detours = template_url_mapping;

            // validate each detours url
            for url in circuit_config.location_data.asset_detours.values() {
                assert!(
                    url.path().ends_with('/'),
                    "url {} must be end with /",
                    url.as_str()
                );
            }
        }

        Self {
            config,
            next_task_id: 0,
            current_task: None,
            handlers: HashMap::new(),
        }
    }

    async fn do_prove(&mut self, req: ProveRequest) -> Result<ProveResponse> {
        self.next_task_id += 1;
        let duration = SystemTime::now().duration_since(UNIX_EPOCH).unwrap();
        let created_at = duration.as_secs() as f64 + duration.subsec_nanos() as f64 * 1e-9;

        let prover_task = UniversalHandler::get_task_from_input(&req.input)?;
        if prover_task.use_openvm_13 {
            eyre::bail!("prover do not support snark params base on openvm 13");
        }
        let prover_task: ProvingTask = prover_task.into();
        let vk = hex::encode(&prover_task.vk);

        let parent_handler = self
            .get_or_load_handler(&req.hard_fork_name, req.proof_type, &vk)
            .await?;

        // OpenVM v2+ aggregation circuits (batch/bundle) need deferral enabled
        // using their immediate child circuit's prover.
        let child_proof_type = match req.proof_type {
            ProofType::Batch => Some(ProofType::Chunk),
            ProofType::Bundle => Some(ProofType::Batch),
            _ => None,
        };
        if let Some(child_type) = child_proof_type {
            let child_vk = self
                .config
                .circuits
                .get(&req.hard_fork_name)
                .and_then(|c| c.child_circuit_vks.get(&child_type))
                .ok_or_else(|| {
                    eyre::eyre!(
                        "missing child circuit vk for {:?} in fork {}",
                        child_type,
                        req.hard_fork_name
                    )
                })?
                .clone();
            let child_handler = self
                .get_or_load_handler(&req.hard_fork_name, child_type, &child_vk)
                .await?;
            let mut parent_guard = parent_handler.lock().await;
            let child_guard = child_handler.lock().await;
            parent_guard.enable_deferral(&*child_guard)?;
        }

        let handle = Handle::current();
        let is_evm = req.proof_type == ProofType::Bundle;
        let task_handle = tokio::task::spawn_blocking(move || {
            handle.block_on(parent_handler.get_proof_data(&prover_task, is_evm))
        });
        self.current_task = Some(task_handle);

        Ok(ProveResponse {
            task_id: self.next_task_id.to_string(),
            proof_type: req.proof_type,
            circuit_version: req.circuit_version,
            hard_fork_name: req.hard_fork_name,
            status: TaskStatus::Proving,
            created_at,
            input: Some(req.input),
            ..Default::default()
        })
    }

    /// Load a handler for the given fork/proof-type/vk, reusing a cached one if available.
    async fn get_or_load_handler(
        &mut self,
        fork_name: &str,
        proof_type: ProofType,
        vk: &str,
    ) -> Result<Arc<Mutex<UniversalHandler>>> {
        if let Some(handler) = self.handlers.get(vk) {
            return Ok(handler.clone());
        }

        let base_config = self
            .config
            .circuits
            .get(fork_name)
            .ok_or_else(|| eyre::eyre!("coordinator sent unexpected forkname {}", fork_name))?;
        let url_base = if let Some(url) = base_config.location_data.asset_detours.get(vk) {
            url.clone()
        } else {
            base_config.location_data.gen_asset_url(vk, proof_type)?
        };
        let asset_path = base_config
            .location_data
            .get_asset(vk, &url_base, &base_config.workspace_path)
            .await?;
        let handler = Arc::new(Mutex::new(UniversalHandler::new(&asset_path)?));
        self.handlers.insert(vk.to_string(), handler.clone());
        Ok(handler)
    }
}
