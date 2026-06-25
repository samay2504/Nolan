use crate::error::WorkerError;
use aws_sdk_s3::Client;
use aws_sdk_s3::primitives::ByteStream;
use std::path::Path;
use tokio::io::AsyncWriteExt;
use tracing::info;

pub struct StorageClient {
    client: Client,
}

impl StorageClient {
    pub fn new(endpoint: &str, access_key: &str, secret_key: &str, _use_ssl: bool) -> Result<Self, WorkerError> {
        let creds = aws_credential_types::Credentials::new(
            access_key,
            secret_key,
            None,
            None,
            "static",
        );
        let endpoint_url = if endpoint.starts_with("http") {
            endpoint.to_string()
        } else {
            format!("http://{}", endpoint)
        };
        
        let config = aws_sdk_s3::config::Builder::new()
            .credentials_provider(creds)
            .endpoint_url(endpoint_url)
            .force_path_style(true) // For MinIO
            .region(aws_sdk_s3::config::Region::new("us-east-1"))
            .build();

        Ok(Self {
            client: Client::from_conf(config),
        })
    }

    pub async fn download_source(&self, bucket: &str, key: &str, dest: &Path) -> Result<(), WorkerError> {
        info!("Downloading s3://{}/{} to {:?}", bucket, key, dest);
        let mut resp = self.client.get_object()
            .bucket(bucket)
            .key(key)
            .send()
            .await
            .map_err(|e| WorkerError::Storage(e.to_string()))?;

        let mut file = tokio::fs::File::create(dest).await
            .map_err(|e| WorkerError::Storage(e.to_string()))?;

        while let Some(bytes) = resp.body.try_next().await.map_err(|e| WorkerError::Storage(e.to_string()))? {
            file.write_all(&bytes).await.map_err(|e| WorkerError::Storage(e.to_string()))?;
        }
        
        file.sync_all().await.map_err(|e| WorkerError::Storage(e.to_string()))?;
        Ok(())
    }

    pub async fn upload_file(&self, bucket: &str, key: &str, path: &Path, content_type: &str, cache_control: &str) -> Result<(), WorkerError> {
        info!("Uploading {:?} to s3://{}/{}", path, bucket, key);
        let body = ByteStream::from_path(path).await.map_err(|e| WorkerError::Storage(e.to_string()))?;
        self.client.put_object()
            .bucket(bucket)
            .key(key)
            .content_type(content_type)
            .cache_control(cache_control)
            .body(body)
            .send()
            .await
            .map_err(|e| WorkerError::Storage(e.to_string()))?;
        Ok(())
    }
}
