mod errors;
pub mod types;
mod api;

use anyhow::{bail, Context, Ok, Result};

use types::*;
use errors::*;
use api::API;
use futures::executor::block_on;
use log;

use crate::key_signer::KeySigner;

pub struct Config {
    pub endpoint: String,
    pub prover_name: String,
    pub prover_version: String,
    pub hard_fork_name: String,
}
pub struct CoordinatorClient<'a> {
    api: API,
    token: Option<String>,
    config: Config,
    key_signer: &'a KeySigner,
}

impl<'a> CoordinatorClient<'a> {
    pub fn new(config: Config, key_signer: &'a KeySigner) -> Result<Self> {
        let mut client = Self {
            api: API::new(config.endpoint)?,
            token: None,
            config,
            key_signer,
        };
        client.login()?;
        Ok(client)
    }

    fn login(&mut self) -> Result<()> {
        let api = self.api;
        let challenge_response = block_on(api.challenge())?;
        if challenge_response.errcode != Success {
            bail!("challenge failed: {}", challenge_response.errmsg)
        }
        let mut token: String;
        if let Some(r) = challenge_response.data {
            token = r.token;
        } else {
            bail!("challenge failed: got empty token")
        }

        let login_message = LoginMessage {
            challenge: token,
            prover_name: self.config.prover_name,
            prover_version: self.config.prover_version,
            hard_fork_name: self.config.hard_fork_name,
        };

        let buffer = login_message.rlp();
        let signature = self.key_signer.sign_buffer(&buffer)?;
        let login_request = LoginRequest {
            message: login_message,
            signature: signature,
        };
        let login_response = block_on(api.login(&login_request, &token))?;
        if login_response.errcode != Success {
            bail!("login failed: {}", login_response.errmsg)
        }
        if let Some(r) = login_response.data {
            token = r.token;
        } else {
            bail!("login failed: got empty token")
        }
        self.token = Some(token);
        Ok(())
    }

    pub fn get_task(&mut self, req: &GetTaskRequest) -> Result<Response<GetTaskResponseData>> {
        let response = block_on(self.api.get_task(req, &self.token.unwrap()))?;
        
        if response.errcode == ErrJWTTokenExpired {
            log::info!("JWT expired, attempting to re-login");
            self.login().context("JWT expired, re-login failed")?;
            log::info!("re-login success");
        } else if response.errcode != Success {
            bail!("get task failed: {}", response.errmsg)
        }
        Ok(response)
    }

    pub fn submit_proof(&mut self, req: &SubmitProofRequest) -> Result<Response<SubmitProofResponseData>> {
        let response = block_on(self.api.submit_proof(req, &self.token.unwrap()))?;
        
        if response.errcode == ErrJWTTokenExpired {
            log::info!("JWT expired, attempting to re-login");
            self.login().context("JWT expired, re-login failed")?;
            log::info!("re-login success");
        } else if response.errcode != Success {
            bail!("submit proof failed: {}", response.errmsg)
        }
        Ok(response)
    }
}