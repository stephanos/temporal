//! Pinned Apalache admission and bounded source-projection controls.
mod decode;
mod replay;
pub use decode::{Itf, decode_itf};
pub use replay::{Confirmation, Replay, Replayed, ReplayedStep};

use std::{
    fs,
    path::{Path, PathBuf},
    process::Command,
    time::Duration,
};
use umpire_backend_process::{Limits, PinnedJar, Refusal, Scratch, java_command, run_process};
#[path = "../../../../dev/xtask/src/apalache_pins.rs"]
mod pins;
pub use pins::{ARCHIVE_SHA256, JAR_SHA256, VERSION};

// A Kettle property at its native bound takes about a minute alone; the gate runs several
// checks concurrently on a shared machine. A timeout only refuses, so headroom never admits.
const CHECK_TIMEOUT: Duration = Duration::from_secs(600);

/// A bounded symbolic answer, distinct from admission failure and native replay.
#[derive(Debug)]
pub enum Bounded {
    /// All executions were checked within the requested transition bound.
    NoViolation {
        /// Maximum transition depth checked.
        length: usize,
        /// Original tool output.
        output: String,
    },
    /// A raw typed-wire witness; task consumers must still decode and replay it.
    Violation {
        /// Requested maximum transition depth.
        length: usize,
        /// Original tool output.
        output: String,
        /// Raw ITF witnesses, pending typed replay.
        traces: Vec<String>,
    },
    /// No safety answer was established; selected checks must fail.
    NotRun {
        /// Requested maximum transition depth.
        length: usize,
        /// Execution failure or inconclusive-output diagnostic.
        reason: String,
        /// Bounded original tool output.
        output: String,
    },
}
impl Bounded {
    /// The requested maximum transition depth, including initial states at zero.
    pub fn length(&self) -> usize {
        match self {
            Self::NoViolation { length, .. }
            | Self::Violation { length, .. }
            | Self::NotRun { length, .. } => *length,
        }
    }
    /// Raw diagnostics, kept outside deterministic rendering.
    pub fn output(&self) -> &str {
        match self {
            Self::NoViolation { output, .. }
            | Self::Violation { output, .. }
            | Self::NotRun { output, .. } => output,
        }
    }
    /// Unconverted ITF witnesses in sorted relative-file order.
    pub fn traces(&self) -> &[String] {
        match self {
            Self::Violation { traces, .. } => traces,
            _ => &[],
        }
    }
    /// Require a completed safety answer, refusing `NotRun`.
    pub fn violated(&self) -> Result<bool, Refusal> {
        match self {
            Self::NoViolation { .. } => Ok(false),
            Self::Violation { .. } => Ok(true),
            Self::NotRun { reason, .. } => Err(Refusal(reason.clone())),
        }
    }
}
impl std::fmt::Display for Bounded {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let verdict = match self {
            Self::NoViolation { .. } => "NoViolation",
            Self::Violation { .. } => "Violation",
            Self::NotRun { .. } => "NotRun",
        };
        write!(f, "bound={} {verdict}", self.length())
    }
}

/// Derive the symbolic transition bound only from a completed native exploration.
pub fn native_bound(counts: umpire_checker::Counts) -> Result<usize, Refusal> {
    if counts.cut {
        return Err(Refusal("Apalache bound requires completed native exploration".into()));
    }
    let bound = counts
        .depth
        .checked_add(1)
        .ok_or_else(|| Refusal("Apalache native depth overflow".into()))?;
    validate_length(bound)?;
    Ok(bound)
}
fn validate_length(length: usize) -> Result<i32, Refusal> {
    i32::try_from(length)
        .map_err(|_| Refusal("Apalache length exceeds pinned tool integer range".into()))
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
        let limits = Limits { timeout: CHECK_TIMEOUT, ..Limits::default() };
        let jar = format!("apalache-{VERSION}/lib/apalache.jar");
        let PinnedJar { root, jar, java } =
            PinnedJar::admit(root, "model/build/apalache", &jar, JAR_SHA256, |tool, scratch| {
                let tool = Self {
                    root: tool.root.clone(),
                    jar: tool.jar.clone(),
                    java: tool.java.clone(),
                    limits,
                };
                let mut command = tool.command(scratch);
                command.arg("version");
                let (status, output) = run_process(&mut command, scratch, "version", limits, None)?;
                validate_version(status.success(), &output)
            })?;
        Ok(Self { root, jar, java, limits })
    }
    fn command(&self, work: &Path) -> Command {
        let mut command = java_command(&self.java);
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
    pub fn check(
        &self,
        module: &umpire_tla::Module,
        invariant: &str,
        length: usize,
    ) -> Result<Bounded, Refusal> {
        self.check_cancelled(module, invariant, length, &std::sync::atomic::AtomicBool::new(false))
    }

    /// Check with cooperative cancellation; the shared runner owns process-tree cleanup.
    #[allow(clippy::print_stderr)]
    pub fn check_cancelled(
        &self,
        module: &umpire_tla::Module,
        invariant: &str,
        length: usize,
        cancelled: &std::sync::atomic::AtomicBool,
    ) -> Result<Bounded, Refusal> {
        eprintln!("Apalache {} property={invariant} bound={length} start", module.name);
        let result = (|| {
            validate_length(length)?;
            umpire_tla::identifier(&module.name)?;
            umpire_tla::identifier(invariant)?;
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
            let checked = (|| {
                let file = scratch.path().join(format!("{}.tla", module.name));
                fs::write(&file, &module.tla)?;
                let cfg = scratch.path().join("model.cfg");
                fs::write(&cfg, &module.cfg)?;
                let out = scratch.path().join("out");
                let mut command = self.command(scratch.path());
                command
                    .arg(format!("--out-dir={}", out.display()))
                    .arg(format!("--run-dir={}", scratch.path().join("results").display()))
                    .arg("check")
                    .arg("--no-deadlock")
                    .arg(format!("--config={}", cfg.display()))
                    .arg(format!("--inv={invariant}"))
                    .arg(format!("--length={length}"))
                    .arg(&file);
                match umpire_backend_process::run_process_cancelled(
                    &mut command,
                    scratch.path(),
                    "check",
                    self.limits,
                    Some(scratch.path()),
                    cancelled,
                ) {
                    Ok((status, output)) => {
                        let mut traces = Vec::new();
                        match collect_itf(scratch.path(), &mut traces, self.limits.dump_bytes) {
                            Ok(()) => {
                                Ok(parse_output(status.code(), output, traces, invariant, length))
                            }
                            Err(error) => Ok(Bounded::NotRun { length, reason: error.0, output }),
                        }
                    }
                    Err(error) => Ok(Bounded::NotRun {
                        length,
                        reason: error.refusal.0,
                        output: error.output,
                    }),
                }
            })();
            scratch.finish(checked)
        })();
        match &result {
            Ok(Bounded::NotRun { .. }) | Err(_) => {
                eprintln!("Apalache {} property={invariant} bound={length} failure", module.name);
            }
            Ok(answer) => eprintln!(
                "Apalache {} property={invariant} bound={length} finish {answer}",
                module.name
            ),
        }
        result
    }
}
fn validate_version(success: bool, output: &str) -> Result<(), Refusal> {
    if !success || !output.lines().any(|line| line.trim() == VERSION) {
        return Err(Refusal(format!("Apalache version mismatch: {output}")));
    }
    Ok(())
}
fn parse_output(
    code: Option<i32>,
    output: String,
    traces: Vec<String>,
    property: &str,
    length: usize,
) -> Bounded {
    let lines: Vec<_> = output
        .lines()
        .map(|line| {
            [" I@", " W@", " E@"]
                .iter()
                .filter_map(|p| line.find(p))
                .min()
                .map_or(line, |at| &line[..at])
                .trim()
        })
        .collect();
    let has = |line: &str| lines.contains(&line);
    let outcomes: Vec<_> =
        lines.iter().filter_map(|line| line.strip_prefix("The outcome is: ")).collect();
    let selected = format!("  > Set an invariant to {property}");
    let header = lines
        .iter()
        .any(|line| line.starts_with(&format!("# APALACHE version: {VERSION} | build: ")));
    let setup = header && has(selected.trim()) && has("PASS #13: BoundedChecker");
    let completed = format!("Checker reports no error up to computation length {length}");
    let vacuous = outcomes == ["ExecutionsTooShort"]
        && has("All executions are shorter than the provided bound.")
        && lines
            .iter()
            .filter_map(|line| {
                line.strip_prefix("Checker reports no error up to computation length ")
            })
            .any(|n| n.parse::<usize>().is_ok_and(|n| n <= length));
    if setup
        && code == Some(0)
        && has("EXITCODE: OK")
        && traces.is_empty()
        && !lines.iter().any(|line| line.contains("state invariant 0 violated."))
        && ((outcomes == ["NoError"]
            && has(&completed)
            && has("State 0: state invariant 0 holds."))
            || vacuous)
    {
        Bounded::NoViolation { length, output }
    } else if setup
        && code == Some(12)
        && outcomes == ["Error"]
        && has("EXITCODE: ERROR (12)")
        && has("Checker has found an error")
        && lines
            .iter()
            .filter_map(|line| line.strip_prefix("State "))
            .filter_map(|line| line.strip_suffix(": state invariant 0 violated."))
            .any(|n| n.parse::<usize>().is_ok_and(|n| n <= length))
        && !traces.is_empty()
    {
        Bounded::Violation { length, output, traces }
    } else {
        Bounded::NotRun {
            length,
            reason: "inconclusive or malformed pinned Apalache outcome".into(),
            output,
        }
    }
}

fn collect_itf(path: &Path, out: &mut Vec<String>, limit: u64) -> Result<(), Refusal> {
    fn collect(
        path: &Path,
        out: &mut Vec<String>,
        remaining: &mut u64,
        entries: &mut usize,
        depth: usize,
    ) -> Result<(), Refusal> {
        if depth > 64 {
            return Err(Refusal("Apalache output nesting exceeds limit".into()));
        }
        let mut paths = Vec::new();
        for entry in fs::read_dir(path)? {
            if *entries == 0 {
                return Err(Refusal("Apalache output file count exceeds limit".into()));
            }
            *entries -= 1;
            paths.push(entry?.path());
        }
        paths.sort();
        for path in paths {
            let meta = fs::symlink_metadata(&path)?;
            if meta.is_symlink() {
                return Err(Refusal("symlink in Apalache output".into()));
            }
            if meta.is_dir() {
                collect(&path, out, remaining, entries, depth + 1)?;
            } else if !meta.is_file() {
                return Err(Refusal("non-regular Apalache output".into()));
            } else if path.file_name().is_some_and(|s| s.to_string_lossy().ends_with(".itf.json")) {
                let size = meta.len();
                *remaining = remaining
                    .checked_sub(size)
                    .ok_or_else(|| Refusal("Apalache aggregate trace limit exceeded".into()))?;
                let mut bytes = Vec::new();
                std::io::Read::read_to_end(
                    &mut std::io::Read::take(fs::File::open(path)?, size.saturating_add(1)),
                    &mut bytes,
                )?;
                if bytes.len() as u64 != size {
                    return Err(Refusal("Apalache trace changed during collection".into()));
                }
                out.push(String::from_utf8(bytes).map_err(|e| Refusal(e.to_string()))?);
            }
        }
        Ok(())
    }
    collect(path, out, &mut { limit }, &mut 4096, 0)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::PermissionsExt;

    fn fake(script: &str) -> Result<(Scratch, Apalache), Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
        let scratch = Scratch::new_in(&root, "model/build/apalache")?;
        let java = scratch.path().join("java");
        fs::write(&java, format!("#!/bin/sh\n{script}\n"))?;
        fs::set_permissions(&java, fs::Permissions::from_mode(0o700))?;
        let tool = Apalache {
            root: scratch.path().to_path_buf(),
            jar: scratch.path().join("fake.jar"),
            java,
            limits: Limits::default(),
        };
        Ok((scratch, tool))
    }

    #[test]
    fn successful_substring_is_not_a_bounded_proof() -> Result<(), Refusal> {
        let (scratch, tool) = fake("echo 'The outcome is: NoError'")?;
        let module = umpire_tla::Tla::safety(
            &umpire_tla::Lift::parse(
                "control.rs",
                include_str!("../../../dev/fixtures/fixtures/sample_kettle/model.rs"),
            )
            .map_err(|e| Refusal(e.to_string()))?,
        )
        .map_err(|e| Refusal(e.to_string()))?;
        let result = tool.check(&module, "ArithmeticSafe", 1);
        assert!(matches!(result?, Bounded::NotRun { .. }), "incomplete tool output was admitted");
        scratch.finish(Ok(()))
    }

    #[test]
    fn traces_have_one_aggregate_budget_and_deterministic_order() -> Result<(), Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
        let scratch = Scratch::new_in(&root, "model/build/apalache")?;
        fs::write(scratch.path().join("z.itf.json"), "zz")?;
        fs::write(scratch.path().join("a.itf.json"), "aa")?;
        assert!(collect_itf(scratch.path(), &mut Vec::new(), 3).is_err());
        let mut traces = Vec::new();
        collect_itf(scratch.path(), &mut traces, 4)?;
        assert_eq!(traces, ["aa", "zz"]);
        std::os::unix::fs::symlink("/", scratch.path().join("escape"))?;
        let error = collect_itf(scratch.path(), &mut Vec::new(), 1024)
            .err()
            .ok_or_else(|| Refusal("output symlink followed".into()))?;
        assert!(error.0.contains("symlink"));
        scratch.finish(Ok(()))
    }
    #[test]
    fn native_depth_is_checked_and_truncation_never_establishes_a_bound() -> Result<(), Refusal> {
        use umpire_checker::Counts;
        assert_eq!(native_bound(Counts { depth: 62, ..Counts::default() })?, 63);
        assert!(native_bound(Counts { cut: true, ..Counts::default() }).is_err());
        assert!(native_bound(Counts { depth: usize::MAX, ..Counts::default() }).is_err());
        assert!(native_bound(Counts { depth: i32::MAX as usize, ..Counts::default() }).is_err());
        Ok(())
    }

    fn success(length: usize) -> String {
        format!(
            "# APALACHE version: {VERSION} | build: 7954daf\n> Set an invariant to ArithmeticSafe\nPASS #13: BoundedChecker\nState 0: state invariant 0 holds.\nState {length}: state invariant 0 holds.\nThe outcome is: NoError\nChecker reports no error up to computation length {length}\nEXITCODE: OK\n"
        )
    }

    #[test]
    fn unchanged_monitor_can_omit_redundant_per_state_checks() {
        let output = success(1).replace("State 1: state invariant 0 holds.\n", "");
        assert!(matches!(
            parse_output(Some(0), output, vec![], "ArithmeticSafe", 1),
            Bounded::NoViolation { .. }
        ));
    }

    #[test]
    fn pinned_outcomes_require_consistent_property_bound_exit_and_completion() {
        let good = success(1);
        assert!(matches!(
            parse_output(Some(0), good.clone(), vec![], "ArithmeticSafe", 1),
            Bounded::NoViolation { .. }
        ));
        for (code, output, property, bound) in [
            (Some(1), good.clone(), "ArithmeticSafe", 1),
            (Some(0), good.clone(), "Other", 1),
            (Some(0), good.clone(), "ArithmeticSafe", 2),
            (Some(0), good.replace(VERSION, "wrong"), "ArithmeticSafe", 1),
            (Some(0), good.replace("PASS #13: BoundedChecker", ""), "ArithmeticSafe", 1),
            (Some(0), good.replace("EXITCODE: OK", ""), "ArithmeticSafe", 1),
            (Some(0), format!("{good}State 0: state invariant 0 violated.\n"), "ArithmeticSafe", 1),
            (Some(0), format!("{good}The outcome is: Error\n"), "ArithmeticSafe", 1),
        ] {
            let answer = parse_output(code, output, vec![], property, bound);
            assert!(matches!(answer, Bounded::NotRun { .. }), "{answer:?}");
            assert!(answer.violated().is_err());
        }
        let not_run = Bounded::NotRun {
            length: 1,
            reason: "/private/path 12.3s".into(),
            output: "raw diagnostics".into(),
        };
        assert_eq!(not_run.to_string(), "bound=1 NotRun");
    }

    #[test]
    fn fake_execution_failures_never_pass_and_scratch_is_removed() -> Result<(), Refusal> {
        let module = umpire_tla::Tla::safety(
            &umpire_tla::Lift::parse(
                "kettle.rs",
                include_str!("../../../dev/fixtures/fixtures/sample_kettle/model.rs"),
            )
            .map_err(|e| Refusal(e.to_string()))?,
        )
        .map_err(|e| Refusal(e.to_string()))?;
        for script in [
            "exit 2",
            "sleep 1",
            "printf 'too much output'",
            "mkdir out; ln -s / out/escape",
            "echo 'The outcome is: NoError'",
        ] {
            let (scratch, mut tool) = fake(script)?;
            tool.limits = Limits {
                timeout: Duration::from_millis(100),
                output_bytes: 8,
                ..Limits::default()
            };
            let answer = tool.check(&module, "ArithmeticSafe", 1)?;
            assert!(matches!(answer, Bounded::NotRun { .. }), "{script}: {answer:?}");
            let cache = scratch.path().join("model/build/apalache");
            assert_eq!(fs::read_dir(cache)?.count(), 0);
            assert!(tool.check(&module, "undeclared", 1).is_err());
            assert!(tool.check(&module, "ArithmeticSafe", usize::MAX).is_err());
            scratch.finish::<_, Refusal>(Ok(()))?;
        }
        let (scratch, tool) = fake("exit 0")?;
        fs::remove_file(&tool.java)?;
        assert!(matches!(tool.check(&module, "ArithmeticSafe", 1)?, Bounded::NotRun { .. }));
        scratch.finish(Ok(()))
    }
    #[test]
    fn version_admission_rejects_missing_wrong_and_failed_release_output() {
        for (success, output) in [(true, ""), (true, "0.62.3"), (false, VERSION)] {
            assert!(validate_version(success, output).is_err());
        }
        assert!(validate_version(true, VERSION).is_ok());
    }
}
