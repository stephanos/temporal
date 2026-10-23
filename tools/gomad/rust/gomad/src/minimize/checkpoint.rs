use crate::evidence::{
    canonical,
    store::{Opened, open},
};
use rustix::fs::{self, AtFlags, FlockOperation, Mode, OFlags};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{
    fs::File,
    io::Write,
    path::{Path, PathBuf},
};

const FORMAT: &str = "gomad.minimize-checkpoint/v2";
const MAX_CHECKPOINT_BYTES: u64 = 16 << 20;

#[derive(Clone, Debug, Deserialize, PartialEq, Eq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Binding {
    pub parent_record_hash: String,
    pub parent_failure_signature: String,
    pub implementation_sha256: String,
    pub attempt_budget: u64,
    pub maximum_bytes: u64,
    pub original_predicate_identity: Value,
    #[serde(default)]
    pub ordering: Option<super::proposals::Ordering>,
}

#[derive(Clone, Debug, Deserialize, PartialEq, Eq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct ArtifactReference {
    pub directory: String,
    pub record_hash: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Checkpoint {
    format: String,
    pub binding: Binding,
    pub state: Value,
    pub accepted: Option<ArtifactReference>,
    pub published: Option<ArtifactReference>,
    pub complete: bool,
    pub sha256: String,
}

impl Checkpoint {
    fn digest(&self) -> Result<String, String> {
        Ok(canonical::digest(&canonical::encode(&serde_json::json!({
            "format": self.format,
            "binding": self.binding,
            "state": self.state,
            "accepted": self.accepted,
            "published": self.published,
            "complete": self.complete,
        }))?))
    }
}

pub(crate) struct Workspace {
    output: File,
    root: File,
    lock: File,
    accepted: File,
    path: PathBuf,
    checkpoint: Checkpoint,
    poisoned: bool,
}

impl Workspace {
    pub(crate) fn open(
        output: &Path,
        binding: Binding,
        initial_state: Value,
        resume: bool,
    ) -> Result<Self, String> {
        if !canonical::valid_digest(&binding.parent_record_hash)
            || !canonical::valid_digest(&binding.parent_failure_signature)
            || !canonical::valid_digest(&binding.implementation_sha256)
            || binding.attempt_budget == 0
            || binding.maximum_bytes == 0
            || binding.original_predicate_identity.is_null()
        {
            return Err("invalid minimizer checkpoint binding".into());
        }
        let output_root = open::pin(output)?;
        let name = format!(".minimize-{}", binding.parent_record_hash);
        match fs::mkdirat(&output_root, name.as_str(), Mode::from_raw_mode(0o700)) {
            Ok(()) => output_root.sync_all().map_err(|error| error.to_string())?,
            Err(error) if error == rustix::io::Errno::EXIST => {}
            Err(error) => return Err(error.to_string()),
        }
        let root = open::child(&output_root, &name, true, false)?;
        let lock = lock(&root)?;
        for entry in open::entries(&root)? {
            match entry.as_str() {
                ".lock" | "checkpoint.json" | "checkpoint.tmp" => {
                    open::child(&root, &entry, false, false)?;
                }
                "accepted" | "execution-observations" => {
                    open::child(&root, &entry, true, false)?;
                }
                _ => return Err(format!("unexpected minimizer workspace entry {entry:?}")),
            }
        }
        match fs::mkdirat(&root, "accepted", Mode::from_raw_mode(0o700)) {
            Ok(()) => root.sync_all().map_err(|error| error.to_string())?,
            Err(error) if error == rustix::io::Errno::EXIST => {}
            Err(error) => return Err(error.to_string()),
        }
        let accepted = open::child(&root, "accepted", true, false)?;
        let bytes = match open::read(&root, "checkpoint.json", MAX_CHECKPOINT_BYTES, false) {
            Ok(bytes) => Some(bytes),
            Err(error) if error == rustix::io::Errno::NOENT.to_string() => None,
            Err(error) => return Err(error),
        };
        let checkpoint = if let Some(bytes) = bytes {
            if !resume {
                return Err("minimizer checkpoint exists; resume is required".into());
            }
            let checkpoint: Checkpoint = serde_json::from_slice(&bytes)
                .map_err(|error| format!("decode minimizer checkpoint: {error}"))?;
            if checkpoint.format != FORMAT
                || canonical::encode(&checkpoint)? != bytes
                || checkpoint.sha256 != checkpoint.digest()?
            {
                return Err("minimizer checkpoint format or hash mismatch".into());
            }
            if checkpoint.binding != binding {
                return Err("minimizer checkpoint binding changed".into());
            }
            checkpoint
        } else {
            if resume {
                return Err("minimizer checkpoint is missing".into());
            }
            Checkpoint {
                format: FORMAT.into(),
                binding,
                state: initial_state,
                accepted: None,
                published: None,
                complete: false,
                sha256: String::new(),
            }
        };
        let mut workspace = Self {
            output: output_root,
            root,
            lock,
            accepted,
            path: output.join(name),
            checkpoint,
            poisoned: false,
        };
        workspace.authenticate(&workspace.checkpoint)?;
        if !resume {
            workspace.persist(workspace.checkpoint.clone())?;
        }
        Ok(workspace)
    }

    pub(crate) fn checkpoint(&self) -> &Checkpoint {
        &self.checkpoint
    }

    pub(crate) fn observation_root(&self) -> PathBuf {
        self.path.clone()
    }

    pub(crate) fn accepted_root(&self) -> PathBuf {
        self.path.join("accepted")
    }

    pub(crate) fn commit(
        &mut self,
        state: Value,
        accepted: Option<ArtifactReference>,
    ) -> Result<(), String> {
        if self.checkpoint.complete || self.checkpoint.published.is_some() {
            return Err("minimizer checkpoint no longer accepts proposals".into());
        }
        let mut next = self.checkpoint.clone();
        next.state = state;
        next.accepted = accepted;
        self.persist(next)
    }

    pub(crate) fn publication(&mut self, reference: ArtifactReference) -> Result<(), String> {
        if self.checkpoint.complete || self.checkpoint.published.is_some() {
            return Err("minimizer checkpoint already published".into());
        }
        let mut next = self.checkpoint.clone();
        next.published = Some(reference);
        self.persist(next)
    }

    pub(crate) fn complete(&mut self) -> Result<(), String> {
        let mut next = self.checkpoint.clone();
        next.complete = true;
        self.persist(next)
    }

    fn authenticate(&self, checkpoint: &Checkpoint) -> Result<(), String> {
        for (root, reference, publication) in [
            (&self.accepted, &checkpoint.accepted, false),
            (&self.output, &checkpoint.published, true),
        ] {
            let Some(reference) = reference else { continue };
            if open::components(&reference.directory)?.len() != 1
                || !canonical::valid_digest(&reference.record_hash)
            {
                return Err("unsafe minimizer artifact reference".into());
            }
            if open::read(root, ".store.json", 4096, false)? != open::LAYOUT {
                return Err("unsupported minimizer artifact store layout".into());
            }
            let pool = open::child(root, ".targets", true, false)?;
            let artifact = open::child(root, &reference.directory, true, false)?;
            let opened = Opened::from_root_with_pool(artifact, Some(&pool))?;
            if opened.manifest().record_sha256 != reference.record_hash {
                return Err("minimizer artifact record reference mismatch".into());
            }
            if publication && opened.stored_bytes() > checkpoint.binding.maximum_bytes {
                return Err("minimizer publication exceeds byte bound".into());
            }
            opened.authenticate_all()?;
        }
        Ok(())
    }

    fn persist(&mut self, mut next: Checkpoint) -> Result<(), String> {
        if self.poisoned {
            return Err("minimizer checkpoint writer is poisoned".into());
        }
        verify_named(&self.root, ".lock", &self.lock)?;
        if !self.checkpoint.sha256.is_empty()
            && open::read(&self.root, "checkpoint.json", MAX_CHECKPOINT_BYTES, false)?
                != canonical::encode(&self.checkpoint)?
        {
            return Err("minimizer checkpoint changed outside its writer".into());
        }
        self.authenticate(&next)?;
        next.sha256 = next.digest()?;
        let bytes = canonical::encode(&next)?;
        if bytes.len() as u64 > MAX_CHECKPOINT_BYTES {
            return Err("minimizer checkpoint exceeds byte bound".into());
        }
        match open::child(&self.root, "checkpoint.tmp", false, false) {
            Ok(_) => fs::unlinkat(&self.root, "checkpoint.tmp", AtFlags::empty())
                .map_err(|error| error.to_string())?,
            Err(error) if error == rustix::io::Errno::NOENT.to_string() => {}
            Err(error) => return Err(error),
        }
        self.poisoned = true;
        let mut file = create_file(&self.root, "checkpoint.tmp")?;
        file.write_all(&bytes)
            .and_then(|()| file.sync_all())
            .map_err(|error| error.to_string())?;
        verify_named(&self.root, "checkpoint.tmp", &file)?;
        verify_named(&self.root, ".lock", &self.lock)?;
        fs::renameat(&self.root, "checkpoint.tmp", &self.root, "checkpoint.json")
            .map_err(|error| error.to_string())?;
        self.root.sync_all().map_err(|error| error.to_string())?;
        verify_named(&self.root, "checkpoint.json", &file)?;
        self.checkpoint = next;
        self.poisoned = false;
        Ok(())
    }
}

fn create_file(root: &File, name: &str) -> Result<File, String> {
    fs::openat(
        root,
        name,
        OFlags::RDWR | OFlags::CREATE | OFlags::EXCL | OFlags::CLOEXEC | OFlags::NOFOLLOW,
        Mode::from_raw_mode(0o600),
    )
    .map(File::from)
    .map_err(|error| error.to_string())
}

fn lock(root: &File) -> Result<File, String> {
    let file = match create_file(root, ".lock") {
        Ok(file) => {
            file.sync_all().map_err(|error| error.to_string())?;
            root.sync_all().map_err(|error| error.to_string())?;
            file
        }
        Err(error) if error == rustix::io::Errno::EXIST.to_string() => {
            open::child(root, ".lock", false, false)?
        }
        Err(error) => return Err(error),
    };
    fs::flock(&file, FlockOperation::NonBlockingLockExclusive)
        .map_err(|error| format!("minimizer workspace lock unavailable: {error}"))?;
    verify_named(root, ".lock", &file)?;
    Ok(file)
}

fn verify_named(root: &File, name: &str, file: &File) -> Result<(), String> {
    let checked = open::child(root, name, false, false)?;
    if !open::same(
        &fs::fstat(file).map_err(|error| error.to_string())?,
        &fs::fstat(&checked).map_err(|error| error.to_string())?,
    ) {
        return Err("minimizer workspace entry substituted".into());
    }
    Ok(())
}
