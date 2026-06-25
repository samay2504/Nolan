use crate::error::WorkerError;
use redis::AsyncCommands;

pub async fn is_already_done(conn: &mut redis::aio::ConnectionManager, job_id: &str) -> Result<bool, WorkerError> {
    let key = format!("pipeline:job:{}:done", job_id);
    let exists: bool = conn.exists(&key).await?;
    Ok(exists)
}

pub async fn mark_done(conn: &mut redis::aio::ConnectionManager, job_id: &str) -> Result<(), WorkerError> {
    let key = format!("pipeline:job:{}:done", job_id);
    // SET NX EX 604800 (7 days)
    let _: () = redis::cmd("SET")
        .arg(&key)
        .arg(1)
        .arg("NX")
        .arg("EX")
        .arg(604800)
        .query_async(conn)
        .await?;
    Ok(())
}
