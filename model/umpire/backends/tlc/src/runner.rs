use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    path::{Path, PathBuf},
};

use crate::{
    Error, Evaluation, Limits, Refusal, Schema, State, TraceState, decode_dump, decode_state,
};

/// Official stable release artifact admitted by this backend.
pub const JAR_VERSION: &str = "1.7.4";
/// SHA-256 of the official v1.7.4 release asset.
pub const JAR_SHA256: &str = "936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88";
const TLC_VERSION: &str = "Version 2.19 of 08 August 2024";
const CACHE: &str = "build/tlc";
use crate::cache::{self, Bundle, Record};
use std::sync::{Mutex, OnceLock};
use umpire_backend_process::{
    PinnedJar, Scratch, bounded_text, bytes_sha256, java_command, run_process, structural_sha256,
};

/// An admitted property result.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Verdict {
    /// TLC completed this property check successfully.
    Verified,
    /// TLC reported the selected property's counterexample.
    Refuted,
}

/// A counterexample with a possible final back-edge to a zero-based trace index.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Trace {
    /// All assignments, retaining private observers for real-edge replay.
    pub states: Vec<TraceState>,
    /// The destination of a lasso or final stuttering step.
    pub back_to: Option<usize>,
}

/// One selected property, checked independently of other violations.
#[derive(Clone, Debug, PartialEq, Eq)]
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
    /// Maximum simultaneous singleton recovery processes within one check. Must be positive.
    /// The default is at most two; gates must also bound concurrent test processes and invocations.
    pub property_workers: usize,
}

impl Tlc {
    /// Explore an emitted module and check all registered claims without a native reference graph.
    /// The returned state set belongs to the complete unrestricted specification; scoped progress
    /// claims use their own observer specifications and retain typed counterexamples.
    ///
    /// # Errors
    /// A refusal, or TLC's typed evaluation error naming the property and state it failed in.
    pub fn check_module(&self, module: &umpire_tla::Module) -> Result<TlcResult, Error> {
        let schema = crate::module::schema(module)?;
        let inventory: Vec<_> = std::iter::once(module.cfg.as_str())
            .chain(module.metadata.progress.iter().map(|claim| claim.cfg.as_str()))
            .chain(module.metadata.negative.iter().map(|claim| claim.cfg.as_str()))
            .collect();
        let mut pending = Vec::new();
        let identity = OnceLock::new();
        let mut result = self.check_inventory(
            &module.name,
            &module.tla,
            &module.cfg,
            &schema,
            &inventory,
            0,
            &identity,
            &mut pending,
        )?;
        for (index, cfg) in module
            .metadata
            .progress
            .iter()
            .map(|claim| &claim.cfg)
            .chain(module.metadata.negative.iter().map(|claim| &claim.cfg))
            .enumerate()
        {
            let additional = self.check_inventory(
                &module.name,
                &module.tla,
                cfg,
                &schema,
                &inventory,
                index + 1,
                &identity,
                &mut pending,
            )?;
            for (operator, property) in additional.properties {
                if result.properties.insert(operator.clone(), property).is_some() {
                    return Err(Refusal(format!("duplicate property result {operator}")).into());
                }
            }
            if u64::try_from(result.output.len().saturating_add(additional.output.len()))
                .unwrap_or(u64::MAX)
                > self.limits.output_bytes
            {
                return Err(Refusal("combined module output limit exceeded".into()).into());
            }
            result.output.push_str(&additional.output);
        }
        for bundle in pending {
            let _ = bundle.publish();
        }
        Ok(result)
    }

    /// Admit the exact jar and mise-pinned Temurin JDK before any model check.
    pub fn discover(root: &Path) -> Result<Self, Refusal> {
        let jar = format!("tla2tools-{JAR_VERSION}.jar");
        PinnedJar::admit(root, CACHE, &jar, JAR_SHA256, |PinnedJar { root, jar, java }, scratch| {
            let limits = Limits::default();
            let mut version = java_command(&java);
            version
                .arg(format!("-Xmx{}m", limits.heap_mb))
                .arg("-cp")
                .arg(&jar)
                .args(["tlc2.TLC", "-h"])
                .current_dir(scratch);
            let (status, output) = run_process(&mut version, scratch, "tlc-version", limits, None)?;
            if status.code() != Some(1) || !output.contains(TLC_VERSION) {
                return Err(Refusal(format!("wrong TLC version: {output}")));
            }
            Ok(Self {
                root,
                jar,
                java,
                limits,
                property_workers: std::thread::available_parallelism()
                    .map_or(1, |n| n.get().min(2)),
            })
        })
        .map_err(|Refusal(message)| {
            if message.starts_with("missing pinned jar") {
                Refusal(format!("{message}; run bash scripts/fetch-tlc.sh"))
            } else {
                Refusal(message)
            }
        })
    }

    /// Explore without claims first, then batch inline properties, recovering singleton traces on failure.
    ///
    /// # Errors
    /// A refusal, or TLC's typed evaluation error naming the property and state it failed in.
    pub fn check(
        &self,
        name: &str,
        tla: &str,
        cfg: &str,
        schema: &Schema,
    ) -> Result<TlcResult, Error> {
        let mut pending = Vec::new();
        let result =
            self.check_inventory(name, tla, cfg, schema, &[cfg], 0, &OnceLock::new(), &mut pending);
        if result.as_ref().is_ok_and(all_verified) {
            for bundle in pending {
                let _ = bundle.publish();
            }
        }
        result
    }

    #[allow(clippy::too_many_arguments)]
    fn check_inventory(
        &self,
        name: &str,
        tla: &str,
        cfg: &str,
        schema: &Schema,
        inventory: &[&str],
        index: usize,
        identity: &OnceLock<Result<(String, String), Refusal>>,
        pending: &mut Vec<Bundle>,
    ) -> Result<TlcResult, Error> {
        // Admission precedes identity work and cache access on both paths.
        schema.validate()?;
        umpire_tla::identifier(name)?;
        let (base, properties) = split_cfg(cfg)?;
        let mut bundle = if cache::enabled(
            std::env::var("UMPIRE_QUICK_CACHE").ok().as_deref(),
            std::env::var("UMPIRE_CACHE_DISABLE").ok().as_deref(),
        ) {
            match self.cache_bundle(
                name,
                tla,
                cfg,
                schema,
                inventory,
                index,
                &base,
                &properties,
                identity,
            ) {
                Ok(bundle) => Some(bundle),
                Err(error) => {
                    cache_report(format_args!(
                        "TLC cache miss {name} configuration {index}: {error}"
                    ));
                    None
                }
            }
        } else {
            None
        };
        if let Some(bundle) = bundle.as_mut() {
            if bundle.load().is_ok() {
                let result = self.check_strategy_cached(name, tla, cfg, schema, true, Some(bundle));
                if result.as_ref().is_ok_and(all_verified) {
                    cache_report(format_args!(
                        "TLC cache hit {name} configuration {index} {}",
                        bundle.key
                    ));
                    return result;
                }
                bundle.replay = false;
                bundle.records = Mutex::new(BTreeMap::new());
            }
            cache_report(format_args!(
                "TLC cache miss {name} configuration {index} {}",
                bundle.key
            ));
        }
        let result = self.check_strategy_cached(name, tla, cfg, schema, true, bundle.as_ref());
        if result.as_ref().is_ok_and(all_verified)
            && let Some(bundle) = bundle
        {
            pending.push(bundle);
        }
        #[cfg(feature = "strategy-equivalence")]
        {
            let independent = self.check_strategy(name, tla, cfg, schema, false);
            assert!(
                equivalent(&result, &independent),
                "TLC strategy equivalence failed for {name}:\nbatched: {result:?}\nindependent: {independent:?}"
            );
        }
        result
    }

    #[allow(clippy::too_many_arguments)]
    fn cache_bundle(
        &self,
        name: &str,
        tla: &str,
        cfg: &str,
        schema: &Schema,
        inventory: &[&str],
        index: usize,
        base: &str,
        properties: &[(String, String)],
        identity: &OnceLock<Result<(String, String), Refusal>>,
    ) -> Result<Bundle, Refusal> {
        let root = cache::root()?;
        let (jar, java) = identity
            .get_or_init(|| self.cache_identity(&root))
            .as_ref()
            .map_err(|error| Refusal(error.0.clone()))?;
        let (key, schema) = self.cache_key(name, tla, cfg, schema, inventory, index, jar, java);
        let mut expected = vec![cache::record_id(base)];
        if !properties.is_empty() {
            expected.push(cache::record_id(&property_cfg(base, properties)));
        }
        Ok(Bundle {
            key,
            schema,
            root,
            expected,
            records: Mutex::new(BTreeMap::new()),
            replay: false,
            limits: self.limits,
        })
    }

    fn cache_identity(&self, root: &Path) -> Result<(String, String), Refusal> {
        let scratch = Scratch::new_in(root, "quick-tlc-v1")?;
        let identity = (|| {
            let jar = bytes_sha256(&fs::read(&self.jar)?);
            let mut command = java_command(&self.java);
            command.arg("-version");
            let (status, java) =
                run_process(&mut command, scratch.path(), "identity", self.limits, None)?;
            if !status.success() {
                return Err(Refusal("Java cache identity unavailable".into()));
            }
            Ok((jar, java))
        })();
        scratch.finish(identity)
    }

    #[allow(clippy::too_many_arguments)]
    fn cache_key(
        &self,
        name: &str,
        tla: &str,
        cfg: &str,
        schema: &Schema,
        inventory: &[&str],
        index: usize,
        jar: &str,
        java: &str,
    ) -> (String, String) {
        let schema = structural_sha256(&(&schema.fields, &schema.observers));
        let key = structural_sha256(&(
            cache::VERSION,
            name,
            tla,
            inventory,
            index,
            cfg,
            &schema,
            jar,
            java,
            &self.java,
            (
                self.limits.timeout,
                self.limits.output_bytes,
                self.limits.dump_bytes,
                self.limits.heap_mb,
                self.property_workers,
            ),
            (std::env::consts::ARCH, std::env::consts::OS, usize::BITS),
        ));
        (key, schema)
    }

    #[cfg(any(test, feature = "strategy-equivalence"))]
    fn check_strategy(
        &self,
        name: &str,
        tla: &str,
        cfg: &str,
        schema: &Schema,
        batch: bool,
    ) -> Result<TlcResult, Error> {
        self.check_strategy_cached(name, tla, cfg, schema, batch, None)
    }

    #[allow(clippy::too_many_arguments)]
    fn check_strategy_cached(
        &self,
        name: &str,
        tla: &str,
        cfg: &str,
        schema: &Schema,
        batch: bool,
        cache: Option<&Bundle>,
    ) -> Result<TlcResult, Error> {
        schema.validate()?;
        umpire_tla::identifier(name)?;
        if self.limits.timeout.is_zero()
            || self.limits.output_bytes == 0
            || self.limits.dump_bytes == 0
            || self.limits.heap_mb == 0
            || self.property_workers == 0
        {
            return Err(Refusal("subprocess limits must be positive".into()).into());
        }
        let (base_cfg, properties) = split_cfg(cfg)?;
        let (output, dump) =
            self.execution(name, tla, &base_cfg, Selection::Explore, schema, cache)?;
        let count = distinct_count(&messages(&output)?)?;
        let states = decode_dump(&dump, count, schema)?;
        let mut result = TlcResult {
            states: states.into_iter().map(|state| state.state).collect(),
            raw_state_count: count,
            properties: BTreeMap::new(),
            output,
        };
        if batch && properties.len() > 1 {
            let cfg = property_cfg(&base_cfg, &properties);
            match self.execution(name, tla, &cfg, Selection::Batch(&properties), schema, cache) {
                Ok((output, _)) => {
                    let violated =
                        messages(&output)?.iter().any(|(code, _)| VIOLATIONS.contains(code));
                    append_output(&mut result.output, &output, self.limits.output_bytes)?;
                    if violated && cache.is_some_and(|cache| cache.replay) {
                        return Err(Refusal("cached property batch is not Verified".into()).into());
                    }
                    if !violated {
                        for (_, property) in properties {
                            result.properties.insert(
                                property,
                                PropertyResult { verdict: Verdict::Verified, trace: None },
                            );
                        }
                        return Ok(result);
                    }
                }
                Err(error @ Error::Evaluation(_)) => {
                    if cache.is_some_and(|cache| cache.replay) {
                        return Err(error);
                    }
                }
                Err(error) => return Err(error),
            }
        }
        if batch
            && properties.len() > 1
            && let Some(cache) = cache
        {
            // Recovery is evidence of a failed batch, even if every singleton later passes.
            cache.records.lock().map_err(|_| Refusal("cache poisoned".into()))?.clear();
        }
        let workers = if batch { self.property_workers } else { 1 };
        for wave in properties.chunks(workers) {
            let checked = if wave.len() == 1 {
                vec![self.check_property_cached(name, tla, &base_cfg, &wave[0], schema, cache)]
            } else {
                self.recover_properties(name, tla, &base_cfg, wave, schema)?
            };
            for ((_, property), checked) in wave.iter().zip(checked) {
                let (answer, output) = checked?;
                append_output(&mut result.output, &output, self.limits.output_bytes)?;
                result.properties.insert(property.clone(), answer);
            }
        }
        Ok(result)
    }

    fn check_property(
        &self,
        name: &str,
        tla: &str,
        base: &str,
        property: &(String, String),
        schema: &Schema,
    ) -> Result<(PropertyResult, String), Error> {
        self.check_property_cached(name, tla, base, property, schema, None)
    }

    #[allow(clippy::too_many_arguments)]
    fn check_property_cached(
        &self,
        name: &str,
        tla: &str,
        base: &str,
        property: &(String, String),
        schema: &Schema,
        cache: Option<&Bundle>,
    ) -> Result<(PropertyResult, String), Error> {
        let cfg = property_cfg(base, std::slice::from_ref(property));
        let (output, _) =
            self.execution(name, tla, &cfg, Selection::Property(&property.1), schema, cache)?;
        let messages = messages(&output)?;
        let violated = messages.iter().any(|(code, _)| VIOLATIONS.contains(code));
        let trace = if violated { Some(decode_trace(&messages, schema)?) } else { None };
        Ok((
            PropertyResult {
                verdict: if violated { Verdict::Refuted } else { Verdict::Verified },
                trace,
            },
            output,
        ))
    }

    fn recover_properties(
        &self,
        name: &str,
        tla: &str,
        base: &str,
        properties: &[(String, String)],
        schema: &Schema,
    ) -> Result<Vec<Result<(PropertyResult, String), Error>>, Error> {
        assert!(properties.len() <= self.property_workers);
        std::thread::scope(|scope| {
            let mut handles = Vec::new();
            for property in properties {
                handles.push(std::thread::Builder::new().spawn_scoped(scope, move || {
                    self.check_property(name, tla, base, property, schema)
                })?);
            }
            handles
                .into_iter()
                .map(|handle| {
                    handle.join().map_err(|_| Refusal("property worker panicked".into()).into())
                })
                .collect()
        })
    }

    #[allow(clippy::too_many_arguments)]
    fn execution(
        &self,
        name: &str,
        tla: &str,
        cfg: &str,
        selection: Selection<'_>,
        schema: &Schema,
        cache: Option<&Bundle>,
    ) -> Result<(String, String), Error> {
        let id = cache::record_id(cfg);
        if let Some(cache) = cache
            && cache.replay
        {
            let records = cache.records.lock().map_err(|_| Refusal("cache poisoned".into()))?;
            let record =
                records.get(&id).ok_or_else(|| Refusal("cache execution missing".into()))?;
            self.admit_execution(tla, selection, schema, true, "cached success", &record.output)?;
            return Ok((record.output.clone(), record.dump.clone()));
        }
        if std::env::var("UMPIRE_QUICK_CACHE").as_deref() == Ok("1")
            || std::env::var("UMPIRE_CACHE_DISABLE").as_deref() == Ok("1")
        {
            cache_report(format_args!("TLC execute {name} record {id}"));
        }
        let scratch = Scratch::new(&self.root)?;
        let result = (|| {
            fs::write(scratch.path().join(format!("{name}.tla")), tla)?;
            self.execute(name, cfg, scratch.path(), selection, schema)
        })();
        let result = scratch.finish(result);
        if let Some(cache) = cache
            && let Ok((output, dump)) = &result
        {
            cache
                .records
                .lock()
                .map_err(|_| Refusal("cache poisoned".into()))?
                .insert(id, Record { output: output.clone(), dump: dump.clone() });
        }
        result
    }

    fn execute(
        &self,
        name: &str,
        cfg: &str,
        directory: &Path,
        selection: Selection<'_>,
        schema: &Schema,
    ) -> Result<(String, String), Error> {
        let label = match selection {
            Selection::Explore => "explore",
            Selection::Property(property) => property,
            Selection::Batch(_) => "batch",
        };
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
            .args([
                "tlc2.TLC",
                "-tool",
                "-workers",
                "1",
                "-fp",
                "0",
                "-cleanup",
                "-checkpoint",
                "0",
                "-config",
            ])
            .arg(&config)
            .arg("-metadir")
            .arg(directory.join("states"))
            .arg("-dump")
            .arg(&dump)
            .arg(name)
            .current_dir(directory);
        let (status, output) =
            run_process(&mut command, directory, label, self.limits, Some(&dump))?;
        let tla = fs::read_to_string(directory.join(format!("{name}.tla")))?;
        self.admit_execution(
            &tla,
            selection,
            schema,
            status.success(),
            &status.to_string(),
            &output,
        )?;
        Ok((output, bounded_text(&dump, self.limits.dump_bytes)?))
    }
    #[allow(clippy::too_many_arguments)]
    fn admit_execution(
        &self,
        tla: &str,
        selection: Selection<'_>,
        schema: &Schema,
        success_status: bool,
        status: &str,
        output: &str,
    ) -> Result<(), Error> {
        let property = match selection {
            Selection::Property(property) => Some(property),
            _ => None,
        };
        if output.len() as u64 > self.limits.output_bytes {
            return Err(Refusal("cached output limit".into()).into());
        }
        let parsed = parse(output).map_err(|error| Refusal(format!("{error}\n{output}")))?;
        if matches!(selection, Selection::Batch(_))
            && !parsed.iter().any(|message| message.code == 2186)
        {
            return Err(Refusal(format!("incomplete TLC property batch:\n{output}")).into());
        }
        failure(&parsed, property, tla, schema)?;
        let messages: Vec<_> =
            parsed.into_iter().map(|message| (message.code, message.body)).collect();
        let violation = match selection {
            Selection::Batch(properties) => batch_violation(&messages, properties, tla)?,
            _ => selected_violation(&messages, property, tla)?,
        };
        let finished = messages.iter().any(|(code, _)| *code == 2186);
        let success = messages.iter().any(|(code, _)| *code == 2193);
        let unexpected_error = messages.iter().any(|(code, _)| {
            (2100..=2179).contains(code)
                && !matches!(*code, 2107 | 2108 | 2110 | 2112 | 2116 | 2121 | 2122)
        });
        if !finished
            || unexpected_error
            || !(success_status || !matches!(selection, Selection::Explore) && violation)
            || (success_status && !success)
        {
            return Err(Refusal(format!(
                "process failed or incomplete for {} (exit {status}):\n{output}",
                property.unwrap_or("reachability")
            ))
            .into());
        }
        if violation && matches!(selection, Selection::Explore) {
            return Err(Refusal(format!("reachability unexpectedly refuted:\n{output}")).into());
        }
        Ok(())
    }
}

fn cache_report(message: std::fmt::Arguments<'_>) {
    use std::io::Write;
    let _ = writeln!(std::io::stderr(), "{message}");
}

fn all_verified(result: &TlcResult) -> bool {
    result
        .properties
        .values()
        .all(|property| property.verdict == Verdict::Verified && property.trace.is_none())
}

#[derive(Clone, Copy)]
enum Selection<'a> {
    Explore,
    Property(&'a str),
    Batch(&'a [(String, String)]),
}

fn property_cfg(base: &str, properties: &[(String, String)]) -> String {
    let mut cfg = base.to_owned();
    for (kind, property) in properties {
        cfg.push('\n');
        cfg.push_str(kind);
        cfg.push(' ');
        cfg.push_str(property);
        cfg.push('\n');
    }
    cfg
}

fn append_output(output: &mut String, additional: &str, limit: u64) -> Result<(), Refusal> {
    if u64::try_from(output.len().saturating_add(additional.len())).unwrap_or(u64::MAX) > limit {
        return Err(Refusal("combined check output limit exceeded".into()));
    }
    output.push_str(additional);
    Ok(())
}

fn batch_violation(
    messages: &[(u32, String)],
    properties: &[(String, String)],
    tla: &str,
) -> Result<bool, Refusal> {
    let mut violated = false;
    for message in messages.iter().filter(|(code, _)| VIOLATIONS.contains(code)) {
        let selected = properties.iter().any(|(kind, property)| {
            (message.0 != 2116 || kind == "PROPERTY")
                && selected_violation(std::slice::from_ref(message), Some(property), tla)
                    .is_ok_and(|violated| violated)
        });
        if !selected {
            return Err(Refusal(format!("unexpected batch property violation: {}", message.1)));
        }
        violated = true;
    }
    Ok(violated)
}

#[cfg(feature = "strategy-equivalence")]
fn equivalent(actual: &Result<TlcResult, Error>, independent: &Result<TlcResult, Error>) -> bool {
    match (actual, independent) {
        (Ok(actual), Ok(independent)) => {
            actual.states == independent.states
                && actual.raw_state_count == independent.raw_state_count
                && actual.properties == independent.properties
        }
        (Err(Error::Evaluation(actual)), Err(Error::Evaluation(independent))) => {
            actual.property == independent.property
                && actual.message == independent.message
                && actual.trace == independent.trace
                && actual.state == independent.state
        }
        (Err(Error::Refusal(_)), Err(Error::Refusal(_))) => true,
        _ => false,
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

const START: &str = "@!@!@STARTMSG ";
const END: &str = "@!@!@ENDMSG ";
/// The violation and behavior messages TLC marks as errors.
const VIOLATIONS: [u32; 5] = [2107, 2108, 2110, 2112, 2116];
const REPORTED: [u32; 9] = [2103, 2107, 2108, 2110, 2112, 2116, 2121, 2122, 2264];

/// One TLC message: its code, whether TLC marks it an error, and its text, followed by the text
/// of each message nested in it.
struct Message {
    code: u32,
    error: bool,
    body: String,
}

fn messages(output: &str) -> Result<Vec<(u32, String)>, Refusal> {
    Ok(parse(output)?.into_iter().map(|message| (message.code, message.body)).collect())
}

fn parse(output: &str) -> Result<Vec<Message>, Refusal> {
    let mut messages = Vec::new();
    let mut remaining = output;
    while let Some(start) = remaining.find(START) {
        let (message, rest) = message(&remaining[start + START.len()..], 0)?;
        messages.push(message);
        remaining = rest;
    }
    Ok(messages)
}

/// The message after a `STARTMSG` marker, and the output after its terminator. TLC nests the
/// cause of a failed evaluation, such as an integer overflow, inside the failure's report.
fn message(input: &str, depth: usize) -> Result<(Message, &str), Refusal> {
    if depth > 4 {
        return Err(Refusal("TLC messages nest too deeply".into()));
    }
    let (header, mut rest) = input
        .split_once(" @!@!@\n")
        .ok_or_else(|| Refusal("truncated TLC message header".into()))?;
    let mut fields = header.split(':');
    let code = fields
        .next()
        .and_then(|code| code.parse().ok())
        .ok_or_else(|| Refusal("invalid TLC message code".into()))?;
    let error = fields.next() == Some("1");
    let mut body = String::new();
    loop {
        let end = rest.find(END).ok_or_else(|| Refusal("truncated TLC message body".into()))?;
        match rest.find(START) {
            Some(start) if start < end => {
                body.push_str(&rest[..start]);
                let (nested, after) = message(&rest[start + START.len()..], depth + 1)?;
                body.push_str(&nested.body);
                body.push('\n');
                rest = after;
            }
            _ => {
                body.push_str(&rest[..end]);
                rest = &rest[end + END.len()..];
                break;
            }
        }
    }
    let (end, rest) = rest
        .split_once(" @!@!@")
        .ok_or_else(|| Refusal("truncated TLC message terminator".into()))?;
    if end.parse::<u32>().ok() != Some(code) {
        return Err(Refusal("mismatched TLC message terminator".into()));
    }
    Ok((Message { code, error, body: body.trim().to_owned() }, rest))
}

/// TLC's first error other than a violation and its behavior. TLC names the property whose
/// evaluation failed, or locates the failing expression, for a failed evaluation; that becomes
/// a typed [`Evaluation`] with the behavior up to the failing state. Any other error refuses.
/// A run checks one `property` only after exploration evaluated the initial predicate and
/// next-state relation, so a failure TLC leaves unnamed in that run is the property's.
fn failure(
    messages: &[Message],
    property: Option<&str>,
    tla: &str,
    schema: &Schema,
) -> Result<(), Error> {
    let Some(first) =
        messages.iter().find(|message| message.error && !REPORTED.contains(&message.code))
    else {
        return Ok(());
    };
    let positions = messages.iter().find(|message| message.code == 2103);
    let prefix = match first.code {
        2111 => Some("Evaluating invariant "),
        2113 => Some("Evaluating action property "),
        _ => None,
    };
    if prefix.is_none() && positions.is_none() {
        return Err(Refusal(format!("TLC execution error {}: {}", first.code, first.body)).into());
    }
    let (head, cause) = first.body.split_once('\n').unwrap_or((&first.body, ""));
    let named = prefix.and_then(|prefix| {
        let rest = head.strip_prefix(prefix)?;
        located(rest, tla).or_else(|| rest.strip_suffix(" failed."))
    });
    let mut message = if named.is_some() { cause.trim() } else { first.body.as_str() }.to_owned();
    let property = named.or(property).map(str::to_owned);
    if let Some(positions) = positions {
        message.push('\n');
        message.push_str(&positions.body);
    }
    let states: Vec<_> = messages
        .iter()
        .filter(|message| matches!(message.code, 2216 | 2217))
        .map(|message| (message.code, message.body.clone()))
        .collect();
    let (trace, state) = if states.is_empty() {
        (None, None)
    } else {
        let state = states
            .last()
            .and_then(|(_, body)| body.split_once('\n'))
            .map(|(_, assignment)| assignment.trim().to_owned());
        (Some(decode_trace(&states, schema)?), state)
    };
    Err(Box::new(Evaluation { property, message, trace, state }).into())
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
    use crate::Value;
    use std::{process::Command, time::Duration};
    use umpire_tla::Type;

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
        let module = umpire_tla::Tla::emit(&umpire_tla::Lift::parse(
            "scaling.rs",
            include_str!("../tests/fixtures/scaling_model.rs"),
        )?)?;
        let tlc = Tlc {
            root: PathBuf::from("/unused"),
            jar: PathBuf::from("/unused"),
            java: PathBuf::from("/missing-java"),
            limits: Limits::default(),
            property_workers: 2,
        };
        let mut invalid = module.clone();
        invalid.metadata.fields.push(invalid.metadata.fields[0].clone());
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.to_string().contains("duplicate authored field"))
        );
        let mut invalid = module.clone();
        invalid.metadata.observer_fields.push(invalid.metadata.observer_fields[0].clone());
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.to_string().contains("duplicate observer"))
        );
        let mut invalid = module.clone();
        invalid.metadata.observer_types.insert("undeclared".into(), Type::Bool);
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.to_string().contains("observer declarations differ"))
        );
        let mut invalid = module.clone();
        invalid.metadata.fields[0].name = umpire_tla::observers::EDGE_TICK.into();
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.to_string().contains("collides"))
        );
        let mut invalid = module.clone();
        invalid.metadata.fields[0].ty = Type::Int { min: 3, max: 2 };
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.to_string().contains("integer bounds"))
        );
        let mut invalid = module.clone();
        invalid.metadata.fields[0].ty = Type::Enum { name: "Empty".into(), variants: Vec::new() };
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.to_string().contains("enum variants"))
        );
        let mut invalid = module.clone();
        invalid.metadata.invariants.push(invalid.metadata.invariants[0].clone());
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.to_string().contains("duplicate property"))
        );
        let mut invalid = module;
        invalid.cfg = "SPECIFICATION Spec\n".into();
        assert!(
            tlc.check_module(&invalid)
                .err()
                .is_some_and(|error| error.to_string().contains("property declarations differ"))
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
            "@!@!@STARTMSG 2111:1 @!@!@\nfailed\n@!@!@STARTMSG 2178:0 @!@!@\nOverflow\n@!@!@ENDMSG 2111 @!@!@",
        ] {
            assert!(messages(text).is_err(), "accepted {text}");
        }
        let crash = parse("@!@!@STARTMSG 1000:1 @!@!@\ncrash\n@!@!@ENDMSG 1000 @!@!@");
        assert!(crash.is_ok_and(|crash| matches!(
            failure(&crash, None, "", &Schema::default()),
            Err(Error::Refusal(Refusal(text))) if text == "TLC execution error 1000: crash"
        )));
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

    /// TLC's own reports, as recorded from TLC 2.19: the cause nests inside the failure.
    #[test]
    fn evaluation_errors_name_the_property_and_state() -> Result<(), Refusal> {
        let schema = Schema {
            fields: BTreeMap::from([("x".into(), Type::Int { min: 0, max: 3 })]),
            observers: BTreeMap::new(),
        };
        let state = |number: usize, x: usize| {
            let action = if number == 1 { "Initial predicate" } else { "Next line 5, col 9" };
            format!(
                "@!@!@STARTMSG 2217:4 @!@!@\n{number}: <{action}>\nx = {x}\n\n@!@!@ENDMSG 2217 @!@!@\n"
            )
        };
        let behavior = "@!@!@STARTMSG 2121:1 @!@!@\nThe behavior up to this point is:\n@!@!@ENDMSG 2121 @!@!@\n";
        let positions = "@!@!@STARTMSG 2103:1 @!@!@\nThe error occurred when TLC was evaluating the nested\nexpressions at the following positions:\n0. Line 4, column 9 to line 4, column 26 in P\n\n@!@!@ENDMSG 2103 @!@!@\n";
        let invariant = format!(
            "@!@!@STARTMSG 2111:1 @!@!@\nEvaluating invariant Wide failed.\n@!@!@STARTMSG 2178:0 @!@!@\nOverflow when computing 3*1000000000\n@!@!@ENDMSG 2178 @!@!@\n@!@!@ENDMSG 2111 @!@!@\n{behavior}{}{}{}{}",
            state(1, 0),
            state(2, 1),
            state(3, 2),
            state(4, 3)
        );
        let assert = format!(
            "@!@!@STARTMSG 2132:1 @!@!@\nThe first argument of Assert evaluated to FALSE; the second argument was:\n\"checked Rust integer overflow\"\n@!@!@ENDMSG 2132 @!@!@\n{behavior}{}{}{positions}",
            state(1, 0),
            state(2, 1)
        );
        let initial = format!(
            "@!@!@STARTMSG 1000:1 @!@!@\n@!@!@STARTMSG 2178:0 @!@!@\nOverflow when computing 3*1000000000\n@!@!@ENDMSG 2178 @!@!@\n@!@!@ENDMSG 1000 @!@!@\n{positions}"
        );
        let evaluation = |output: &str| match failure(&parse(output)?, None, "", &schema) {
            Err(Error::Evaluation(evaluation)) => Ok(evaluation),
            other => Err(Refusal(format!("not an evaluation error: {other:?}"))),
        };
        let wide = evaluation(&invariant)?;
        assert_eq!(wide.property.as_deref(), Some("Wide"));
        assert_eq!(wide.message, "Overflow when computing 3*1000000000");
        assert_eq!(wide.state.as_deref(), Some("x = 3"));
        assert_eq!(wide.trace.as_ref().map(|trace| trace.states.len()), Some(4));
        assert_eq!(
            wide.to_string(),
            "TLC evaluation error in property Wide: Overflow when computing 3*1000000000\n\
             at state 4 of the behavior:\nx = 3"
        );
        let next = evaluation(&assert)?;
        assert_eq!(next.property, None);
        assert!(next.message.contains("checked Rust integer overflow"), "{next}");
        assert!(next.message.contains("Line 4, column 9"), "{next}");
        assert!(next.to_string().starts_with("TLC evaluation error in the next-state relation"));
        assert_eq!(next.state.as_deref(), Some("x = 1"));
        let unnamed = failure(&parse(&initial)?, Some("Checked"), "", &schema);
        assert!(matches!(
            unnamed,
            Err(Error::Evaluation(evaluation)) if evaluation.property.as_deref() == Some("Checked")
                && evaluation.to_string().ends_with("in an initial state")
        ));
        let init = evaluation(&initial)?;
        assert!(init.trace.is_none() && init.state.is_none());
        assert!(init.to_string().starts_with(
            "TLC evaluation error in the initial predicate: Overflow when computing 3*1000000000"
        ));
        Ok(())
    }

    #[test]
    #[cfg(unix)]
    fn escaping_cache_symlinks_are_refused_before_creating_files() -> Result<(), Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        let result = (|| {
            let fake = scratch.path().join("repo");
            let outside = scratch.path().join("outside");
            fs::create_dir_all(&fake)?;
            fs::create_dir(&outside)?;
            std::os::unix::fs::symlink(&outside, fake.join("build"))?;
            assert!(Scratch::new(&fake).is_err());
            assert!(
                !outside.join("tlc").exists(),
                "escaping cache wrote outside the root before admission"
            );
            let scripts = fake.join("scripts");
            fs::create_dir(&scripts)?;
            let script = scripts.join("fetch-tlc.sh");
            fs::copy(root.join("scripts/fetch-tlc.sh"), &script)?;
            let mut fetch = Command::new("bash");
            fetch.arg(&script).current_dir(&fake);
            let (status, output) =
                run_process(&mut fetch, scratch.path(), "fetch-symlink", Limits::default(), None)?;
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
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        let result = (|| {
            let error = Tlc::discover(scratch.path())
                .err()
                .ok_or_else(|| Refusal("missing jar admitted".into()))?;
            assert!(error.0.contains("missing pinned jar"));
            fs::create_dir_all(scratch.path().join("build/tlc"))?;
            fs::write(scratch.path().join("build/tlc/tla2tools-1.7.4.jar"), b"wrong version")?;
            fs::write(scratch.path().join("mise.toml"), "java = \"temurin-27\"\n")?;
            let error = Tlc::discover(scratch.path())
                .err()
                .ok_or_else(|| Refusal("wrong jar admitted".into()))?;
            assert!(error.0.contains("integrity mismatch"));
            let mut command = Command::new("sleep");
            command.arg("2");
            let limits = Limits { timeout: Duration::from_millis(5), ..Limits::default() };
            assert!(run_process(&mut command, scratch.path(), "timeout", limits, None).is_err());
            let mut command = Command::new("printf");
            command.arg("too much output");
            let limits = Limits { output_bytes: 2, ..Limits::default() };
            assert!(run_process(&mut command, scratch.path(), "output", limits, None).is_err());
            let dump = scratch.path().join("oversized.dump");
            fs::write(&dump, "excess")?;
            let mut command = Command::new("sleep");
            command.arg("2");
            let limits = Limits { dump_bytes: 2, ..Limits::default() };
            assert!(
                run_process(&mut command, scratch.path(), "dump", limits, Some(&dump)).is_err()
            );
            Ok(())
        })();
        scratch.finish(result)
    }
}

#[cfg(test)]
mod batch_tests;

#[cfg(test)]
mod cache_tests;
