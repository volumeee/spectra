use reqwest::Client;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::time::Duration;

#[derive(Clone)]
pub struct SpectraClient {
    base_url: String,
    api_key: Option<String>,
    client: Client,
}

#[derive(Deserialize)]
struct ApiResponse {
    success: bool,
    data: Option<Value>,
    error: Option<ApiError>,
}

#[derive(Deserialize)]
struct ApiError {
    code: String,
    message: String,
}

impl SpectraClient {
    pub fn new(base_url: &str) -> Self {
        Self {
            base_url: base_url.trim_end_matches('/').to_string(),
            api_key: None,
            client: Client::builder()
                .timeout(Duration::from_secs(60))
                .build()
                .unwrap(),
        }
    }

    pub fn with_api_key(mut self, key: &str) -> Self {
        self.api_key = Some(key.to_string());
        self
    }

    pub async fn screenshot(&self, params: Value) -> Result<Value, String> {
        self.execute("screenshot", "capture", params).await
    }

    pub async fn pdf(&self, params: Value) -> Result<Value, String> {
        self.execute("pdf", "generate", params).await
    }

    pub async fn scrape(&self, params: Value) -> Result<Value, String> {
        self.execute("scrape", "extract", params).await
    }

    pub async fn execute(&self, plugin: &str, method: &str, params: Value) -> Result<Value, String> {
        let url = format!("{}/api/{}/{}", self.base_url, plugin, method);
        let mut req = self.client.post(&url).json(&params);
        if let Some(ref key) = self.api_key {
            req = req.header("Authorization", format!("Bearer {}", key));
        }
        let resp = req.send().await.map_err(|e| e.to_string())?;
        let api_resp: ApiResponse = resp.json().await.map_err(|e| e.to_string())?;
        if !api_resp.success {
            if let Some(err) = api_resp.error {
                return Err(format!("[{}] {}", err.code, err.message));
            }
            return Err("request failed".to_string());
        }
        Ok(api_resp.data.unwrap_or(Value::Null))
    }

    pub async fn health(&self) -> Result<bool, String> {
        let url = format!("{}/health", self.base_url);
        let resp = self.client.get(&url).send().await.map_err(|e| e.to_string())?;
        Ok(resp.status().is_success())
    }
}
