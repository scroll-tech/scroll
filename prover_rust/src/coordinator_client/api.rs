use super::types::*;
use anyhow::{bail, Result};
use reqwest::{header::CONTENT_TYPE, Url};
use serde::Serialize;
use core::time::Duration;
use reqwest_middleware::{ClientBuilder, ClientWithMiddleware};
use reqwest_retry::{RetryTransientMiddleware, policies::ExponentialBackoff};

pub struct Api {
    url_base: Url,
    send_timeout: Duration,
    pub client: ClientWithMiddleware,
}

impl Api {
    pub fn new(url_base: &String, send_timeout: Duration, retry_count: u32, retry_wait_time_sec: u64) -> Result<Self> {
        let retry_wait_duration = core::time::Duration::from_secs(retry_wait_time_sec);
        let retry_policy = ExponentialBackoff::builder()
        .retry_bounds(retry_wait_duration / 2, retry_wait_duration)
        .build_with_max_retries(retry_count);

        let client = ClientBuilder::new(reqwest::Client::new())
        .with(RetryTransientMiddleware::new_with_policy(retry_policy))
        .build();

        Ok(Self {
            url_base: Url::parse(url_base)?,
            send_timeout,
            client,
        })
    }

    pub async fn challenge(&self) -> Result<Response<ChallengeResponseData>> {
        let method = "/coordinator/v1/challenge";
        let url = self.build_url(method)?;

        let response = self
            .client
            .get(url)
            .header(CONTENT_TYPE, "application/json")
            .timeout(self.send_timeout)
            .send()
            .await?;

        let response_body = response.text().await?;

        serde_json::from_str(&response_body).map_err(|e| anyhow::anyhow!(e))
    }

    pub async fn login(
        &self,
        req: &LoginRequest,
        token: &String,
    ) -> Result<Response<LoginResponseData>> {
        let method = "/coordinator/v1/login";
        self.post_with_token(method, req, token).await
    }

    pub async fn get_task(
        &self,
        req: &GetTaskRequest,
        token: &String,
    ) -> Result<Response<GetTaskResponseData>> {
        let method = "/coordinator/v1/get_task";
        self.post_with_token(method, req, token).await
    }

    pub async fn submit_proof(
        &self,
        req: &SubmitProofRequest,
        token: &String,
    ) -> Result<Response<SubmitProofResponseData>> {
        let method = "/coordinator/v1/submit_proof";
        self.post_with_token(method, req, token).await
    }

    async fn post_with_token<Req, Resp>(
        &self,
        method: &str,
        req: &Req,
        token: &String,
    ) -> Result<Resp>
    where
        Req: ?Sized + Serialize,
        Resp: serde::de::DeserializeOwned,
    {
        let url = self.build_url(method)?;
        let request_body = serde_json::to_string(req)?;

        log::info!("[coordinator client], {method}, request: {request_body}");
        let response = self
            .client
            .post(url)
            .header(CONTENT_TYPE, "application/json")
            .bearer_auth(token)
            .body(request_body)
            .timeout(self.send_timeout)
            .send()
            .await?;

        if response.status() != http::status::StatusCode::OK {
            log::error!(
                "[coordinator client], {method}, status not ok: {}",
                response.status()
            );
            bail!(
                "[coordinator client], {method}, status not ok: {}",
                response.status()
            )
        }

        let response_body = response.text().await?;

        log::info!("[coordinator client], {method}, response: {response_body}");
        serde_json::from_str(&response_body).map_err(|e| anyhow::anyhow!(e))
    }

    fn build_url(&self, method: &str) -> Result<Url> {
        self.url_base.join(method).map_err(|e| anyhow::anyhow!(e))
    }
}
