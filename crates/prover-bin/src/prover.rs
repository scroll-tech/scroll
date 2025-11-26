use async_trait::async_trait;
use scroll_proving_sdk::{
    config::Config as SdkConfig,
    prover::{
        proving_service::{
            GetVkRequest, GetVkResponse, ProveRequest, ProveResponse, QueryTaskRequest,
            QueryTaskResponse,
        },
        ProvingService,
    },
};
use serde::{Deserialize, Serialize};
use std::path::Path;

mod local;
pub use local::{LocalProver, LocalProverConfig};

mod axiom;
pub use axiom::{AxiomProver, AxiomProverConfig};

#[derive(Debug)]
pub enum Prover {
    Local(LocalProver),
    Axiom(AxiomProver),
}

#[derive(Debug, Copy, Clone, PartialEq, Eq, Hash, Serialize, Deserialize, clap::ValueEnum)]
pub enum ProverKind {
    Local,
    Axiom,
}

impl ProverKind {
    pub fn create_from_file<P: AsRef<Path>>(
        &self,
        file_name: P,
    ) -> eyre::Result<(SdkConfig, Prover)> {
        match self {
            ProverKind::Local => {
                let config = LocalProverConfig::from_file(file_name)?;
                let sdk_config = config.sdk_config.clone();
                let prover = LocalProver::new(config);
                Ok((sdk_config, Prover::Local(prover)))
            }
            ProverKind::Axiom => {
                let config = AxiomProverConfig::from_file(file_name)?;
                let sdk_config = config.sdk_config.clone();
                let prover = AxiomProver::new(config);
                Ok((sdk_config, Prover::Axiom(prover)))
            }
        }
    }
}

#[async_trait]
impl ProvingService for Prover {
    fn is_local(&self) -> bool {
        match self {
            Prover::Local(p) => p.is_local(),
            Prover::Axiom(p) => p.is_local(),
        }
    }

    async fn get_vks(&self, req: GetVkRequest) -> GetVkResponse {
        match self {
            Prover::Local(p) => p.get_vks(req).await,
            Prover::Axiom(p) => p.get_vks(req).await,
        }
    }

    async fn prove(&mut self, req: ProveRequest) -> ProveResponse {
        match self {
            Prover::Local(p) => p.prove(req).await,
            Prover::Axiom(p) => p.prove(req).await,
        }
    }

    async fn query_task(&mut self, req: QueryTaskRequest) -> QueryTaskResponse {
        match self {
            Prover::Local(p) => p.query_task(req).await,
            Prover::Axiom(p) => p.query_task(req).await,
        }
    }
}

impl From<LocalProver> for Prover {
    fn from(p: LocalProver) -> Self {
        Prover::Local(p)
    }
}

impl From<AxiomProver> for Prover {
    fn from(p: AxiomProver) -> Self {
        Prover::Axiom(p)
    }
}
