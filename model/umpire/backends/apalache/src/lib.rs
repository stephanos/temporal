//! Pinned Apalache admission and bounded source-projection controls.
use std::{
    fs,
    path::{Path, PathBuf},
    process::Command,
    time::Duration,
};
use umpire_tlc::{
    Limits, Refusal,
    process::{Scratch, run_process},
};
/// Selected official release.
pub const VERSION: &str = "0.62.4";
/// SHA-256 of the versioned official release archive.
pub const ARCHIVE_SHA256: &str = "b781b53bba32fac14ac2e02b0332e1dc2edc64cb7b850c0608d8142cc7aa2ef4";
/// SHA-256 of its unmodified assembly jar.
pub const JAR_SHA256: &str = "e5223915507b65cc5f1638bd037ede60f8a4fc1644aff46355779f9b05df9262";
/// A bounded answer; tool errors return a refusal rather than an answer.
#[derive(Debug)]
pub struct Bounded {
    /// Whether the selected invariant was violated within the bound.
    pub violated: bool,
    /// Maximum Model transitions checked, with initial states at depth zero.
    pub length: usize,
    /// Raw output retained separately from deterministic fixture rendering.
    pub output: String,
    /// Original ITF JSON witnesses, for typed replay integration.
    pub traces: Vec<String>,
}
/// An admitted release and mise-selected Java, resolved under the workspace build root.
pub struct Apalache {
    root: PathBuf,
    jar: PathBuf,
    java: PathBuf,
    /// Per-process hard limits.
    pub limits: Limits,
}
impl Apalache {
    /// Check the exact installed release before invoking a Model.
    pub fn discover(root: &Path) -> Result<Self, Refusal> {
        let root = root.canonicalize()?;
        let scratch = Scratch::new_in(&root, "model/build/apalache")?;
        let result = (|| {
            let jar = root
                .join(format!("model/build/apalache/apalache-{VERSION}/lib/apalache.jar"))
                .canonicalize()?;
            let cache = root.join("model/build/apalache").canonicalize()?;
            if !cache.starts_with(&root) || !jar.starts_with(&cache) {
                return Err(Refusal("Apalache jar escapes workspace build root".into()));
            }
            let mut digest = Command::new("shasum");
            digest.args(["-a", "256"]).arg(&jar).current_dir(&scratch.0);
            let (status, output) =
                run_process(&mut digest, &scratch.0, "digest", Limits::default(), None)?;
            if !status.success() || output.split_whitespace().next() != Some(JAR_SHA256) {
                return Err(Refusal("Apalache integrity mismatch".into()));
            }
            let mut resolve = Command::new("mise");
            resolve.args(["which", "java"]).current_dir(&root);
            let (status, output) =
                run_process(&mut resolve, &scratch.0, "mise", Limits::default(), None)?;
            let java = PathBuf::from(output.trim());
            if !status.success() || !java.is_absolute() || !java.is_file() {
                return Err(Refusal(format!("Apalache Java resolution failed: {output}")));
            }
            let tool = Self {
                root: root.clone(),
                jar,
                java,
                limits: Limits { timeout: Duration::from_secs(120), ..Limits::default() },
            };
            let mut command = tool.command(&scratch.0);
            command.arg("version");
            let (status, output) =
                run_process(&mut command, &scratch.0, "version", tool.limits, None)?;
            if !status.success() || !output.lines().any(|line| line.trim() == VERSION) {
                return Err(Refusal(format!("Apalache version mismatch: {output}")));
            }
            Ok(tool)
        })();
        scratch.finish(result)
    }
    fn command(&self, work: &Path) -> Command {
        let mut command = Command::new(&self.java);
        command
            .arg(format!("-Xmx{}m", self.limits.heap_mb))
            .arg(format!("-Duser.home={}", work.display()))
            .arg(format!("-Djava.io.tmpdir={}", work.display()))
            .arg("-jar")
            .arg(&self.jar)
            .current_dir(work);
        command
    }
    /// Check one declared property, preserving the original ITF witness before cleanup.
    #[allow(clippy::print_stderr)]
    pub fn check(
        &self,
        module: &umpire_lift::Module,
        invariant: &str,
        length: usize,
    ) -> Result<Bounded, Refusal> {
        if !module
            .metadata
            .invariants
            .iter()
            .chain(&module.metadata.postconditions)
            .chain(&module.metadata.negative)
            .chain(&module.metadata.arithmetic)
            .any(|p| p.operator == invariant)
        {
            return Err(Refusal(format!("undeclared Apalache safety property {invariant}")));
        }
        let scratch = Scratch::new_in(&self.root, "model/build/apalache")?;
        let result = (|| {
            let file = scratch.0.join(format!("{}.tla", module.name));
            fs::write(&file, &module.tla)?;
            let cfg = scratch.0.join("model.cfg");
            fs::write(&cfg, &module.cfg)?;
            eprintln!("Apalache {} property={invariant} bound={length} start", module.name);
            let mut command = self.command(&scratch.0);
            command
                .arg(format!("--out-dir={}", scratch.0.join("out").display()))
                .arg(format!("--run-dir={}", scratch.0.join("results").display()))
                .arg("check")
                .arg(format!("--config={}", cfg.display()))
                .arg(format!("--inv={invariant}"))
                .arg(format!("--length={length}"))
                .arg(&file);
            let (status, output) =
                run_process(&mut command, &scratch.0, "check", self.limits, None)?;
            let violated = status.code() == Some(12)
                && output.contains("state invariant 0 violated.")
                && output.contains("violated");
            if !violated && (!status.success() || !output.contains("The outcome is: NoError")) {
                return Err(Refusal(format!("Apalache NotRun: {output}")));
            }
            let mut traces = Vec::new();
            collect_itf(&scratch.0, &mut traces, self.limits.dump_bytes)?;
            if violated && traces.is_empty() {
                return Err(Refusal("Apalache violation has no ITF witness".into()));
            }
            eprintln!(
                "Apalache {} property={invariant} bound={length} finish violated={violated}",
                module.name
            );
            Ok(Bounded { violated, length, output, traces })
        })();
        scratch.finish(result)
    }
}
fn collect_itf(path: &Path, out: &mut Vec<String>, limit: u64) -> Result<(), Refusal> {
    for entry in fs::read_dir(path)? {
        let entry = entry?;
        let ty = entry.file_type()?;
        if ty.is_symlink() {
            return Err(Refusal("symlink in Apalache output".into()));
        }
        if ty.is_dir() {
            collect_itf(&entry.path(), out, limit)?;
        } else if entry.file_name().to_string_lossy().ends_with(".itf.json") {
            if entry.metadata()?.len() > limit {
                return Err(Refusal("Apalache trace exceeds limit".into()));
            }
            out.push(fs::read_to_string(entry.path())?);
        }
    }
    Ok(())
}
