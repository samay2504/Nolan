use crate::config::Config;
use crate::error::WorkerError;
use crate::idempotency;
use crate::pipeline::v1::TranscodeJob;
use crate::storage::StorageClient;
use prost::Message;
use redis::AsyncCommands;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::Duration;
use tracing::{error, info, span, Level, instrument};
use redis::streams::{StreamReadOptions, StreamReadReply, StreamId};
use tokio::sync::Semaphore;

pub struct Consumer {
    valkey_conn: redis::aio::ConnectionManager,
    s3_client: StorageClient,
    config: Config,
    worker_id: String,
    shutdown_flag: Arc<AtomicBool>,
}

impl Consumer {
    pub fn new(
        valkey_conn: redis::aio::ConnectionManager,
        s3_client: StorageClient,
        config: Config,
        worker_id: String,
        shutdown_flag: Arc<AtomicBool>,
    ) -> Self {
        Self {
            valkey_conn,
            s3_client,
            config,
            worker_id,
            shutdown_flag,
        }
    }

    #[instrument(skip(self))]
    pub async fn run(mut self) -> Result<(), WorkerError> {
        // Ensure consumer group exists
        let result: redis::RedisResult<()> = self.valkey_conn.xgroup_create_mkstream(
            &self.config.stream_key,
            &self.config.consumer_group,
            "$",
        ).await;

        if let Err(e) = result {
            if !e.to_string().contains("BUSYGROUP") {
                return Err(WorkerError::Valkey(e));
            }
        }

        info!("Consumer group ready. Starting to process jobs.");
        let semaphore = Arc::new(Semaphore::new(self.config.worker_concurrency));

        // Consume loop
        let opts = StreamReadOptions::default().group(&self.config.consumer_group, &self.worker_id).block(5000).count(1);

        while !self.shutdown_flag.load(Ordering::SeqCst) {
            let reply: redis::RedisResult<StreamReadReply> = self.valkey_conn.xread_options(
                &[&self.config.stream_key],
                &[">"],
                &opts,
            ).await;

            match reply {
                Ok(reply) => {
                    for key in reply.keys {
                        for id in key.ids {
                            if let Some(payload) = id.map.get("payload") {
                                if let Ok(data) = redis::from_redis_value::<Vec<u8>>(payload) {
                                    self.process_message(&id.id, &data, semaphore.clone()).await?;
                                }
                            }
                        }
                    }
                }
                Err(e) => {
                    // Nil is returned on timeout
                    if e.kind() != redis::ErrorKind::IoError {
                        error!(error = %e, "Error reading from stream");
                    }
                    tokio::time::sleep(Duration::from_millis(100)).await;
                }
            }
        }

        Ok(())
    }

    async fn process_message(&mut self, stream_id: &str, data: &[u8], semaphore: Arc<Semaphore>) -> Result<(), WorkerError> {
        let job = TranscodeJob::decode(data)?;
        let span = span!(Level::INFO, "process_job", job_id = %job.job_id, video_id = %job.video_id);
        let _enter = span.enter();

        info!("Received job {}", job.job_id);

        if idempotency::is_already_done(&mut self.valkey_conn, &job.job_id).await? {
            info!("Job already processed, skipping.");
            self.ack_message(stream_id).await?;
            return Ok(());
        }

        // Mark as processing
        let _: () = self.valkey_conn.hset(
            format!("pipeline:job:{}", job.job_id),
            "status",
            "processing",
        ).await?;

        // Acquire concurrency permit
        let permit = semaphore.acquire_owned().await.map_err(|e| WorkerError::Permanent(e.to_string()))?;

        info!("Downloading source video...");
        let temp_dir = tempfile::tempdir().map_err(|e| WorkerError::Permanent(e.to_string()))?;
        let input_path = temp_dir.path().join("source");
        
        self.s3_client.download_source(&job.source_bucket, &job.source_key, &input_path).await?;

        let output_dir = temp_dir.path().join("outputs");
        let targets = job.targets.clone();
        
        info!("Starting transcode pipeline...");
        let output = tokio::task::spawn_blocking(move || {
            crate::transcode::transcode_video(&input_path, &output_dir, &targets)
        }).await.map_err(|e| WorkerError::Permanent(e.to_string()))??;

        info!("Uploading renditions...");
        for rendition in output.renditions {
            // Upload manifest
            let manifest_key = format!("{}/hls/{}/master.m3u8", job.video_id, rendition.resolution);
            self.s3_client.upload_file("processed", &manifest_key, &rendition.manifest_path, "application/vnd.apple.mpegurl", "public, max-age=10").await?;
            
            // Upload segments
            for segment_path in rendition.segment_paths {
                let segment_name = segment_path.file_name().unwrap().to_str().unwrap();
                let segment_key = format!("{}/hls/{}/{}", job.video_id, rendition.resolution, segment_name);
                self.s3_client.upload_file("processed", &segment_key, &segment_path, "video/mp2t", "public, max-age=31536000, immutable").await?;
            }
        }

        idempotency::mark_done(&mut self.valkey_conn, &job.job_id).await?;
        self.ack_message(stream_id).await?;
        
        let _: () = self.valkey_conn.hset(
            format!("pipeline:job:{}", job.job_id),
            "status",
            "completed",
        ).await?;

        info!("Job completed successfully.");
        drop(permit);
        Ok(())
    }

    async fn ack_message(&mut self, stream_id: &str) -> Result<(), WorkerError> {
        let _: () = self.valkey_conn.xack(
            &self.config.stream_key,
            &self.config.consumer_group,
            &[stream_id],
        ).await?;
        Ok(())
    }
}
