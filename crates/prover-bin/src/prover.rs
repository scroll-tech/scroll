use crate::zk_circuits_handler::{universal::UniversalHandler, assets::AssetsHandler, CircuitsHandler};
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
use serde::{Deserialize, Serialize};
use std::{
    collections::HashMap,
    fs::File,
    path::{Path, PathBuf},
    sync::{Arc, OnceLock},
    time::{SystemTime, UNIX_EPOCH},
};
use tokio::{runtime::Handle, sync::Mutex, task::JoinHandle};

#[derive(Clone, Serialize, Deserialize)]
pub struct AssetsLocationData {
    /// the base url to form a general downloading url for an asset, MUST HAVE A TRAILING SLASH
    pub base_url: url::Url,
    /// a altered url for specififed vk
    pub asset_detours: HashMap<String, url::Url>,
}

impl AssetsLocationData {

    fn gen_asset_url(&self, vk: &str, proof_type: ProofType) -> Result<url::Url> {
        if let Some(url) = self.asset_detours.get(vk) {
            Ok(url.clone())
        } else {
            Ok(self.base_url
            .join(match proof_type {
                ProofType::Chunk => format!("chunk/{vk}/"),
                ProofType::Batch => format!("batch/{vk}/"),
                ProofType::Bundle => format!("bundle/{vk}/"),
                _ => unreachable!("unreconginzed type"),
            }.as_str())?)
        }
    }

    pub fn validate(self) -> Result<Self> {
        if !self.base_url.path().ends_with('/') {
            eyre::bail!("base_url must have a trailing slash, got: {}", self.base_url);
        }
        Ok(self)
    }

    pub async fn get_asset(&self, vk: &str, proof_type: ProofType, base_path: impl AsRef<Path>) -> Result<PathBuf> {
        let download_files = ["app.vmexe", "openvm.toml"];
        
        // Step 1: Create a local path for storage
        let storage_path = base_path.as_ref().join(vk);
        std::fs::create_dir_all(&storage_path)?;
        
        // Step 2 & 3: Download each file if needed
        let url_base = self.gen_asset_url(vk, proof_type)?;
        let client = reqwest::Client::new();
        
        for filename in download_files.iter() {
            let local_file_path = storage_path.join(filename);
            let download_url = url_base.join(filename)?;

            // Check if file already exists
            if local_file_path.exists() {
                // Get file metadata to check size
                if let Ok(metadata) = std::fs::metadata(&local_file_path) {
                    // Make a HEAD request to get remote file size
                    
                    if let Ok(head_resp) = client.head(download_url.clone()).send().await {
                        if let Some(content_length) = head_resp.headers().get("content-length") {
                            if let Ok(remote_size) = content_length.to_str().unwrap_or("0").parse::<u64>() {
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
                eyre::bail!("Failed to download {}: HTTP status {}", filename, response.status());
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
    pub hard_fork_name: String,
    /// The path to save assets for a specified hard fork phase
    pub workspace_path: String,
    /// cached vk value to save some initial cost, for debugging only
    #[serde(default)]
    pub vks: HashMap<ProofType, String>,
}

pub struct LocalProver {
    config: LocalProverConfig,
    next_task_id: u64,
    current_task: Option<JoinHandle<Result<String>>>,

    handlers: HashMap<String, OnceLock<Arc<dyn CircuitsHandler>>>,
}

#[async_trait]
impl ProvingService for LocalProver {
    fn is_local(&self) -> bool {
        true
    }
    async fn get_vks(&self, _: GetVkRequest) -> GetVkResponse {
        // get vk has been deprecated in new prover with dynamic asset loading scheme
        GetVkResponse { vks: vec![], error: None }
    }
    async fn prove(&mut self, req: ProveRequest) -> ProveResponse {
        let handler = self.get_or_init_handler(&req.hard_fork_name, req.proof_type);
        match self.do_prove(req, handler).await {
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
                    Err(e) => QueryTaskResponse {
                        task_id: req.task_id,
                        status: TaskStatus::Failed,
                        error: Some(format!("proving task panicked: {}", e)),
                        ..Default::default()
                    },
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

impl LocalProver {
    pub fn new(config: LocalProverConfig) -> Self {
        let handlers = config
            .circuits
            .keys()
            .map(|k| (k.clone(), OnceLock::new()))
            .collect();
        Self {
            config,
            next_task_id: 0,
            current_task: None,
            handlers,
        }
    }

    async fn do_prove(
        &mut self,
        req: ProveRequest,
        handler: Arc<dyn CircuitsHandler>,
    ) -> Result<ProveResponse> {
        self.next_task_id += 1;
        let duration = SystemTime::now().duration_since(UNIX_EPOCH).unwrap();
        let created_at = duration.as_secs() as f64 + duration.subsec_nanos() as f64 * 1e-9;

        let req_clone = req.clone();
        let handle = Handle::current();
        let task_handle =
            tokio::task::spawn_blocking(move || handle.block_on(handler.get_proof_data(req_clone)));
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

    fn get_or_init_handler(&self, hard_fork_name: &str, proof_type: ProofType) -> Arc<dyn CircuitsHandler> {
        let lk = self
            .handlers
            .get(hard_fork_name)
            .expect("coordinator should never sent unexpected forkname");

        lk.get_or_init(||{
            // if we got assigned a task for an unknown hard fork, there is something wrong in the
            // coordinator
            let config = self.config.circuits.get(hard_fork_name).unwrap();
            let circuits_handler = Mutex::new(UniversalHandler::new(config, proof_type).unwrap());
            Arc::new(circuits_handler)
        }).clone()
    }

    pub fn dump_verifier_assets(&self, hard_fork_name: &str, out_path: &Path) -> Result<()> {
        let config = self
            .config
            .circuits
            .get(hard_fork_name)
            .ok_or_else(|| eyre::eyre!("no corresponding config for fork {hard_fork_name}"))?;

        if !config.vks.is_empty() {
            eyre::bail!("clean vks cache first or we will have wrong dumped vk");
        }

        let workspace_path = &config.workspace_path;
        let universal_prover = AssetsHandler::new(config);
        let _ = universal_prover
            .get_evm_prover()
            .dump_universal_verifier(Some(out_path))?;

        #[derive(Debug, serde::Serialize)]
        struct VKDump {
            pub chunk_vk: String,
            pub batch_vk: String,
            pub bundle_vk: String,
        }

        let dump = VKDump {
            chunk_vk: universal_prover.get_vk_and_cache(ProofType::Chunk),
            batch_vk: universal_prover.get_vk_and_cache(ProofType::Batch),
            bundle_vk: universal_prover.get_vk_and_cache(ProofType::Bundle),
        };

        let f = File::create(out_path.join("openVmVk.json"))?;
        serde_json::to_writer(f, &dump)?;

        // Copy verifier.bin from workspace bundle directory to output path
        let bundle_verifier_path = Path::new(workspace_path)
            .join("bundle")
            .join("verifier.bin");
        if bundle_verifier_path.exists() {
            let dest_path = out_path.join("verifier.bin");
            std::fs::copy(&bundle_verifier_path, &dest_path)
                .map_err(|e| eyre::eyre!("Failed to copy verifier.bin: {}", e))?;
        } else {
            eprintln!(
                "Warning: verifier.bin not found at {:?}",
                bundle_verifier_path
            );
        }

        Ok(())
    }
}
