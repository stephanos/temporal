use super::*;
use crate::evidence::{
    record::Stream,
    store::{Publication, Store},
};
use std::path::PathBuf;

struct Fixture {
    root: PathBuf,
    path: PathBuf,
}
impl Fixture {
    fn new(source: &Publication, raw: &[u8]) -> Self {
        let mut record = source.record.clone();
        if let Ok(inputs) = crate::qualify::spec::strict::<Inputs>(raw) {
            record.identity.configuration =
                canonical::digest(&canonical::encode(&inputs.configuration).unwrap());
            record.execution_identity = record.identity.digest().unwrap();
        }
        record.streams.insert(
            FILE.into(),
            Stream {
                bytes: raw.len() as u64,
                sha256: canonical::digest(raw),
                records: 0,
            },
        );
        let mut payloads = source.payloads.clone();
        payloads.insert(FILE.into(), raw.to_vec());
        payloads.insert("record.json".into(), record.bytes().unwrap());
        let root = std::env::temp_dir().join(format!(
            "gomad-native-presence-{}",
            crate::campaign::origin::identity().unwrap()
        ));
        let published = Store {
            root: root.clone(),
            maximum_bytes: 128 << 20,
            target_pool: None,
            fault: None,
        }
        .publish(Publication { record, payloads })
        .unwrap();
        Self {
            root,
            path: published.path,
        }
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.root).unwrap();
    }
}

fn shape(baseline: &Value, variant: &str) -> Value {
    let mut value = baseline.clone();
    let config = value["configuration"].as_object_mut().unwrap();
    match variant {
        "optional-none-missing" => {
            config.remove("oracle_plan");
        }
        "optional-none-null" => {
            config.insert("oracle_plan".into(), Value::Null);
        }
        "optional-some-empty-rejected" => {
            config.insert(
                "oracle_plan".into(),
                json!({"schema":"gomad.wasm.soft-assertion-observation/v1","maximum_bytes":4096}),
            );
        }
        "optional-some-valid" => {
            let handle = config["incarnations"][0]["handle"].clone();
            config.get_mut("incarnations").unwrap()[0]["workload_kind"] = json!("go-test");
            config.insert("oracle_plan".into(),json!({"schema":"gomad.wasm.soft-assertion-observation/v1","maximum_bytes":4096,"soft_assertions":[{"handle":handle,"report_only":false}]}));
        }
        "default-vector-missing" => {
            config.remove("nodes");
        }
        "default-vector-empty" => {
            config.insert("nodes".into(), json!([]));
        }
        "default-vector-null" => {
            config.insert("nodes".into(), Value::Null);
        }
        "required-vector-empty" => {
            config.insert("crash_groups".into(), json!([]));
        }
        "required-vector-missing" => {
            config.remove("crash_groups");
        }
        "required-vector-null" => {
            config.insert("crash_groups".into(), Value::Null);
        }
        _ => unreachable!(),
    }
    value
}

fn spelling(value: &Value, kind: &str) -> Vec<u8> {
    let raw = canonical::encode(value).unwrap();
    match kind {
        "canonical" => raw,
        "pretty" => serde_json::to_vec_pretty(value).unwrap(),
        "order" => format!(
            "{{{}}}",
            value
                .as_object()
                .unwrap()
                .iter()
                .rev()
                .map(|(key, value)| format!(
                    "{}:{}",
                    serde_json::to_string(key).unwrap(),
                    serde_json::to_string(value).unwrap()
                ))
                .collect::<Vec<_>>()
                .join(",")
        )
        .into_bytes(),
        "escaped" => String::from_utf8(raw)
            .unwrap()
            .replacen("\"schema\":", "\"\\u0073chema\":", 1)
            .replacen(
                "gomad.cluster-replay-inputs",
                "\\u0067omad.cluster-replay-inputs",
                1,
            )
            .into_bytes(),
        _ => unreachable!(),
    }
}

fn authentication(opened: &Opened, path: &std::path::Path, raw: &[u8]) -> Value {
    let descriptor = Payload {
        file: FILE.into(),
        bytes: raw.len() as u64,
        sha256: canonical::digest(raw),
        limit: 16 << 20,
    };
    assert_eq!(descriptor.read(opened, 16 << 20).unwrap(), raw);
    let mut errors = Vec::new();
    for variant in ["digest", "length", "path"] {
        let mut changed = descriptor.clone();
        match variant {
            "digest" => changed.sha256 = canonical::digest(b"foreign bytes"),
            "length" => changed.bytes += 1,
            "path" => changed.file = "evidence/foreign.json".into(),
            _ => unreachable!(),
        }
        let error = changed.read(opened, 16 << 20).unwrap_err();
        errors.push(json!({"variant":variant,"descriptor":changed,"error":error}));
    }
    let appended = [raw, b"\n"].concat();
    std::fs::write(path.join(FILE), &appended).unwrap();
    let error = descriptor.read(opened, 16 << 20).unwrap_err();
    errors.push(json!({"variant":"raw-append","descriptor":descriptor,"raw_base64":crate::campaign::exploration::base64(&appended),"error":error}));
    json!(errors)
}

#[test]
fn native_optional_and_vector_presence_survives_store_and_actual_inputs_decoder() {
    let corpus = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../testdata/parity");
    let source = crate::fixture_cluster::publication(&corpus, 7).unwrap();
    let baseline: Value = serde_json::from_slice(&source.payloads[FILE]).unwrap();
    for variant in [
        "optional-none-missing",
        "optional-none-null",
        "optional-some-empty-rejected",
        "optional-some-valid",
        "default-vector-missing",
        "default-vector-empty",
        "default-vector-null",
        "required-vector-empty",
        "required-vector-missing",
        "required-vector-null",
    ] {
        let value = shape(&baseline, variant);
        let valid_publication = if variant == "optional-some-valid" {
            let inputs: Inputs = serde_json::from_value(value.clone()).unwrap();
            Some(
                crate::fixture_cluster::execute_configuration(
                    inputs.configuration,
                    inputs.toolchain_sha256,
                    BTreeMap::from([(
                        "target/module.wasm".into(),
                        source.payloads["target/module.wasm"].clone(),
                    )]),
                )
                .unwrap(),
            )
        } else {
            None
        };
        let source = valid_publication.as_ref().unwrap_or(&source);
        let value = if valid_publication.is_some() {
            serde_json::from_slice(&source.payloads[FILE]).unwrap()
        } else {
            value
        };
        for kind in ["canonical", "pretty", "order", "escaped"] {
            let raw = spelling(&value, kind);
            let fixture = Fixture::new(source, &raw);
            let opened = Opened::open(&fixture.path).unwrap();
            opened.authenticate_all().unwrap();
            let decoded = super::inputs(&opened);
            let (canonical_inputs, decoder_error, admission_error) = match decoded {
                Ok(inputs) => {
                    super::bind(&opened, &inputs).unwrap();
                    let oracle_error = oracles::validate(&inputs.configuration).err();
                    let admitted = super::verify_retained(&opened);
                    if variant == "optional-some-empty-rejected" {
                        assert!(inputs.configuration.oracle_plan.is_some());
                        assert_eq!(
                            oracle_error.as_deref(),
                            Some("invalid typed cluster oracle plan")
                        );
                        assert!(admitted.is_err());
                    } else {
                        assert!(oracle_error.is_none());
                        admitted.as_ref().unwrap();
                    }
                    (
                        Some(canonical::encode(&inputs).unwrap()),
                        None,
                        admitted.err(),
                    )
                }
                Err(error) => {
                    assert!(matches!(
                        variant,
                        "default-vector-null" | "required-vector-missing" | "required-vector-null"
                    ));
                    (None, Some(error), None)
                }
            };
            assert_eq!(opened.payload(FILE, 16 << 20).unwrap(), raw);
            let manifest = opened.manifest().clone();
            let mutations = authentication(&opened, &fixture.path, &raw);
            println!(
                "GOMAD_PRESENCE_OBSERVATION {}",
                json!({"schema":"gomad.native-retained-presence/v1","original_root":"TestRetainedOrdinarySpellingKeepsRawAuthenticationAndPresence","variant":variant,"spelling":kind,
                "inputs":{"baseline":baseline,"raw_base64":crate::campaign::exploration::base64(&raw),"manifest":manifest},
                "observed":{"canonical_base64":canonical_inputs.as_ref().map(|raw|crate::campaign::exploration::base64(raw)),"decoder_error":decoder_error,"admission_error":admission_error,"authentication":mutations}})
            );
        }
    }
}
