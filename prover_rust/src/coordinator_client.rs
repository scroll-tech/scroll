mod api;
mod errors;
pub mod listener;
pub mod types;

use anyhow::{bail, Context, Ok, Result};
use std::rc::Rc;

use api::API;
use errors::*;
use listener::Listener;
use log;
use tokio::runtime::Runtime;
use types::*;

use crate::key_signer::KeySigner;

pub struct Config {
    pub endpoint: String,
    pub prover_name: String,
    pub prover_version: String,
}

pub struct CoordinatorClient {
    api: API,
    token: Option<String>,
    config: Config,
    key_signer: Rc<KeySigner>,
    rt: Runtime,
    listener: Box<dyn Listener>,
}

impl CoordinatorClient {
    pub fn new(
        config: Config,
        key_signer: Rc<KeySigner>,
        listener: Box<dyn Listener>,
    ) -> Result<Self> {
        let rt = tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()?;

        let mut client = Self {
            api: API::new(&config.endpoint)?,
            token: None,
            config,
            key_signer,
            rt,
            listener,
        };
        client.login()?;
        Ok(client)
    }

    fn login(&mut self) -> Result<()> {
        let api = &self.api;
        let challenge_response = self.rt.block_on(api.challenge())?;
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
            challenge: token.clone(),
            prover_name: self.config.prover_name.clone(),
            prover_version: self.config.prover_version.clone(),
        };

        let buffer = login_message.rlp();
        let signature = self.key_signer.sign_buffer(&buffer)?;
        let login_request = LoginRequest {
            message: login_message,
            signature,
        };
        let login_response = self.rt.block_on(api.login(&login_request, &token))?;
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

    fn action_with_re_login<T, F, R>(&mut self, req: &R, mut f: F) -> Result<Response<T>>
    where
        F: FnMut(&mut Self, &R) -> Result<Response<T>>,
    {
        let response = f(self, req)?;
        if response.errcode == ErrJWTTokenExpired {
            log::info!("JWT expired, attempting to re-login");
            self.login().context("JWT expired, re-login failed")?;
            log::info!("re-login success");
            return f(self, req);
        } else if response.errcode != Success {
            bail!("action failed: {}", response.errmsg)
        }
        Ok(response)
    }

    fn do_get_task(&mut self, req: &GetTaskRequest) -> Result<Response<GetTaskResponseData>> {
        self.rt
            .block_on(self.api.get_task(req, self.token.as_ref().unwrap()))
    }

    pub fn get_task(&mut self, req: &GetTaskRequest) -> Result<Response<GetTaskResponseData>> {
        self.action_with_re_login(req, |s, req| s.do_get_task(req))
    }

    fn do_submit_proof(
        &mut self,
        req: &SubmitProofRequest,
    ) -> Result<Response<SubmitProofResponseData>> {
        let response = self
            .rt
            .block_on(self.api.submit_proof(req, &self.token.as_ref().unwrap()))?;
        self.listener.on_proof_submitted(req);
        Ok(response)
    }

    pub fn submit_proof(
        &mut self,
        req: &SubmitProofRequest,
    ) -> Result<Response<SubmitProofResponseData>> {
        self.action_with_re_login(req, |s, req| s.do_submit_proof(req))
    }
}
