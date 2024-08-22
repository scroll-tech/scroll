mod api;
mod errors;
pub mod listener;
pub mod types;

use anyhow::{bail, Context, Ok, Result};
use std::rc::Rc;

use api::Api;
use errors::*;
use listener::Listener;
use tokio::runtime::Runtime;
use types::*;

use crate::{config::Config, key_signer::KeySigner};

pub use errors::ProofStatusNotOKError;

pub struct CoordinatorClient<'a> {
    api: Api,
    token: Option<String>,
    config: &'a Config,
    key_signer: Rc<KeySigner>,
    rt: Runtime,
    listener: Box<dyn Listener>,
    vks: Vec<String>,
}

impl<'a> CoordinatorClient<'a> {
    pub fn new(
        config: &'a Config,
        key_signer: Rc<KeySigner>,
        listener: Box<dyn Listener>,
        vks: Vec<String>,
    ) -> Result<Self> {
        let rt = tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()?;

        let api = Api::new(
            &config.coordinator.base_url,
            core::time::Duration::from_secs(config.coordinator.connection_timeout_sec),
            config.coordinator.retry_count,
            config.coordinator.retry_wait_time_sec,
        )?;
        let mut client = Self {
            api,
            token: None,
            config,
            key_signer,
            rt,
            listener,
            vks,
        };
        client.login()?;
        Ok(client)
    }

    fn login(&mut self) -> Result<()> {
        let api = &self.api;
        let challenge_response = self.rt.block_on(api.challenge())?;
        if challenge_response.errcode != ErrorCode::Success {
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
            prover_version: crate::version::get_version(),
            prover_types: vec![self.config.prover_type],
            vks: self.vks.clone(),
        };

        let buffer = rlp::encode(&login_message);
        let signature = self.key_signer.sign_buffer(&buffer)?;
        let login_request = LoginRequest {
            message: login_message,
            public_key: self.key_signer.get_public_key(),
            signature,
        };
        let login_response = self.rt.block_on(api.login(&login_request, &token))?;
        if login_response.errcode != ErrorCode::Success {
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
        if response.errcode == ErrorCode::ErrJWTTokenExpired {
            log::info!("JWT expired, attempting to re-login");
            self.login().context("JWT expired, re-login failed")?;
            log::info!("re-login success");
            return self.action_with_re_login(req, f);
        } else if response.errcode != ErrorCode::Success {
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
            .block_on(self.api.submit_proof(req, self.token.as_ref().unwrap()))?;
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
