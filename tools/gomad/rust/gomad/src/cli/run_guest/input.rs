use crate::{
    campaign::origin,
    evidence::{canonical, store},
    install::manifest,
};
use std::{
    fs,
    io::Write,
    os::unix::fs::{DirBuilderExt as _, OpenOptionsExt as _},
    path::{Path, PathBuf},
};

pub(super) struct Input {
    pub bytes: Vec<u8>,
    pub directory: PathBuf,
    digest: String,
}

impl Input {
    pub(super) fn capture(module: &Path, artifacts: &Path) -> Result<Self, String> {
        let bytes = manifest::read(module, 512 << 20)?;
        let digest = canonical::digest(&bytes);
        let root = std::path::absolute(artifacts)
            .map_err(|e| e.to_string())?
            .join("unaccepted-inputs");
        store::durable_directory(&root, |_| Ok(()))?;
        let directory = root.join(origin::identity()?);
        fs::DirBuilder::new()
            .mode(0o700)
            .create(&directory)
            .map_err(|e| e.to_string())?;
        let input = Self {
            bytes,
            directory,
            digest,
        };
        input.write("input.bin", &input.bytes)?;
        input.write("input.json", &canonical::encode(&serde_json::json!({"schema":"gomad.unaccepted-input/v1","accepted":false,"replayable_artifact":false,"bytes":input.bytes.len(),"sha256":input.digest,"module_path_hex":origin::argument(module.as_os_str())}))?)?;
        Ok(input)
    }

    fn write(&self, name: &str, bytes: &[u8]) -> Result<(), String> {
        let mut file = fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(self.directory.join(name))
            .map_err(|e| e.to_string())?;
        file.write_all(bytes)
            .and_then(|()| file.sync_all())
            .map_err(|e| e.to_string())?;
        fs::File::open(&self.directory)
            .and_then(|directory| directory.sync_all())
            .map_err(|e| e.to_string())
    }

    pub(super) fn reject(&self, status: i32, message: &str) -> Result<(), String> {
        self.write("refusal.json", &canonical::encode(&serde_json::json!({"schema":"gomad.unaccepted-input-refusal/v1","accepted":false,"status":status,"message":message,"input_sha256":self.digest}))?)
    }

    pub(super) fn accepted(self) -> Result<(), String> {
        for name in ["input.bin", "input.json"] {
            fs::remove_file(self.directory.join(name)).map_err(|e| e.to_string())?;
        }
        fs::remove_dir(&self.directory).map_err(|e| e.to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn refusal_keeps_exact_unaccepted_bytes_after_original_is_removed() {
        let root =
            std::env::temp_dir().join(format!("gomad-unaccepted-{}", origin::identity().unwrap()));
        fs::DirBuilder::new().mode(0o700).create(&root).unwrap();
        let source = root.join("actual-module.wasm");
        fs::write(&source, b"actual emitted input bytes").unwrap();
        let staged = Input::capture(&source, &root.join("artifacts")).unwrap();
        let location = staged.directory.clone();
        staged.reject(2, "unsupported metadata").unwrap();
        fs::remove_file(&source).unwrap();
        assert_eq!(
            fs::read(location.join("input.bin")).unwrap(),
            b"actual emitted input bytes"
        );
        let metadata: serde_json::Value =
            serde_json::from_slice(&fs::read(location.join("input.json")).unwrap()).unwrap();
        assert_eq!(metadata["accepted"], false);
        assert_eq!(metadata["replayable_artifact"], false);
        assert_eq!(
            metadata["sha256"],
            canonical::digest(b"actual emitted input bytes")
        );
        assert!(crate::evidence::store::Opened::open(&location).is_err());
        fs::remove_dir_all(root).unwrap();
    }
    #[test]
    fn accepted_input_retires_only_its_own_private_diagnostic_copy() {
        let root =
            std::env::temp_dir().join(format!("gomad-accepted-{}", origin::identity().unwrap()));
        fs::DirBuilder::new().mode(0o700).create(&root).unwrap();
        let source = root.join("actual-module.wasm");
        fs::write(&source, b"retained by accepted preparation").unwrap();
        let staged = Input::capture(&source, &root.join("artifacts")).unwrap();
        let location = staged.directory.clone();
        staged.accepted().unwrap();
        assert!(!location.exists());
        assert_eq!(
            fs::read(source).unwrap(),
            b"retained by accepted preparation"
        );
        fs::remove_dir_all(root).unwrap();
    }
}
