use crate::evidence::{
    canonical,
    store::{Opened, open},
};
use serde::{
    Deserialize, Deserializer, Serialize,
    de::{self, MapAccess, SeqAccess, Visitor},
};
use serde_json::{Value, json};
use std::{collections::BTreeMap, fmt, fs::File, path::Path};

#[derive(Clone, Copy)]
pub struct ReadLimits {
    pub json_bytes: u64,
    pub stream_bytes: u64,
    pub total_bytes: u64,
    pub module_bytes: u64,
}
impl Default for ReadLimits {
    fn default() -> Self {
        Self {
            json_bytes: 8 * 1024 * 1024,
            stream_bytes: 256 * 1024 * 1024,
            total_bytes: 512 * 1024 * 1024,
            module_bytes: 256 * 1024 * 1024,
        }
    }
}

#[derive(Clone, Debug, Serialize)]
pub struct NativeExecution {
    pub outcome: Value,
    pub streams: BTreeMap<String, Vec<u8>>,
    pub handle: Option<Value>,
}

#[derive(Clone, Debug, Serialize)]
pub struct NativeRun {
    pub request: Value,
    pub report: Value,
    pub profile: Value,
    pub root: NativeExecution,
    pub incarnations: Vec<NativeExecution>,
    pub executed_steps: Option<Value>,
    pub cutoff: Option<Value>,
    pub modules: Vec<Value>,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum VariantRole {
    Ordinary,
    DiagnosticProfile,
    GuestProfile,
    DiagnosticFuelOff,
}
#[derive(Clone, Copy)]
pub struct GcMask<'a> {
    /// An empty baseline declares that GOGC is absent and the runtime default applies.
    pub baseline: &'a str,
    pub candidate: &'a str,
}
#[derive(Clone, Copy)]
pub enum ComparisonMask<'a> {
    Exact,
    Gc {
        baseline: &'a str,
        candidate: &'a str,
    },
    Variant {
        baseline: VariantRole,
        candidate: VariantRole,
        gc: Option<GcMask<'a>>,
    },
    FuelOff,
    Sampling {
        interval_micros: u64,
        max_samples: u64,
    },
    Gowasm {
        proof: &'a super::module::VariantProof,
        baseline: VariantRole,
        candidate: VariantRole,
    },
}

struct Unique(Value);
impl<'de> Deserialize<'de> for Unique {
    fn deserialize<D: Deserializer<'de>>(decoder: D) -> Result<Self, D::Error> {
        decoder.deserialize_any(UniqueVisitor)
    }
}
struct UniqueVisitor;
impl<'de> Visitor<'de> for UniqueVisitor {
    type Value = Unique;
    fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str("JSON without duplicate fields")
    }
    fn visit_bool<E: de::Error>(self, v: bool) -> Result<Unique, E> {
        Ok(Unique(v.into()))
    }
    fn visit_i64<E: de::Error>(self, v: i64) -> Result<Unique, E> {
        Ok(Unique(v.into()))
    }
    fn visit_u64<E: de::Error>(self, v: u64) -> Result<Unique, E> {
        Ok(Unique(v.into()))
    }
    fn visit_f64<E: de::Error>(self, v: f64) -> Result<Unique, E> {
        serde_json::Number::from_f64(v)
            .map(|v| Unique(Value::Number(v)))
            .ok_or_else(|| E::custom("nonfinite JSON"))
    }
    fn visit_str<E: de::Error>(self, v: &str) -> Result<Unique, E> {
        Ok(Unique(v.into()))
    }
    fn visit_string<E: de::Error>(self, v: String) -> Result<Unique, E> {
        Ok(Unique(v.into()))
    }
    fn visit_unit<E: de::Error>(self) -> Result<Unique, E> {
        Ok(Unique(Value::Null))
    }
    fn visit_seq<A: SeqAccess<'de>>(self, mut seq: A) -> Result<Unique, A::Error> {
        let mut out = Vec::new();
        while let Some(Unique(v)) = seq.next_element()? {
            out.push(v);
        }
        Ok(Unique(out.into()))
    }
    fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Unique, A::Error> {
        let mut out = serde_json::Map::new();
        while let Some(key) = map.next_key::<String>()? {
            if out.contains_key(&key) {
                return Err(de::Error::custom("duplicate native evidence field"));
            }
            let Unique(v) = map.next_value()?;
            out.insert(key, v);
        }
        Ok(Unique(out.into()))
    }
}
fn decode(bytes: &[u8]) -> Result<Value, String> {
    let mut decoder = serde_json::Deserializer::from_slice(bytes);
    let Unique(value) = Unique::deserialize(&mut decoder).map_err(|error| error.to_string())?;
    decoder.end().map_err(|error| error.to_string())?;
    Ok(value)
}

struct Reader {
    root: File,
    limits: ReadLimits,
    remaining: u64,
}
impl Reader {
    fn raw(&mut self, name: &str, limit: u64) -> Result<Vec<u8>, String> {
        let bytes = open::read(&self.root, name, limit.min(self.remaining), false)?;
        self.remaining = self
            .remaining
            .checked_sub(bytes.len() as u64)
            .ok_or("native evidence total capacity")?;
        Ok(bytes)
    }
    fn json(&mut self, name: &str) -> Result<Value, String> {
        decode(&self.raw(name, self.limits.json_bytes)?)
    }
    fn execution(&mut self, prefix: &str, handle: bool) -> Result<NativeExecution, String> {
        let outcome = self.json(&format!("{prefix}outcome.json"))?;
        validate_outcome(&outcome)?;
        let catalogue = self.json(&format!("{prefix}streams.json"))?;
        let mut streams = BTreeMap::new();
        for (name, entry) in catalogue
            .as_object()
            .ok_or("native stream catalogue must be an object")?
        {
            open::components(name)?;
            if name.contains('/')
                || name.starts_with("module-")
                || name.starts_with('.')
                || [
                    "outcome.json",
                    "streams.json",
                    "handle.json",
                    "report.json",
                    "request.json",
                    "profile.json",
                    "rootmodules.json",
                    "executed-steps.json",
                    "cutoff.json",
                ]
                .contains(&name.as_str())
            {
                return Err("native stream name aliases metadata".into());
            }
            if entry.as_object().is_none_or(|entry| {
                entry.len() != 2 || !entry.contains_key("bytes") || !entry.contains_key("sha256")
            }) {
                return Err("native stream catalogue has unknown fields".into());
            }
            let size = entry["bytes"].as_u64().ok_or("native stream size absent")?;
            let digest = entry["sha256"]
                .as_str()
                .ok_or("native stream digest absent")?;
            if !canonical::valid_digest(digest) || size > self.limits.stream_bytes {
                return Err("invalid native stream admission".into());
            }
            let bytes = self.raw(&format!("{prefix}{name}"), size)?;
            if bytes.len() as u64 != size || canonical::digest(&bytes) != digest {
                return Err("native stream size or digest mismatch".into());
            }
            streams.insert(name.clone(), bytes);
        }
        Ok(NativeExecution {
            outcome,
            streams,
            handle: if handle {
                Some(self.json(&format!("{prefix}handle.json"))?)
            } else {
                None
            },
        })
    }
}

fn validate_outcome(value: &Value) -> Result<(), String> {
    if value["type"] != "result"
        || value["termination"].as_str().is_none_or(str::is_empty)
        || value["message"].as_str().is_none()
        || value["fuel_remaining"].as_u64().is_none()
        || value["peak_memory_bytes"].as_u64().is_none()
        || value["output_bytes"].as_u64().is_none()
        || value.get("exit_code").is_some_and(|code| {
            !code.is_null() && code.as_u64().is_none_or(|code| code > u64::from(u32::MAX))
        })
    {
        return Err("native outcome is incomplete".into());
    }
    Ok(())
}

#[cfg(feature = "helper-diagnostic-profile")]
fn validate_profile(profile: &Value) -> Result<(), String> {
    crate::profile::validate(profile)
}
#[cfg(not(feature = "helper-diagnostic-profile"))]
fn validate_profile(_: &Value) -> Result<(), String> {
    Err("diagnostic evidence validation requires the diagnostic reader role".into())
}

/// Reads private retained native artifacts without requiring their source module to remain present.
/// Terminal failures remain observable; acceptance is a separate caller decision.
/// # Errors
/// Refuses nonprivate paths, altered payloads, incomplete raw outcomes or mismatched request/module bindings.
#[expect(
    clippy::too_many_lines,
    reason = "Retained run admission checks private files, request roles and every declared stream in order"
)]
pub fn read_profile_run(root: &Path, limits: ReadLimits) -> Result<NativeRun, String> {
    let mut reader = Reader {
        root: open::pin(root)?,
        limits,
        remaining: limits.total_bytes,
    };
    let request_raw = reader.raw("request.json", limits.json_bytes)?;
    let request = decode(&request_raw)?;
    let report = reader.json("report.json")?;
    if report["schema"] != "gomad.linked-profile-run/v1"
        || report["request_sha256"] != canonical::digest(&request_raw)
        || !report["host_sha256"]
            .as_str()
            .is_some_and(canonical::valid_digest)
        || !report["linked_implementation_sha256"]
            .as_str()
            .is_some_and(canonical::valid_digest)
        || !report["host_implementation_sha256"]
            .as_str()
            .is_some_and(canonical::valid_digest)
    {
        return Err("native profile report identity mismatch".into());
    }
    let diagnostic = report["diagnostic_only"]
        .as_bool()
        .ok_or("native diagnostic role absent")?;
    let features: Vec<&str> = report["features"]
        .as_array()
        .ok_or("native feature vector absent")?
        .iter()
        .map(|value| value.as_str().ok_or("native feature must be a string"))
        .collect::<Result<_, _>>()?;
    let profile = if diagnostic {
        if !matches!(
            features.as_slice(),
            ["diagnostic-profile"]
                | ["diagnostic-profile", "guest-profile"]
                | ["diagnostic-fuel-off", "diagnostic-profile"]
        ) {
            return Err("native diagnostic feature vector is not admitted".into());
        }
        let profile = reader.json("profile.json")?;
        validate_profile(&profile)?;
        profile
    } else {
        if !features.is_empty() || !report["attribution"].is_null() {
            return Err("ordinary native run claims diagnostic features or attribution".into());
        }
        match rustix::fs::statat(
            &reader.root,
            "profile.json",
            rustix::fs::AtFlags::SYMLINK_NOFOLLOW,
        ) {
            Err(rustix::io::Errno::NOENT) => {}
            _ => return Err("ordinary native run must not contain a diagnostic profile".into()),
        }
        Value::Null
    };
    let modules = reader
        .json("rootmodules.json")?
        .as_array()
        .ok_or("native module catalogue must be an array")?
        .clone();
    let guests: Vec<(&Value, Option<&Value>)> = match request["kind"].as_str() {
        Some("standalone") => vec![(&request["guest"], None)],
        Some("cluster") => request["incarnations"]
            .as_array()
            .ok_or("native cluster incarnations absent")?
            .iter()
            .map(|entry| (&entry["guest"], Some(&entry["handle"])))
            .collect(),
        _ => return Err("unsupported native request kind".into()),
    };
    if guests.len() != modules.len() || guests.is_empty() {
        return Err("native retained module count mismatch".into());
    }
    for (index, ((guest, handle), module)) in guests.iter().zip(&modules).enumerate() {
        if module["request_module"] != guest["module"]
            || module["sha256"] != guest["module_sha256"]
            || module["handle"] != handle.map_or(Value::Null, |value| (*value).clone())
            || module["file_name"] != format!("module-{index}.wasm")
        {
            return Err("native retained module request binding mismatch".into());
        }
        let size = module["bytes"]
            .as_u64()
            .ok_or("native module size absent")?;
        if size > limits.module_bytes {
            return Err("native module capacity".into());
        }
        let bytes = reader.raw(&format!("module-{index}.wasm"), size)?;
        if bytes.len() as u64 != size || module["sha256"] != canonical::digest(&bytes) {
            return Err("native module size or digest mismatch".into());
        }
    }
    let root_execution = reader.execution("", false)?;
    let mut incarnations = Vec::new();
    let (executed_steps, cutoff) = if request["kind"] == "cluster" {
        let steps = reader.json("executed-steps.json")?;
        let cutoff = reader.json("cutoff.json")?;
        let retained = report["retained_incarnations"]
            .as_u64()
            .ok_or("native retained incarnation count absent")?;
        let retained =
            usize::try_from(retained).map_err(|_| "native incarnation count overflow")?;
        if retained > guests.len() {
            return Err("native retained incarnation count exceeds request".into());
        }
        // The native execution reports its retained count even when a terminal failure stops the schedule.
        for index in 0..retained {
            let directory = format!("incarnation-{index}");
            match open::child(&reader.root, &directory, true, false) {
                Ok(_) => {
                    let execution = reader.execution(&format!("{directory}/"), true)?;
                    if !guests.iter().any(|(_, handle)| {
                        handle.is_some_and(|handle| Some(handle) == execution.handle.as_ref())
                    }) {
                        return Err("native retained incarnation handle was not declared".into());
                    }
                    if incarnations
                        .iter()
                        .any(|prior: &NativeExecution| prior.handle == execution.handle)
                    {
                        return Err("duplicate native incarnation handle".into());
                    }
                    incarnations.push(execution);
                }
                Err(error) => return Err(error),
            }
        }
        (Some(steps), Some(cutoff))
    } else {
        (None, None)
    };
    Ok(NativeRun {
        request,
        report,
        profile,
        root: root_execution,
        incarnations,
        executed_steps,
        cutoff,
        modules,
    })
}

/// Uses the authenticated campaign store's complete manifest before returning raw payload bytes.
/// # Errors
/// Refuses any manifest or payload authentication failure.
pub fn campaign_payloads(path: &Path, bound: u64) -> Result<BTreeMap<String, Vec<u8>>, String> {
    let opened = Opened::open(path)?;
    opened.authenticate_all()?;
    let mut remaining = bound;
    let mut payloads = BTreeMap::new();
    for (name, entry) in &opened.manifest().files {
        if entry.bytes > remaining {
            return Err("campaign witness capacity".into());
        }
        let bytes = opened.payload(name, remaining)?;
        remaining -= bytes.len() as u64;
        payloads.insert(name.clone(), bytes);
    }
    Ok(payloads)
}

fn outcome_projection(
    outcome: &Value,
    observational: bool,
    mask_fuel: bool,
) -> Result<Value, String> {
    validate_outcome(outcome)?;
    let mut projected = outcome.clone();
    projected
        .as_object_mut()
        .ok_or("native outcome must be an object")?
        .remove("metrics");
    if observational {
        let object = projected
            .as_object_mut()
            .ok_or("native outcome must be an object")?;
        if mask_fuel {
            object.remove("fuel_remaining");
        }
        object.remove("peak_memory_bytes");
    }
    Ok(projected)
}

fn mask_gc(guest: &mut Value, expected: &str, remove_memory_limit: bool) -> Result<(), String> {
    if expected.contains(['=', '\0']) {
        return Err("GC mask needs an explicit value".into());
    }
    let model = guest
        .get_mut("model")
        .and_then(Value::as_object_mut)
        .ok_or("GC variant requires native modeled environment")?;
    let environment = model
        .get_mut("environment")
        .and_then(Value::as_array_mut)
        .ok_or("native environment absent")?;
    let original_count = environment.len();
    let original_bytes = environment.iter().try_fold(0usize, |sum, value| {
        let value = value
            .as_str()
            .ok_or("native environment entry must be a string")?;
        sum.checked_add(value.len() + 1)
            .ok_or("environment byte overflow")
    })?;
    let expected_declaration = json!({"bytes":original_bytes,"count":original_count,"chunks":original_bytes.div_ceil(64*1024)});
    let mut seen = false;
    let mut kept = Vec::new();
    for entry in environment.iter() {
        let entry = entry
            .as_str()
            .ok_or("native environment entry must be a string")?;
        if let Some(value) = entry.strip_prefix("GOGC=") {
            if seen || value != expected {
                return Err("GC variant value or uniqueness mismatch".into());
            }
            seen = true;
        } else if !(remove_memory_limit && entry.starts_with("GOMEMLIMIT=")) {
            kept.push(Value::String(entry.into()));
        }
    }
    if seen == expected.is_empty() {
        return Err("declared GOGC variant absent".into());
    }
    *environment = kept;
    // Config stores a derived packed-string declaration, not a second independent environment.
    let bytes = environment.iter().try_fold(0usize, |sum, value| {
        sum.checked_add(value.as_str().unwrap_or_default().len() + 1)
            .ok_or("environment byte overflow")
    })?;
    let declaration =
        json!({"bytes":bytes,"count":environment.len(),"chunks":bytes.div_ceil(64*1024)});
    let configuration = model
        .get_mut("configuration")
        .and_then(Value::as_object_mut)
        .ok_or("native model configuration absent")?;
    let bootstrap = json!({"bytes":0,"count":0,"chunks":0});
    if configuration.get("environment") == Some(&bootstrap) {
        // Native admission replaces the zero bootstrap declaration with actual packed strings.
        return Ok(());
    }
    if configuration.get("environment") != Some(&expected_declaration) {
        return Err("native derived environment declaration mismatch".into());
    }
    configuration.insert("environment".into(), declaration);
    Ok(())
}
fn request_projection(
    request: &Value,
    gc: Option<&str>,
    remove_memory_limit: bool,
) -> Result<Value, String> {
    let mut request = request.clone();
    if let Some(gc) = gc {
        match request["kind"].as_str() {
            Some("standalone") => mask_gc(&mut request["guest"], gc, remove_memory_limit)?,
            Some("cluster") => {
                for incarnation in request["incarnations"]
                    .as_array_mut()
                    .ok_or("native incarnations absent")?
                {
                    mask_gc(&mut incarnation["guest"], gc, remove_memory_limit)?;
                }
            }
            _ => return Err("unsupported native request kind".into()),
        }
    }
    Ok(request)
}

fn guard_role(run: &NativeRun, role: VariantRole) -> Result<(), String> {
    let (features, suffix): (&[&str], &str) = match role {
        VariantRole::Ordinary => (&[], ""),
        VariantRole::DiagnosticProfile => (&["diagnostic-profile"], "-diagnostic-profile"),
        VariantRole::GuestProfile => (&["diagnostic-profile", "guest-profile"], "-guest-profile"),
        VariantRole::DiagnosticFuelOff => (
            &["diagnostic-fuel-off", "diagnostic-profile"],
            "-diagnostic-fuel-off",
        ),
    };
    if run.report["diagnostic_only"] != !features.is_empty()
        || run.report["features"] != json!(features)
        || run.report["configuration"]
            != format!("{}{suffix}", gomad_wasmhost::linked::CONFIGURATION)
        || [
            "host_sha256",
            "host_implementation_sha256",
            "linked_implementation_sha256",
        ]
        .into_iter()
        .any(|name| {
            !run.report[name]
                .as_str()
                .is_some_and(canonical::valid_digest)
        })
    {
        return Err(
            "native comparison role, compiled configuration or feature vector mismatch".into(),
        );
    }
    Ok(())
}

fn validate_binary_streams(execution: &NativeExecution) -> Result<(), String> {
    for (name, bytes) in &execution.streams {
        if matches!(name.as_str(), "transcript" | "wasi") {
            super::count_operations(bytes, 64 << 20)?;
        }
    }
    Ok(())
}

fn report_role(run: &NativeRun) -> Result<VariantRole, String> {
    match run.report["features"]
        .as_array()
        .map(|features| {
            features
                .iter()
                .map(|value| value.as_str().ok_or("native feature must be a string"))
                .collect::<Result<Vec<_>, _>>()
        })
        .transpose()?
    {
        Some(features) if features.is_empty() => Ok(VariantRole::Ordinary),
        Some(features) if features == ["diagnostic-profile"] => Ok(VariantRole::DiagnosticProfile),
        Some(features) if features == ["diagnostic-profile", "guest-profile"] => {
            Ok(VariantRole::GuestProfile)
        }
        Some(features) if features == ["diagnostic-fuel-off", "diagnostic-profile"] => {
            Ok(VariantRole::DiagnosticFuelOff)
        }
        _ => Err("native helper role is not admitted".into()),
    }
}

fn normalize_cache(run: &NativeRun, request: &mut Value) -> Result<(), String> {
    let Some(value) = request.get("compile_cache") else {
        return Ok(());
    };
    if value.is_null() {
        request
            .as_object_mut()
            .ok_or("native request object absent")?
            .remove("compile_cache");
        return Ok(());
    }
    let path = Path::new(
        value
            .as_str()
            .ok_or("native compile cache must be an absolute path")?,
    );
    if !path.is_absolute()
        || path == Path::new("/")
        || path.components().any(|part| {
            !matches!(
                part,
                std::path::Component::RootDir | std::path::Component::Normal(_)
            )
        })
        || path.as_os_str()
            != path
                .components()
                .collect::<std::path::PathBuf>()
                .as_os_str()
    {
        return Err("native compile cache path must be absolute and clean".into());
    }
    let mut descriptor = File::open("/").map_err(|error| error.to_string())?;
    for part in path.components() {
        if let std::path::Component::Normal(part) = part {
            descriptor = File::from(
                rustix::fs::openat(
                    &descriptor,
                    part,
                    rustix::fs::OFlags::RDONLY
                        | rustix::fs::OFlags::DIRECTORY
                        | rustix::fs::OFlags::NOFOLLOW
                        | rustix::fs::OFlags::CLOEXEC,
                    rustix::fs::Mode::empty(),
                )
                .map_err(|error| error.to_string())?,
            );
        }
    }
    let checked = open::pin(path)?;
    let a = rustix::fs::fstat(&descriptor).map_err(|error| error.to_string())?;
    let b = rustix::fs::fstat(&checked).map_err(|error| error.to_string())?;
    if a.st_dev != b.st_dev
        || a.st_ino != b.st_ino
        || run.root.outcome["termination"] != "exit"
        || run.root.outcome["exit_code"] != 0
        || run.root.outcome["metrics"]["prepare_nanos"]
            .as_u64()
            .is_none_or(|value| value == 0)
        || run.root.outcome["metrics"]["cache_hit"].as_bool().is_none()
    {
        return Err(
            "native compile cache lacks live private admission or successful native metrics".into(),
        );
    }
    let role = report_role(run)?;
    guard_role(run, role)?;
    let bound = match run.request["kind"].as_str() {
        Some("standalone") => run.request["guest"]["module_sha256"]
            .as_str()
            .is_some_and(canonical::valid_digest),
        Some("cluster") => run.request["incarnations"]
            .as_array()
            .is_some_and(|incarnations| {
                !incarnations.is_empty()
                    && incarnations.iter().all(|incarnation| {
                        incarnation["guest"]["module_sha256"]
                            .as_str()
                            .is_some_and(canonical::valid_digest)
                    })
            }),
        _ => false,
    };
    if !bound {
        return Err("native compile cache module binding absent".into());
    }
    request
        .as_object_mut()
        .ok_or("native request object absent")?
        .remove("compile_cache");
    Ok(())
}

/// Compares native behavior under an explicitly selected observation contract.
/// # Errors
/// Refuses undeclared roles, malformed native transcripts or any semantic request/terminal change.
#[expect(
    clippy::too_many_lines,
    reason = "Each declared observation mask is authenticated before exact native behavior comparison"
)]
pub fn compare(
    before: &NativeRun,
    after: &NativeRun,
    mask: ComparisonMask<'_>,
) -> Result<(), String> {
    let (gc, roles, observational) = match mask {
        ComparisonMask::Exact => (None, None, false),
        ComparisonMask::Gc {
            baseline,
            candidate,
        } => (
            Some(GcMask {
                baseline,
                candidate,
            }),
            None,
            false,
        ),
        ComparisonMask::Variant {
            baseline,
            candidate,
            gc,
        } => (gc, Some((baseline, candidate)), true),
        ComparisonMask::FuelOff => (
            None,
            Some((
                VariantRole::DiagnosticProfile,
                VariantRole::DiagnosticFuelOff,
            )),
            true,
        ),
        ComparisonMask::Sampling { .. } => (
            None,
            Some((VariantRole::GuestProfile, VariantRole::GuestProfile)),
            true,
        ),
        ComparisonMask::Gowasm {
            baseline,
            candidate,
            ..
        } => (None, Some((baseline, candidate)), true),
    };
    let mask_fuel = match mask {
        ComparisonMask::FuelOff | ComparisonMask::Gowasm { .. } => true,
        ComparisonMask::Variant {
            baseline,
            candidate,
            gc,
        } => {
            gc.is_some()
                || baseline == VariantRole::DiagnosticFuelOff
                || candidate == VariantRole::DiagnosticFuelOff
        }
        _ => false,
    };
    if let Some((baseline, candidate)) = roles {
        guard_role(before, baseline)?;
        guard_role(after, candidate)?;
        for field in ["host_os", "host_arch"] {
            if before.report[field].as_str().is_none_or(str::is_empty)
                || before.report[field] != after.report[field]
            {
                return Err("native variant comparison changed actual host platform".into());
            }
        }
    }
    if gc.is_some_and(|gc| gc.candidate.is_empty()) {
        return Err("GC comparison needs an explicit candidate value".into());
    }
    let remove_memory_limit = gc.is_some_and(|gc| gc.candidate == "off");
    let mut before_request = request_projection(
        &before.request,
        gc.map(|gc| gc.baseline),
        remove_memory_limit,
    )?;
    let mut after_request = request_projection(
        &after.request,
        gc.map(|gc| gc.candidate),
        remove_memory_limit,
    )?;
    if !matches!(mask, ComparisonMask::Exact) {
        normalize_cache(before, &mut before_request)?;
        normalize_cache(after, &mut after_request)?;
    }
    if let ComparisonMask::Sampling {
        interval_micros,
        max_samples,
    } = mask
    {
        if !(1..=1_000_000).contains(&interval_micros)
            || !(1..=100_000).contains(&max_samples)
            || before.request["kind"] != "standalone"
            || after.request["kind"] != "standalone"
            || !before.request["guest"]["guest_profile"].is_null()
            || after.request["guest"]["guest_profile"]
                != json!({"interval_micros":interval_micros,"max_samples":max_samples})
        {
            return Err(
                "native sampling comparison changed declared bounds or sampling object".into(),
            );
        }
        let guest = after_request["guest"]
            .as_object_mut()
            .ok_or("native guest object absent")?;
        match before_request["guest"].get("guest_profile") {
            Some(value) => {
                guest.insert("guest_profile".into(), value.clone());
            }
            None => {
                guest.remove("guest_profile");
            }
        }
    }
    if let ComparisonMask::Gowasm { proof, .. } = mask {
        if !proof.matches_requests(&before.request, &after.request) {
            return Err("native GOWASM request is outside its authenticated Prepared proof".into());
        }
        after_request["guest"]["module"] = before_request["guest"]["module"].clone();
        after_request["guest"]["module_sha256"] = before_request["guest"]["module_sha256"].clone();
    }
    if before_request != after_request
        || before.executed_steps != after.executed_steps
        || before.cutoff != after.cutoff
        || before.incarnations.len() != after.incarnations.len()
    {
        return Err("native request, clock, steps or incarnation set changed".into());
    }
    for (b, a) in std::iter::once((&before.root, &after.root))
        .chain(before.incarnations.iter().zip(&after.incarnations))
    {
        validate_binary_streams(b)?;
        validate_binary_streams(a)?;
        if b.handle != a.handle
            || b.streams != a.streams
            || outcome_projection(&b.outcome, observational, mask_fuel)?
                != outcome_projection(&a.outcome, observational, mask_fuel)?
        {
            return Err("native terminal or retained behavior stream changed".into());
        }
    }
    Ok(())
}

/// Authenticates an exact native full-tape replay before comparing retained raw behavior.
/// # Errors
/// Refuses changed identities, prefixes, altered traces, incomplete declarations or semantic outcomes.
pub fn compare_replay(
    before: &NativeRun,
    after: &NativeRun,
    expected_identity: &gomad_wasmhost::linked::ChoiceIdentity,
) -> Result<(), String> {
    compare_replay_mode(before, after, expected_identity, false)
}

pub(crate) fn compare_prefix_replay(
    before: &NativeRun,
    after: &NativeRun,
    identity: &gomad_wasmhost::linked::ChoiceIdentity,
) -> Result<(), String> {
    compare_replay_mode(before, after, identity, true)
}

#[expect(
    clippy::too_many_lines,
    reason = "Full and prefix replay authenticate tape consumption before narrow bookkeeping normalization"
)]
fn compare_replay_mode(
    before: &NativeRun,
    after: &NativeRun,
    expected_identity: &gomad_wasmhost::linked::ChoiceIdentity,
    interrupted: bool,
) -> Result<(), String> {
    let original_model = &before.request["guest"]["model"];
    let replay_model = &after.request["guest"]["model"];
    if before.request["kind"] != "standalone"
        || after.request["kind"] != "standalone"
        || original_model["configuration"]["mode"] != 1
        || replay_model["configuration"]["mode"] != if interrupted { 3 } else { 2 }
        || !original_model["configuration"]["replay"].is_null()
        || !original_model["replay_trace"].is_null()
    {
        return Err("native full replay requires recorded mode one and replay mode two".into());
    }
    let trace = before
        .root
        .streams
        .get("choice")
        .ok_or("native recorded choice trace absent")?;
    if trace.len() > 64 << 20 {
        return Err("native replay trace capacity".into());
    }
    let raw: Vec<u8> = serde_json::from_value(replay_model["replay_trace"].clone())
        .map_err(|error| error.to_string())?;
    let declaration: gomad_wasmhost::linked::ReplayRequest =
        serde_json::from_value(replay_model["configuration"]["replay"].clone())
            .map_err(|error| error.to_string())?;
    declaration
        .validate(64 << 20)
        .map_err(|error| error.to_string())?;
    let expected_target = super::workloads::hex(&expected_identity.target_sha256);
    if raw != *trace
        || if interrupted {
            replay_model["configuration"]["replay"]["prefix_control"]
                != json!({"cutoff":declaration.decisions,"override":null})
        } else {
            declaration.prefix_control.is_some()
        }
        || canonical::encode(&declaration.execution_identity)?
            != canonical::encode(expected_identity)?
        || before.request["guest"]["module_sha256"] != expected_target
        || declaration.trace_sha256 != canonical::digest(trace)
        || declaration.trace_bytes != trace.len() as u64
        || declaration.trace_frames != (trace.len() as u64).div_ceil(65536)
    {
        return Err(
            "native full replay changed recorded bytes, framing, identity or prefix control".into(),
        );
    }
    let inspection =
        gomad_wasmhost::linked::inspect_choices(trace, expected_identity).map_err(str::to_owned)?;
    let plan =
        gomad_wasmhost::linked::project_choices(trace, expected_identity).map_err(str::to_owned)?;
    let tape_sha256 = canonical::digest(&plan.bytes);
    if declaration.decisions != inspection.decisions
        || declaration.decisions != plan.decisions.len() as u64
        || declaration.tape_sha256 != inspection.tape_sha256
        || declaration.tape_sha256 != tape_sha256
    {
        return Err(
            "native full replay declaration differs from helper-owned tape projection".into(),
        );
    }
    for field in [
        "host_sha256",
        "host_implementation_sha256",
        "linked_implementation_sha256",
        "configuration",
        "features",
        "diagnostic_only",
        "host_os",
        "host_arch",
    ] {
        if before.report.get(field).is_none() || before.report[field] != after.report[field] {
            return Err("native full replay changed retained role or source identity".into());
        }
    }
    let before_metadata = &before.root.outcome["local_model"];
    let after_metadata = &after.root.outcome["local_model"];
    if before_metadata["cursor"] != 0
        || before_metadata["tape_records"] != 0
        || before_metadata["tape_sha256"] != ""
        || before_metadata.get("next_expected") != Some(&Value::Null)
        || after_metadata["cursor"] != declaration.decisions
        || after_metadata["tape_records"] != declaration.decisions
        || after_metadata["tape_sha256"] != tape_sha256
        || after_metadata.get("next_expected") != Some(&Value::Null)
    {
        return Err(
            "native full replay bookkeeping does not prove complete tape consumption".into(),
        );
    }
    let mut normalized = after.clone();
    normalized.request["guest"]["model"]["configuration"]["mode"] =
        original_model["configuration"]["mode"].clone();
    let configuration = normalized.request["guest"]["model"]["configuration"]
        .as_object_mut()
        .ok_or("native model configuration absent")?;
    match original_model["configuration"].get("replay") {
        Some(value) => {
            configuration.insert("replay".into(), value.clone());
        }
        None => {
            configuration.remove("replay");
        }
    }
    let model = normalized.request["guest"]["model"]
        .as_object_mut()
        .ok_or("native request model absent")?;
    match original_model.get("replay_trace") {
        Some(value) => {
            model.insert("replay_trace".into(), value.clone());
        }
        None => {
            model.remove("replay_trace");
        }
    }
    // These three fields encode the independently authenticated replay tape, not guest behavior.
    for key in ["cursor", "tape_records", "tape_sha256"] {
        normalized.root.outcome["local_model"][key] = before_metadata[key].clone();
    }
    compare(before, &normalized, ComparisonMask::Exact)
}

/// # Errors
/// Refuses any module change outside the authenticated Prepared provenance proof.
pub fn compare_module_variant(
    before: &NativeRun,
    after: &NativeRun,
    proof: &super::module::VariantProof,
) -> Result<(), String> {
    let baseline = report_role(before)?;
    let candidate = report_role(after)?;
    if baseline != candidate {
        return Err("native module variant changed the helper role".into());
    }
    compare(
        before,
        after,
        ComparisonMask::Gowasm {
            proof,
            baseline,
            candidate,
        },
    )
}

/// # Errors
/// Refuses undeclared sampling options, roles or changed guest behavior.
pub fn compare_sampling(
    before: &NativeRun,
    after: &NativeRun,
    interval_micros: u64,
    max_samples: u64,
) -> Result<(), String> {
    compare(
        before,
        after,
        ComparisonMask::Sampling {
            interval_micros,
            max_samples,
        },
    )
}

#[cfg(test)]
#[path = "evidence_tests.rs"]
mod tests;
