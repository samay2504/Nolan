use thiserror::Error;

#[derive(Debug, Error)]
pub enum WorkerError {
    #[error("retryable: {0}")]
    Retryable(String),
    #[error("permanent: {0}")]
    Permanent(String),
    #[error("malformed input: {0}")]
    MalformedInput(String),
    #[error("valkey error: {0}")]
    Valkey(#[from] redis::RedisError),
    #[error("storage error: {0}")]
    Storage(String),
    #[error("ffmpeg error: {0}")]
    Ffmpeg(String),
    #[error("config error: {0}")]
    Config(String),
    #[error("protobuf decode error: {0}")]
    ProtoDecode(#[from] prost::DecodeError),
    #[error("io error: {0}")]
    Io(#[from] std::io::Error),
}

impl WorkerError {
    pub fn is_retryable(&self) -> bool {
        matches!(self, Self::Retryable(_) | Self::Valkey(_) | Self::Storage(_))
    }
}
