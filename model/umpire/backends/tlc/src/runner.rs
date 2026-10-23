use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    path::{Path, PathBuf},
    process::Command,
    time::Duration,
};

use crate::{Refusal, Schema, State, TraceState, decode_dump, decode_state};

/// Official stable release artifact admitted by this backend.
pub const JAR_VERSION: &str = "1.7.4";
/// SHA-256 of the official v1.7.4 release asset.
pub const JAR_SHA256: &str = "936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88";
const TLC_VERSION: &str = "Version 2.19 of 08 August 2024";
use crate::process::{Scratch, bounded_text, java_command, pinned_java, run_process};

/// Hard subprocess admission limits. A limit produces a refusal, never a verdict.
#[derive(Clone, Copy, Debug)]
pub struct Limits {
    /// Maximum wall time per Java process.
    pub timeout: Duration,
    /// Combined maximum stdout/stderr bytes per process and per check.
    pub output_bytes: u64,
    /// Maximum reachable dump size.
    pub dump_bytes: u64,
    /// Maximum Java heap in MiB.
    pub heap_mb: u32,
}

impl Default for Limits {
    fn default() -> Self {
        Self {
            timeout: Duration::from_secs(60),
            output_bytes: 8 * 1024 * 1024,
            dump_bytes: 128 * 1024 * 1024,
            heap_mb: 512,
        }
    }
}

/// An admitted property result.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Verdict {
    /// TLC completed this property check successfully.
    Verified,
    /// TLC reported the selected property's counterexample.
    Refuted,
}

/// A counterexample with a possible final back-edge to a zero-based trace index.
#[derive(Clone, Debug)]
pub struct Trace {
    /// All assignments, retaining private observers for real-edge replay.
    pub states: Vec<TraceState>,
    /// The destination of a lasso or final stuttering step.
    pub back_to: Option<usize>,
}

/// One selected property, checked independently of other violations.
#[derive(Clone, Debug)]
pub struct PropertyResult {
    /// The selected property's answer.
    pub verdict: Verdict,
    /// Concrete counterexample when refuted.
    pub trace: Option<Trace>,
}

/// Complete model reachability and separately checked property results.
#[derive(Clone, Debug)]
pub struct TlcResult {
    /// Deduplicated authored states after exact observer projection.
    pub states: BTreeSet<State>,
    /// Number of complete assignments before observer projection.
    pub raw_state_count: usize,
    /// Results keyed by the selected TLA+ operator name.
    pub properties: BTreeMap<String, PropertyResult>,
    /// Raw tool output, retained as execution evidence.
    pub output: String,
}

/// A pinned jar and the Java executable resolved from this repository's mise pin.
#[derive(Clone, Debug)]
pub struct Tlc {
    root: PathBuf,
    jar: PathBuf,
    java: PathBuf,
    /// Bounds applied to every subprocess.
    pub limits: Limits,
}

impl Tlc {
    /// Explore an emitted module and check all registered claims without a native reference graph.
    /// The returned state set belongs to the complete unrestricted specification; scoped progress
    /// claims use their own observer specifications and retain typed counterexamples.
    pub fn check_module(&self, module: &umpire_lift::Module) -> Result<TlcResult, Refusal> {
        let schema = crate::module::schema(module)?;
        let mut result = self.check(&module.name, &module.tla, &module.cfg, &schema)?;
        for cfg in module
            .metadata
            .progress
            .iter()
            .map(|claim| &claim.cfg)
            .chain(module.metadata.negative.iter().map(|claim| &claim.cfg))
        {
            let additional = self.check(&module.name, &module.tla, cfg, &schema)?;
            for (operator, property) in additional.properties {
                if result.properties.insert(operator.clone(), property).is_some() {
                    return Err(Refusal(format!("duplicate property result {operator}")));
                }
            }
            if u64::try_from(result.output.len().saturating_add(additional.output.len()))
                .unwrap_or(u64::MAX)
                > self.limits.output_bytes
            {
                return Err(Refusal("combined module output limit exceeded".into()));
            }
            result.output.push_str(&additional.output);
        }
        Ok(result)
    }

    /// Admit the exact jar and mise-pinned Temurin JDK before any model check.
    pub fn discover(root: &Path) -> Result<Self, Refusal> {
        let root = root.canonicalize()?;
        let jar = root.join(format!("model/build/tlc/tla2tools-{JAR_VERSION}.jar"));
        if !jar.is_file() {
            return Err(Refusal(format!(
                "missing pinned jar {}; run bash model/scripts/fetch-tlc.sh",
                jar.display()
            )));
        }
        let jar = jar.canonicalize()?;
        let cache = root.join("model/build/tlc").canonicalize()?;
        if !cache.starts_with(&root) || !jar.starts_with(&cache) {
            return Err(Refusal("pinned jar escapes the repository cache".into()));
        }
        let scratch = Scratch::new(&root)?;
        let admitted = (|| {
            let mut digest = Command::new("shasum");
            digest.args(["-a", "256"]).arg(&jar).current_dir(&root);
            let (status, output) =
                run_process(&mut digest, &scratch.0, "digest", Limits::default(), None)?;
            if !status.success() || output.split_whitespace().next() != Some(JAR_SHA256) {
                return Err(Refusal(format!(
                    "pinned jar integrity/version mismatch: {}",
                    jar.display()
                )));
            }
            let java = pinned_java(&root, &scratch.0)?;
            let mut version = java_command(&java);
            version.args(["-cp"]).arg(&jar).args(["tlc2.TLC", "-h"]).current_dir(&root);
            let (status, output) =
                run_process(&mut version, &scratch.0, "tlc-version", Limits::default(), None)?;
            if status.code() != Some(1) || !output.contains(TLC_VERSION) {
                return Err(Refusal(format!("wrong TLC version: {output}")));
            }
            Ok(Self { root, jar, java, limits: Limits::default() })
        })();
        scratch.finish(admitted)
    }

    /// Explore without claims first, then independently check every inline PROPERTY/INVARIANT.
    pub fn check(
        &self,
        name: &str,
        tla: &str,
        cfg: &str,
        schema: &Schema,
    ) -> Result<TlcResult, Refusal> {
        schema.validate()?;
        if name.is_empty() || !name.bytes().all(|c| c.is_ascii_alphanumeric() || c == b'_') {
            return Err(Refusal("module name must be one TLA+ identifier".into()));
        }
        if self.limits.timeout.is_zero()
            || self.limits.output_bytes == 0
            || self.limits.dump_bytes == 0
            || self.limits.heap_mb == 0
        {
            return Err(Refusal("subprocess limits must be positive".into()));
        }
        let (base_cfg, properties) = split_cfg(cfg)?;
        let scratch = Scratch::new(&self.root)?;
        let result = (|| {
            fs::write(scratch.0.join(format!("{name}.tla")), tla)?;
            let (output, dump) = self.execute(name, &base_cfg, &scratch.0, "explore", None)?;
            let exploration_messages = messages(&output)?;
            let count = distinct_count(&exploration_messages)?;
            let states = decode_dump(&dump, count, schema)?;
            let mut result = TlcResult {
                states: states.into_iter().map(|state| state.state).collect(),
                raw_state_count: count,
                properties: BTreeMap::new(),
                output,
            };
            for (kind, property) in properties {
                let property_cfg = format!("{base_cfg}\n{kind} {property}\n");
                let (output, _) =
                    self.execute(name, &property_cfg, &scratch.0, &property, Some(&property))?;
                let messages = messages(&output)?;
                let violated = messages
                    .iter()
                    .any(|(code, _)| matches!(*code, 2107 | 2108 | 2110 | 2112 | 2116));
                let trace = if violated { Some(decode_trace(&messages, schema)?) } else { None };
                result.properties.insert(
                    property,
                    PropertyResult {
                        verdict: if violated { Verdict::Refuted } else { Verdict::Verified },
                        trace,
                    },
                );
                if u64::try_from(result.output.len().saturating_add(output.len()))
                    .unwrap_or(u64::MAX)
                    > self.limits.output_bytes
                {
                    return Err(Refusal("combined check output limit exceeded".into()));
                }
                result.output.push_str(&output);
            }
            Ok(result)
        })();
        scratch.finish(result)
    }

    fn execute(
        &self,
        name: &str,
        cfg: &str,
        directory: &Path,
        label: &str,
        property: Option<&str>,
    ) -> Result<(String, String), Refusal> {
        let config = directory.join("Run.cfg");
        fs::write(&config, cfg)?;
        let dump = directory.join(format!("{label}.dump"));
        let mut command = java_command(&self.java);
        command
            .arg(format!("-Xmx{}m", self.limits.heap_mb))
            .arg(format!("-Duser.home={}", directory.display()))
            .arg(format!("-Djava.io.tmpdir={}", directory.display()))
            .args([
                "-XX:+UseSerialGC",
                "-Dtlc2.tool.fp.FPSet.impl=tlc2.tool.fp.MSBDiskFPSet",
                "-cp",
            ])
            .arg(&self.jar)
            .args(["tlc2.TLC", "-tool", "-workers", "1", "-cleanup", "-checkpoint", "0", "-config"])
            .arg(&config)
            .arg("-metadir")
            .arg(directory.join("states"))
            .arg("-dump")
            .arg(&dump)
            .arg(name)
            .current_dir(directory);
        let (status, output) =
            run_process(&mut command, directory, label, self.limits, Some(&dump))?;
        let messages = messages(&output).map_err(|error| Refusal(format!("{error}\n{output}")))?;
        let tla = fs::read_to_string(directory.join(format!("{name}.tla")))?;
        let violation = selected_violation(&messages, property, &tla)?;
        let finished = messages.iter().any(|(code, _)| *code == 2186);
        let success = messages.iter().any(|(code, _)| *code == 2193);
        let unexpected_error = messages.iter().any(|(code, _)| {
            (2100..=2179).contains(code)
                && !matches!(*code, 2107 | 2108 | 2110 | 2112 | 2116 | 2121 | 2122)
        });
        if !finished
            || unexpected_error
            || !(status.success() || property.is_some() && violation)
            || (status.success() && !success)
        {
            return Err(Refusal(format!(
                "process failed or incomplete for {} (exit {status}):\n{output}",
                property.unwrap_or("reachability")
            )));
        }
        if violation && property.is_none() {
            return Err(Refusal(format!("reachability unexpectedly refuted:\n{output}")));
        }
        Ok((output, bounded_text(&dump, self.limits.dump_bytes)?))
    }
}

fn selected_violation(
    messages: &[(u32, String)],
    property: Option<&str>,
    tla: &str,
) -> Result<bool, Refusal> {
    let mut violated = false;
    for (code, body) in messages {
        let prefix = match code {
            2107 | 2110 => "Invariant ",
            2108 => "Property ",
            2112 => "Action property ",
            2116 => {
                if property.is_none() {
                    return Err(Refusal(
                        "unexpected liveness violation during full exploration".into(),
                    ));
                }
                violated = true;
                continue;
            }
            _ => continue,
        };
        let actual = body.strip_prefix(prefix).and_then(|rest| {
            // A state-level `[]P` conjunct of a PROPERTY is checked as an invariant and named
            // by its source span rather than by the operator.
            located(rest, tla).or_else(|| rest.split_whitespace().next())
        });
        if actual != property || property.is_none() {
            return Err(Refusal(format!("unexpected property violation: {body}")));
        }
        violated = true;
    }
    Ok(violated)
}

/// The single-line top-level operator containing a span reported as
/// `line L, col C to line L2, col C2 of module M is violated.`
fn located<'a>(body: &str, tla: &'a str) -> Option<&'a str> {
    let rest = body.strip_prefix("line ")?;
    let (first, rest) = rest.split_once(", col ")?;
    let (_, rest) = rest.split_once(" to line ")?;
    let (last, _) = rest.split_once(", col ")?;
    let (first, last) = (first.parse::<usize>().ok()?, last.parse::<usize>().ok()?);
    if first != last {
        return None;
    }
    let line = tla.lines().nth(first.checked_sub(1)?)?;
    let (operator, _) = line.split_once(" == ")?;
    operator.bytes().all(|c| c.is_ascii_alphanumeric() || c == b'_').then_some(operator)
}

pub(super) fn split_cfg(cfg: &str) -> Result<(String, Vec<(String, String)>), Refusal> {
    let mut base = String::new();
    let mut properties = Vec::new();
    let mut names = BTreeSet::new();
    for line in cfg.lines() {
        let words: Vec<_> = line.split_whitespace().collect();
        if words.first().is_some_and(|word| matches!(*word, "INVARIANT" | "PROPERTY")) {
            if words.len() != 2
                || !words[1].bytes().all(|c| c.is_ascii_alphanumeric() || c == b'_')
                || !names.insert(words[1].to_owned())
            {
                return Err(Refusal(format!(
                    "expected one unique property operator per config line: {line}"
                )));
            }
            properties.push((words[0].to_owned(), words[1].to_owned()));
        } else {
            if words.first().is_some_and(|word| {
                matches!(
                    *word,
                    "INVARIANTS"
                        | "PROPERTIES"
                        | "CONSTRAINT"
                        | "CONSTRAINTS"
                        | "ACTION_CONSTRAINT"
                        | "ACTION_CONSTRAINTS"
                        | "VIEW"
                        | "SYMMETRY"
                )
            }) || (words.first() == Some(&"CHECK_DEADLOCK") && words.get(1) != Some(&"TRUE"))
            {
                return Err(Refusal(format!(
                    "config would alter complete reachability or deadlock admission: {line}"
                )));
            }
            base.push_str(line);
            base.push('\n');
        }
    }
    Ok((base, properties))
}

fn messages(output: &str) -> Result<Vec<(u32, String)>, Refusal> {
    let mut messages = Vec::new();
    let mut remaining = output;
    while let Some(start) = remaining.find("@!@!@STARTMSG ") {
        remaining = &remaining[start + "@!@!@STARTMSG ".len()..];
        let (header, rest) = remaining
            .split_once(" @!@!@\n")
            .ok_or_else(|| Refusal("truncated TLC message header".into()))?;
        let code = header
            .split(':')
            .next()
            .and_then(|code| code.parse().ok())
            .ok_or_else(|| Refusal("invalid TLC message code".into()))?;
        let (body, rest) = rest
            .split_once("@!@!@ENDMSG ")
            .ok_or_else(|| Refusal("truncated TLC message body".into()))?;
        let (end, rest) = rest
            .split_once(" @!@!@")
            .ok_or_else(|| Refusal("truncated TLC message terminator".into()))?;
        if end.parse::<u32>().ok() != Some(code) {
            return Err(Refusal("mismatched TLC message terminator".into()));
        }
        if header.split(':').nth(1) == Some("1")
            && !matches!(code, 2107 | 2108 | 2110 | 2112 | 2116 | 2121 | 2122 | 2264)
        {
            return Err(Refusal(format!("TLC execution error {code}: {body}")));
        }
        messages.push((code, body.trim().to_owned()));
        remaining = rest;
    }
    Ok(messages)
}

fn distinct_count(messages: &[(u32, String)]) -> Result<usize, Refusal> {
    let body = messages
        .iter()
        .rev()
        .find(|(code, _)| *code == 2199)
        .map(|(_, body)| body)
        .ok_or_else(|| Refusal("missing final TLC state statistics".into()))?;
    if !body.ends_with("0 states left on queue.") {
        return Err(Refusal(format!("TLC exploration queue was not exhausted: {body}")));
    }
    body.split("states generated,")
        .nth(1)
        .and_then(|rest| rest.split_whitespace().next())
        .and_then(|count| count.replace(',', "").parse().ok())
        .ok_or_else(|| Refusal(format!("invalid final TLC state statistics: {body}")))
}

fn decode_trace(messages: &[(u32, String)], schema: &Schema) -> Result<Trace, Refusal> {
    let mut trace = Trace { states: Vec::new(), back_to: None };
    for (code, body) in messages {
        match code {
            2216 | 2217 => {
                let (header, assignment) = body
                    .split_once('\n')
                    .ok_or_else(|| Refusal("counterexample state has no assignment".into()))?;
                if header.split(':').next().and_then(|number| number.parse::<usize>().ok())
                    != Some(trace.states.len() + 1)
                {
                    return Err(Refusal("nonconsecutive counterexample state numbers".into()));
                }
                trace.states.push(decode_state(assignment, schema)?);
            }
            2122 => {
                let index = body
                    .split_whitespace()
                    .find_map(|word| {
                        word.trim_matches(|c: char| !c.is_ascii_digit()).parse::<usize>().ok()
                    })
                    .ok_or_else(|| Refusal(format!("invalid back-edge: {body}")))?;
                trace.back_to = index.checked_sub(1);
            }
            2218 => trace.back_to = trace.states.len().checked_sub(1),
            _ => {}
        }
    }
    if trace.states.is_empty()
        && let Some((_, body)) = messages.iter().find(|(code, _)| matches!(*code, 2107 | 2108))
    {
        let (_, assignment) = body
            .split_once('\n')
            .ok_or_else(|| Refusal("initial violation has no state assignment".into()))?;
        trace.states.push(decode_state(assignment, schema)?);
    }
    if trace.states.is_empty() || trace.back_to.is_some_and(|index| index >= trace.states.len()) {
        return Err(Refusal("missing or invalid concrete counterexample trace".into()));
    }
    Ok(trace)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{Type, Value};

    #[test]
    fn located_property_invariants_name_only_the_selected_operator() -> Result<(), Refusal> {
        let tla =
            "---- MODULE M ----\nOther == TRUE\nProgress_p == (A => B) /\\ [](x <= 4)\n====\n";
        let located = |line: usize| {
            vec![(
                2110,
                format!(
                    "Invariant line {line}, col 30 to line {line}, col 41 of module M is violated."
                ),
            )]
        };
        assert!(selected_violation(&located(3), Some("Progress_p"), tla)?);
        for (messages, property) in [
            (located(2), Some("Progress_p")),
            (located(3), Some("Other")),
            (located(3), None),
            (
                vec![(
                    2110,
                    "Invariant line 3, col 1 to line 4, col 2 of module M is violated.".into(),
                )],
                Some("Progress_p"),
            ),
        ] {
            assert!(selected_violation(&messages, property, tla).is_err());
        }
        Ok(())
    }

    #[test]
    fn typed_module_refuses_inconsistent_declarations_before_spawning()
    -> Result<(), Box<dyn std::error::Error>> {
        let module = umpire_lift::Tla::emit(&umpire_lift::Lift::parse(
            "scaling.rs",
            include_str!("../tests/fixtures/scaling_model.rs"),
        )?)?;
        let tlc = Tlc {
            root: PathBuf::from("/unused"),
            jar: PathBuf::from("/unused"),
            java: PathBuf::from("/missing-java"),
            limits: Limits::default(),
        };
        let mut invalid = module.clone();
        invalid.metadata.fields.push(invalid.metadata.fields[0].clone());
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.0.contains("duplicate authored field"))
        );
        let mut invalid = module.clone();
        invalid.metadata.observer_fields.push(invalid.metadata.observer_fields[0].clone());
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.0.contains("duplicate observer"))
        );
        let mut invalid = module.clone();
        invalid.metadata.observer_types.insert("undeclared".into(), umpire_lift::Type::Bool);
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.0.contains("observer declarations differ"))
        );
        let mut invalid = module.clone();
        invalid.metadata.fields[0].name = "__umpire_edge_tick".into();
        assert!(tlc.check_module(&invalid).err().is_some_and(|error| error.0.contains("collides")));
        let mut invalid = module.clone();
        invalid.metadata.fields[0].ty = umpire_lift::Type::Int { min: 3, max: 2 };
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.0.contains("integer bounds"))
        );
        let mut invalid = module.clone();
        invalid.metadata.fields[0].ty =
            umpire_lift::Type::Enum { name: "Empty".into(), variants: Vec::new() };
        assert!(
            tlc.check_module(&invalid).err().is_some_and(|error| error.0.contains("enum variants"))
        );
        let mut invalid = module.clone();
        invalid.metadata.invariants.push(invalid.metadata.invariants[0].clone());
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.0.contains("duplicate property"))
        );
        let mut invalid = module;
        invalid.cfg = "SPECIFICATION Spec\n".into();
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.0.contains("property declarations differ"))
        );
        Ok(())
    }

    #[test]
    fn counterexample_requires_consecutive_states_and_a_valid_lasso() {
        let schema = Schema {
            fields: BTreeMap::from([("x".into(), Type::Bool)]),
            observers: BTreeMap::new(),
        };
        let state = |number| (2217, format!("{number}: <Initial predicate>\nx = TRUE"));
        assert!(decode_trace(&[state(2)], &schema).is_err());
        assert!(decode_trace(&[state(1), state(3)], &schema).is_err());
        assert!(decode_trace(&[state(1), (2122, "Back to state 2".into())], &schema).is_err());
        let trace = decode_trace(&[state(1), (2218, "2: Stuttering".into())], &schema)
            .unwrap_or_else(|error| panic!("{error}"));
        assert_eq!(trace.back_to, Some(0));
        assert_eq!(trace.states[0].state["x"], Value::Bool(true));
    }

    #[test]
    fn incomplete_messages_and_exploration_are_refused() {
        for text in [
            "@!@!@STARTMSG 2193:0 @!@!@\n",
            "@!@!@STARTMSG 2193:0 @!@!@\nOK\n@!@!@ENDMSG 2186 @!@!@",
            "@!@!@STARTMSG 1000:1 @!@!@\ncrash\n@!@!@ENDMSG 1000 @!@!@",
        ] {
            assert!(messages(text).is_err(), "accepted {text}");
        }
        assert!(
            distinct_count(&[(
                2199,
                "3 states generated, 2 distinct states found, 1 states left on queue.".into()
            )])
            .is_err()
        );
        for cfg in [
            "CHECK_DEADLOCK FALSE",
            "CONSTRAINT Cut",
            "PROPERTY a b",
            "PROPERTY a\nPROPERTY a",
            "VIEW Projection",
        ] {
            assert!(split_cfg(cfg).is_err(), "accepted {cfg}");
        }
    }

    #[test]
    #[cfg(unix)]
    fn escaping_cache_symlinks_are_refused_before_creating_files() -> Result<(), Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        let result = (|| {
            let fake = scratch.0.join("repo");
            let outside = scratch.0.join("outside");
            fs::create_dir_all(fake.join("model"))?;
            fs::create_dir(&outside)?;
            std::os::unix::fs::symlink(&outside, fake.join("model/build"))?;
            assert!(Scratch::new(&fake).is_err());
            assert!(
                !outside.join("tlc").exists(),
                "escaping cache wrote outside the root before admission"
            );
            let scripts = fake.join("model/scripts");
            fs::create_dir(&scripts)?;
            let script = scripts.join("fetch-tlc.sh");
            fs::copy(root.join("model/scripts/fetch-tlc.sh"), &script)?;
            let mut fetch = Command::new("bash");
            fetch.arg(&script).current_dir(&fake);
            let (status, output) =
                run_process(&mut fetch, &scratch.0, "fetch-symlink", Limits::default(), None)?;
            assert!(!status.success());
            assert!(output.contains("TLC cache escapes repository"), "{output}");
            assert!(
                !outside.join("tlc").exists(),
                "fetch script wrote through an escaping symlink"
            );
            Ok(())
        })();
        scratch.finish(result)
    }

    #[test]
    fn process_limits_kill_and_reap_and_jar_admission_fails_closed() -> Result<(), Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        let result = (|| {
            let error = Tlc::discover(&scratch.0)
                .err()
                .ok_or_else(|| Refusal("missing jar admitted".into()))?;
            assert!(error.0.contains("missing pinned jar"));
            fs::create_dir_all(scratch.0.join("model/build/tlc"))?;
            fs::write(scratch.0.join("model/build/tlc/tla2tools-1.7.4.jar"), b"wrong version")?;
            fs::write(scratch.0.join("mise.toml"), "java = \"temurin-27\"\n")?;
            let error = Tlc::discover(&scratch.0)
                .err()
                .ok_or_else(|| Refusal("wrong jar admitted".into()))?;
            assert!(error.0.contains("integrity/version mismatch"));
            let mut command = Command::new("sleep");
            command.arg("2");
            let limits = Limits { timeout: Duration::from_millis(5), ..Limits::default() };
            assert!(run_process(&mut command, &scratch.0, "timeout", limits, None).is_err());
            let mut command = Command::new("printf");
            command.arg("too much output");
            let limits = Limits { output_bytes: 2, ..Limits::default() };
            assert!(run_process(&mut command, &scratch.0, "output", limits, None).is_err());
            let dump = scratch.0.join("oversized.dump");
            fs::write(&dump, "excess")?;
            let mut command = Command::new("sleep");
            command.arg("2");
            let limits = Limits { dump_bytes: 2, ..Limits::default() };
            assert!(run_process(&mut command, &scratch.0, "dump", limits, Some(&dump)).is_err());
            Ok(())
        })();
        scratch.finish(result)
    }
}
