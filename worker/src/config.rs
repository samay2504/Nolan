use crate::error::WorkerError;
use std::env;

#[derive(Debug, Clone)]
pub struct Config {
    pub valkey_url: String,
    pub minio_endpoint: String,
    pub minio_access_key: String,
    pub minio_secret_key: String,
    pub minio_use_ssl: bool,
    pub worker_concurrency: usize,
    pub max_job_retries: u32,
    pub reclaim_timeout_ms: u64,
    pub stream_key: String,
    pub consumer_group: String,
}

impl Config {
    pub fn from_env() -> Result<Self, WorkerError> {
        // Construct valkey URL from env vars
        let valkey_host = env::var("VALKEY_HOST").unwrap_or_else(|_| "valkey".to_string());
        let valkey_pass = env::var("WORKER_VALKEY_PASSWORD").unwrap_or_default();
        let valkey_url = if valkey_pass.is_empty() {
            format!("redis://{}", valkey_host)
        } else {
            format!("redis://worker:{}@{}", valkey_pass, valkey_host)
        };

        Ok(Config {
            valkey_url,
            minio_endpoint: env::var("MINIO_ENDPOINT").unwrap_or_else(|_| "http://localhost:9000".to_string()),
            minio_access_key: env::var("MINIO_ROOT_USER").unwrap_or_else(|_| "minioadmin".to_string()),
            minio_secret_key: env::var("MINIO_ROOT_PASSWORD").unwrap_or_else(|_| "minioadmin".to_string()),
            minio_use_ssl: env::var("MINIO_USE_SSL").map(|v| v == "true").unwrap_or(false),
            worker_concurrency: env::var("WORKER_CONCURRENCY").unwrap_or_else(|_| "2".to_string()).parse().unwrap_or(2),
            max_job_retries: env::var("MAX_JOB_RETRIES").unwrap_or_else(|_| "3".to_string()).parse().unwrap_or(3),
            reclaim_timeout_ms: env::var("RECLAIM_TIMEOUT_MS").unwrap_or_else(|_| "300000".to_string()).parse().unwrap_or(300000),
            stream_key: env::var("STREAM_KEY").unwrap_or_else(|_| "pipeline:jobs:transcode".to_string()),
            consumer_group: env::var("CONSUMER_GROUP").unwrap_or_else(|_| "transcode-workers".to_string()),
        })
    }
}
