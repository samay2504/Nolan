fn main() -> Result<(), Box<dyn std::error::Error>> {
    prost_build::Config::new()
        .out_dir("src") // Or OUT_DIR, but this makes it easier to include
        .compile_protos(&["../proto/pipeline/v1/transcode.proto"], &["../proto"])?;
    Ok(())
}
