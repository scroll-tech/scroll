use super::types::*;
use anyhow::Result;
use reqwest::{header::CONTENT_TYPE, Url};
use serde::Serialize;

pub struct API {
    url_base: Url,
    pub client: reqwest::Client,
}

impl API {
    pub fn new(url_base: &String) -> Result<Self> {
        Ok(Self {
            url_base: Url::parse(&url_base)?,
            client: reqwest::Client::new(),
        })
    }

    pub async fn challenge(&self) -> Result<Response<ChallengeResponseData>> {
        let method = "/coordinator/v1/challenge";
        let url = self.build_url(method)?;

        let response = self.client
            .get(url)
            .header(CONTENT_TYPE, "application/json")
            .send()
            .await?;

        let response_body = response.text().await?;

        serde_json::from_str(&response_body).map_err(|e| anyhow::anyhow!(e))
    }

    pub async fn login(&self, req: &LoginRequest, token: &String) -> Result<Response<LoginResponseData>> {
        let method = "/coordinator/v1/login";
        self.post_with_token(&method, req, token).await
    }

    pub async fn get_task(&self, req: &GetTaskRequest, token: &String) -> Result<Response<GetTaskResponseData>> {
        let method = "/coordinator/v1/get_task";
        self.post_with_token(&method, req, token).await
    }

    pub async fn submit_proof(&self, req: &SubmitProofRequest, token: &String)  -> Result<Response<SubmitProofResponseData>> {
        let method = "/coordinator/v1/submit_proof";
        self.post_with_token(&method, req, token).await
    }

    async fn post_with_token<Req, Resp>(&self, method: &str, req: &Req, token: &String) -> Result<Resp>
    where
        Req: ?Sized + Serialize,
        Resp: serde::de::DeserializeOwned
    {
        let url = self.build_url(method)?;
        let request_body = serde_json::to_string(req)?;

        log::info!("[coordinator client], {method}, request: {request_body}");
        let response = self.client
            .post(url)
            .header(CONTENT_TYPE, "application/json")
            .bearer_auth(token)
            .body(request_body)
            .send()
            .await?;

        let response_body = response.text().await?;

        log::info!("[coordinator client], {method}, response: {response_body}");
        serde_json::from_str(&response_body).map_err(|e| anyhow::anyhow!(e))
    }

    fn build_url(&self, method: &str) -> Result<Url> {
        self.url_base.join(method).map_err(|e| anyhow::anyhow!(e))
    }
}