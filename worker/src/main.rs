use clap::Parser;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use tracing::{info, error};

mod config;
mod error;
mod consumer;
mod storage;
mod transcode;
mod idempotency;

// Include generated protobufs
pub mod pipeline {
    pub mod v1 {
        include!("pipeline.v1.rs");
    }
}

#[derive(Parser, Debug)]
#[command(author, version, about)]
struct Args {
    #[arg(short, long)]
    config: Option<String>,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 1. Initialize logging
    tracing_subscriber::fmt()
        .json()
        .with_env_filter(tracing_subscriber::EnvFilter::from_default_env())
        .init();

    // 2. Parse args & load config
    let _args = Args::parse();
    let cfg = config::Config::from_env()?;

    let hostname = std::env::var("HOSTNAME").unwrap_or_else(|_| "worker".to_string());
    let worker_id = format!("{}-{}", hostname, ulid::Ulid::new());
    info!(worker_id = %worker_id, concurrency = cfg.worker_concurrency, "Starting Nolan worker");

    // 3. Connect to Valkey
    let redis_client = redis::Client::open(cfg.valkey_url.clone())?;
    let valkey_conn = redis_client.get_connection_manager().await?;

    // 4. Connect to MinIO/S3
    let s3_client = storage::StorageClient::new(
        &cfg.minio_endpoint,
        &cfg.minio_access_key,
        &cfg.minio_secret_key,
        cfg.minio_use_ssl,
    )?;

    // 5. Setup shutdown handler
    let shutdown_flag = Arc::new(AtomicBool::new(false));
    let shutdown_flag_clone = shutdown_flag.clone();
    tokio::spawn(async move {
        tokio::signal::ctrl_c().await.expect("failed to listen for event");
        info!("Received shutdown signal");
        shutdown_flag_clone.store(true, Ordering::SeqCst);
    });

    // 6. Start consumer loop
    let consumer = consumer::Consumer::new(valkey_conn, s3_client, cfg, worker_id, shutdown_flag);
    if let Err(e) = consumer.run().await {
        error!(error = %e, "Consumer exited with error");
        std::process::exit(1);
    }

    info!("Worker exited gracefully");
    Ok(())
}
