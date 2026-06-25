use crate::error::WorkerError;
use crate::pipeline::v1::Rendition;
use std::path::{Path, PathBuf};
use std::process::Command;
use tracing::{info, error};

pub struct TranscodeOutput {
    pub renditions: Vec<RenditionOutput>,
    pub thumbnail_path: Option<PathBuf>,
}

pub struct RenditionOutput {
    pub resolution: String,
    pub container: String,
    pub manifest_path: PathBuf,
    pub segment_paths: Vec<PathBuf>,
}

pub fn transcode_video(input_path: &Path, output_dir: &Path, targets: &[Rendition]) -> Result<TranscodeOutput, WorkerError> {
    info!("Starting ffmpeg transcode for {:?}", input_path);
    
    let mut outputs = Vec::new();
    
    for target in targets {
        let res_str = match target.resolution {
            1 => "480p",
            2 => "720p",
            3 => "1080p",
            4 => "4k",
            _ => "480p",
        };
        
        let target_dir = output_dir.join(res_str);
        std::fs::create_dir_all(&target_dir).map_err(|e| WorkerError::Permanent(e.to_string()))?;
        
        let manifest_name = "master.m3u8";
        let manifest_path = target_dir.join(manifest_name);
        
        let scale_filter = match target.resolution {
            1 => "scale=-2:480",
            2 => "scale=-2:720",
            3 => "scale=-2:1080",
            4 => "scale=-2:2160",
            _ => "scale=-2:480",
        };
        
        info!("Running ffmpeg for {}", res_str);
        
        let output = Command::new("ffmpeg")
            .current_dir(&target_dir)
            .arg("-y")
            .arg("-i")
            .arg(input_path)
            .arg("-vf")
            .arg(scale_filter)
            .arg("-c:v")
            .arg("libvpx-vp9") 
            .arg("-b:v")
            .arg("1000k") 
            .arg("-c:a")
            .arg("libopus")
            .arg("-f")
            .arg("hls")
            .arg("-hls_time")
            .arg("4")
            .arg("-hls_playlist_type")
            .arg("vod")
            .arg("-hls_segment_type")
            .arg("fmp4")
            .arg("-hls_segment_filename")
            .arg("segment_%03d.m4s")
            .arg(manifest_name)
            .output()
            .map_err(|e| WorkerError::Ffmpeg(e.to_string()))?;
            
        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            error!("ffmpeg failed: {}", stderr);
            return Err(WorkerError::Ffmpeg(stderr.to_string()));
        }
        
        // Collect segments
        let mut segments = Vec::new();
        for entry in std::fs::read_dir(&target_dir).unwrap() {
            let entry = entry.unwrap();
            let path = entry.path();
            let ext = path.extension().and_then(|s| s.to_str());
            if ext == Some("ts") || ext == Some("m4s") || ext == Some("mp4") {
                segments.push(path);
            }
        }
        
        outputs.push(RenditionOutput {
            resolution: res_str.to_string(),
            container: "hls".to_string(),
            manifest_path,
            segment_paths: segments,
        });
    }

    Ok(TranscodeOutput {
        renditions: outputs,
        thumbnail_path: None,
    })
}
