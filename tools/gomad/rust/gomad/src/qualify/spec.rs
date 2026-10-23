//! The public `qualify --spec FILE` owner grammar. Field names follow the
//! existing owner spec so one file drives either binary; fields the Rust host
//! cannot honor are refused rather than ignored.

use serde::Deserialize;
use serde_json::Value;
use std::path::{Path, PathBuf};

const MAXIMUM_SPEC_BYTES: u64 = 16 << 20;
const MAXIMUM_NESTING: usize = 128;

#[derive(Debug, Default, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct DeclaredManifest {
    #[serde(rename = "Provisional", default)]
    pub provisional: bool,
    #[serde(rename = "IntegratedSourceManifestPath", default)]
    pub integrated_source_manifest_path: String,
    #[serde(rename = "Schema", default)]
    pub schema: String,
    #[serde(rename = "ID", default)]
    pub id: String,
    #[serde(rename = "Workload", default)]
    pub workload: String,
    #[serde(rename = "Profile", default)]
    pub profile: String,
    #[serde(rename = "Cell", default)]
    pub cell: String,
    #[serde(rename = "IntegratedSourceSHA256", default)]
    pub integrated_source_sha256: String,
    #[serde(rename = "WorkloadSourceSHA256", default)]
    pub workload_source_sha256: String,
    #[serde(rename = "CompilerSHA256", default)]
    pub compiler_sha256: String,
    #[serde(rename = "RuntimeSHA256", default)]
    pub runtime_sha256: String,
    #[serde(rename = "HelperSHA256", default)]
    pub helper_sha256: String,
    #[serde(rename = "EngineSHA256", default)]
    pub engine_sha256: String,
    #[serde(rename = "ModelSHA256", default)]
    pub model_sha256: String,
    #[serde(rename = "HostConfigSHA256", default)]
    pub host_config_sha256: String,
    #[serde(rename = "HostOS", default)]
    pub host_os: String,
    #[serde(rename = "HostArch", default)]
    pub host_arch: String,
    #[serde(rename = "Seed", default)]
    pub seed: u64,
    #[serde(rename = "Repeat", default)]
    pub repeat: u64,
    #[serde(rename = "Fuel", default)]
    pub fuel: u64,
    #[serde(rename = "MemoryBytes", default)]
    pub memory_bytes: u64,
    #[serde(rename = "Unsupported", default)]
    pub unsupported: Vec<String>,
}

#[derive(Debug, Default, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Target {
    #[serde(rename = "Backend", default)]
    pub backend: String,
    #[serde(rename = "Kind", default)]
    pub kind: String,
    #[serde(rename = "Source", default)]
    pub source: String,
    #[serde(rename = "Args", default)]
    pub args: Vec<String>,
    #[serde(rename = "BuildTags", default)]
    pub build_tags: Vec<String>,
    #[serde(rename = "WorkingDir", default)]
    pub working_dir: String,
    /// Accepted for grammar compatibility; the Rust installation owns its
    /// preparation cache, so a supplied root must only be well formed.
    #[serde(rename = "PreparationRoot", default)]
    pub preparation_root: String,
    #[serde(rename = "Scenario", default)]
    pub scenario: Option<Value>,
}

#[derive(Debug, Default, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Campaign {
    #[serde(rename = "Target", default)]
    pub target: Target,
    #[serde(rename = "Environment", default)]
    pub environment: Vec<String>,
    #[serde(rename = "ExecutionTimeout", default)]
    pub execution_timeout: u64,
    #[serde(rename = "OverallTimeout", default)]
    pub overall_timeout: u64,
    #[serde(rename = "TerminateGrace", default)]
    pub terminate_grace: u64,
    #[serde(rename = "OnFailure", default)]
    pub on_failure: String,
    #[serde(rename = "FailureBudget", default)]
    pub failure_budget: u64,
    #[serde(rename = "OutputLimit", default)]
    pub output_limit: u64,
    #[serde(rename = "WorldTransitionLimit", default)]
    pub world_transition_limit: u64,
    #[serde(rename = "ChoiceTraceLimit", default)]
    pub choice_trace_limit: u64,
    #[serde(rename = "SuccessBytesLimit", default)]
    pub success_bytes_limit: u64,
    #[serde(rename = "Artifacts", default)]
    pub artifacts: String,
    /// Go-host campaign build label; the Rust records bind the linked
    /// implementation identity instead.
    #[serde(rename = "RunnerBuild", default)]
    #[expect(
        dead_code,
        reason = "Accepted owner grammar without a Rust-host meaning"
    )]
    pub runner_build: String,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Spec {
    #[serde(rename = "RuntimeWorkload", default)]
    pub runtime_workload: Option<super::workloads::recovery::Request>,
    #[serde(rename = "Manifest")]
    pub manifest: DeclaredManifest,
    #[serde(rename = "Campaign")]
    pub campaign: Campaign,
    #[serde(rename = "ArtifactRoot", default)]
    pub artifact_root: PathBuf,
}

mod preflight;

/// Strictly decodes bounded qualification JSON; duplicate keys, trailing data
/// and excessive nesting are refused before typed decoding.
pub(crate) fn strict<T: serde::de::DeserializeOwned>(bytes: &[u8]) -> Result<T, String> {
    strict_with_nesting(bytes, MAXIMUM_NESTING)
}

/// Strict decoding with a caller-selected tighter nesting bound.
/// # Errors
/// Refuses zero or a bound beyond the qualification maximum.
pub(crate) fn strict_with_nesting<T: serde::de::DeserializeOwned>(
    bytes: &[u8],
    maximum: usize,
) -> Result<T, String> {
    if maximum == 0 || maximum > MAXIMUM_NESTING {
        return Err("invalid JSON nesting bound".into());
    }
    let end = preflight::first(bytes, maximum)?;
    if has_trailing(&bytes[end..]) {
        return Err("qualification JSON has trailing data".into());
    }
    serde_json::from_value(bounded_value(&bytes[..end])?)
        .map_err(|error| format!("qualification JSON: {error}"))
}

/// Allows an owned typed decoder to preserve first-value errors before trailing
/// input. Syntax and bounded nesting are still checked before that decoder.
pub(crate) fn decode_first<T>(
    bytes: &[u8],
    decode: impl FnOnce(Value, bool) -> Result<T, String>,
) -> Result<T, String> {
    let end = preflight::first(bytes, MAXIMUM_NESTING)?;
    decode(bounded_value(&bytes[..end])?, has_trailing(&bytes[end..]))
}

fn has_trailing(bytes: &[u8]) -> bool {
    bytes
        .iter()
        .any(|byte| !matches!(byte, b' ' | b'\t' | b'\r' | b'\n'))
}

fn bounded_value(bytes: &[u8]) -> Result<Value, String> {
    let mut deserializer = serde_json::Deserializer::from_slice(bytes);
    // Both callers first admit the entire slice with the lexical depth128
    // guard, including the original empty-container edge. Other decoders
    // keep serde's default recursion limit.
    deserializer.disable_recursion_limit();
    Unique::deserialize(&mut deserializer)
        .map(|value| value.0)
        .map_err(|error| format!("qualification JSON: {error}"))
}

/// A JSON value whose objects must not repeat a member name.
struct Unique(Value);

impl<'de> Deserialize<'de> for Unique {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct Visitor;
        impl<'de> serde::de::Visitor<'de> for Visitor {
            type Value = Unique;
            fn expecting(&self, formatter: &mut std::fmt::Formatter) -> std::fmt::Result {
                formatter.write_str("JSON")
            }
            fn visit_bool<E>(self, value: bool) -> Result<Unique, E> {
                Ok(Unique(Value::Bool(value)))
            }
            fn visit_i64<E>(self, value: i64) -> Result<Unique, E> {
                Ok(Unique(value.into()))
            }
            fn visit_u64<E>(self, value: u64) -> Result<Unique, E> {
                Ok(Unique(value.into()))
            }
            fn visit_f64<E: serde::de::Error>(self, value: f64) -> Result<Unique, E> {
                serde_json::Number::from_f64(value)
                    .map(|number| Unique(Value::Number(number)))
                    .ok_or_else(|| E::custom("non-finite number"))
            }
            fn visit_str<E>(self, value: &str) -> Result<Unique, E> {
                Ok(Unique(Value::String(value.to_owned())))
            }
            fn visit_string<E>(self, value: String) -> Result<Unique, E> {
                Ok(Unique(Value::String(value)))
            }
            fn visit_unit<E>(self) -> Result<Unique, E> {
                Ok(Unique(Value::Null))
            }
            fn visit_seq<A: serde::de::SeqAccess<'de>>(
                self,
                mut sequence: A,
            ) -> Result<Unique, A::Error> {
                let mut values = Vec::new();
                while let Some(Unique(value)) = sequence.next_element()? {
                    values.push(value);
                }
                Ok(Unique(Value::Array(values)))
            }
            fn visit_map<A: serde::de::MapAccess<'de>>(
                self,
                mut map: A,
            ) -> Result<Unique, A::Error> {
                let mut values = serde_json::Map::new();
                while let Some(name) = map.next_key::<String>()? {
                    let Unique(value) = map.next_value()?;
                    if values.insert(name.clone(), value).is_some() {
                        return Err(serde::de::Error::custom(format!(
                            "duplicate JSON member {name:?}"
                        )));
                    }
                }
                Ok(Unique(Value::Object(values)))
            }
        }
        deserializer.deserialize_any(Visitor)
    }
}

pub(crate) fn read_bounded(path: &Path, limit: u64) -> Result<Vec<u8>, String> {
    use std::io::Read as _;
    let file = std::fs::File::open(path).map_err(|error| {
        if error.kind() == std::io::ErrorKind::NotFound {
            format!("open {}: no such file or directory", path.display())
        } else {
            format!("open {}: {error}", path.display())
        }
    })?;
    let metadata = file.metadata().map_err(|error| error.to_string())?;
    if !metadata.is_file() {
        return Err(format!("{} is not a regular file", path.display()));
    }
    let mut bytes = Vec::new();
    file.take(limit + 1)
        .read_to_end(&mut bytes)
        .map_err(|error| error.to_string())?;
    if bytes.len() as u64 > limit {
        return Err(format!("{} exceeds {limit} bytes", path.display()));
    }
    Ok(bytes)
}

/// # Errors
/// Refuses unreadable, malformed or Options-bearing specs with the grammar status.
pub(crate) fn read(path: &Path) -> Result<Spec, String> {
    let bytes = read_bounded(path, MAXIMUM_SPEC_BYTES)?;
    let value: Value = strict(&bytes)?;
    if value.as_object().is_some_and(|object| {
        object
            .keys()
            .any(|name| name.eq_ignore_ascii_case("Options"))
    }) {
        return Err(
            "qualification spec Options must be absent; installation owns the pinned backend"
                .into(),
        );
    }
    serde_json::from_value(value).map_err(|error| format!("qualification spec: {error}"))
}

/// Declared owner digests keep the existing `sha256:` spelling.
pub(crate) fn declared_digest(value: &str) -> Result<Option<String>, String> {
    if value.is_empty() {
        return Ok(None);
    }
    value
        .strip_prefix("sha256:")
        .filter(|digest| crate::evidence::canonical::valid_digest(digest))
        .map(|digest| Some(digest.to_owned()))
        .ok_or_else(|| format!("invalid declared qualification digest {value:?}"))
}

#[cfg(test)]
#[path = "receiving_depth_tests.rs"]
mod receiving_depth_tests;

#[cfg(test)]
mod tighter_nesting_tests {
    use super::*;
    fn nested(count: usize, value: &str) -> Vec<u8> {
        format!("{}{}{}", "[".repeat(count), value, "]".repeat(count)).into_bytes()
    }
    #[test]
    fn measurement_depth_preserves_original_scalar_and_empty_container_edges() {
        assert!(strict_with_nesting::<Value>(&nested(32, "0"), 32).is_ok());
        assert!(strict_with_nesting::<Value>(&nested(33, "0"), 32).is_err());
        assert!(strict_with_nesting::<Value>(&nested(33, ""), 32).is_ok());
        assert!(strict_with_nesting::<Value>(&nested(34, ""), 32).is_err());
    }
    #[test]
    fn tighter_bound_preserves_unique_complete_values_and_default_limit() {
        for raw in [br#"{"fd":1,"fd":2}"#.as_slice(), b"{} {}"] {
            assert!(strict_with_nesting::<Value>(raw, 32).is_err());
        }
        assert!(strict::<Value>(&nested(128, "0")).is_ok());
        assert!(strict::<Value>(&nested(129, "0")).is_err());
        assert!(strict_with_nesting::<Value>(b"{}", 0).is_err());
        assert!(strict_with_nesting::<Value>(b"{}", 129).is_err());
    }
}
