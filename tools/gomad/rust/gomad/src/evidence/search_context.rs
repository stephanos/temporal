use super::{canonical, inspection::Details, store::Opened};
use crate::campaign::exploration::site_scope::{self, SiteScope};
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};

pub(crate) const FILE: &str = "provenance/search-context.json";
pub(crate) const MAX_BYTES: u64 = 16 << 20;
const SCHEMA: &str = "gomad.search-context/v1";

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Context {
    pub schema: String,
    pub patterns: Vec<String>,
    pub resolution: site_scope::Resolution,
    pub scope: SiteScope,
    pub plan_sha256: String,
    pub round: u64,
    pub identity: linked::ChoiceIdentity,
}

impl std::fmt::Debug for Context {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_tuple("Context")
            .field(&serde_json::to_value(self).map_err(|_| std::fmt::Error)?)
            .finish()
    }
}

impl PartialEq for Context {
    fn eq(&self, other: &Self) -> bool {
        self.schema == other.schema
            && self.patterns == other.patterns
            && self.resolution == other.resolution
            && self.scope == other.scope
            && self.plan_sha256 == other.plan_sha256
            && self.round == other.round
            && self.identity.target_sha256 == other.identity.target_sha256
            && self.identity.toolchain_build_key == other.identity.toolchain_build_key
            && self.identity.goos == other.identity.goos
            && self.identity.goarch == other.identity.goarch
            && self.identity.implementation_sha256 == other.identity.implementation_sha256
    }
}
impl Eq for Context {}

impl Context {
    pub(crate) fn new(
        patterns: Vec<String>,
        resolution: site_scope::Resolution,
        plan_sha256: String,
        round: u64,
        identity: linked::ChoiceIdentity,
    ) -> Self {
        Self {
            schema: SCHEMA.into(),
            scope: resolution.scope(&patterns),
            patterns,
            resolution,
            plan_sha256,
            round,
            identity,
        }
    }

    pub(crate) fn bytes(&self) -> Result<Vec<u8>, String> {
        self.validate()?;
        let bytes = canonical::encode(self)?;
        if bytes.len() as u64 > MAX_BYTES {
            return Err("search context exceeds byte bound".into());
        }
        Ok(bytes)
    }

    fn validate(&self) -> Result<(), String> {
        self.scope.validate()?;
        if self.schema != SCHEMA
            || site_scope::parse_patterns(&self.patterns.join(","))? != self.patterns
            || !canonical::valid_digest(&self.plan_sha256)
            || self.scope != self.resolution.scope(&self.patterns)
            || !canonical::valid_digest(&self.identity.toolchain_build_key)
            || self.identity.goos != "wasip1"
            || self.identity.goarch != "wasm"
        {
            return Err("invalid search context schema, scope or identity".into());
        }
        Ok(())
    }
}

pub(crate) fn read(opened: &Opened) -> Result<Option<Context>, String> {
    if !opened.manifest().files.contains_key(FILE) {
        return Ok(None);
    }
    let bytes = opened.payload(FILE, MAX_BYTES)?;
    let context: Context = serde_json::from_slice(&bytes)
        .map_err(|error| format!("invalid search context: {error}"))?;
    if context.bytes()? != bytes {
        return Err("search context is not canonical".into());
    }
    if opened.manifest().record.streams.contains_key(FILE) {
        return Err("search context must be manifest-only provenance".into());
    }
    let record = &opened.manifest().record;
    let target = crate::campaign::exploration::choice::hex(&context.identity.target_sha256);
    if target != record.identity.target
        || context.identity.toolchain_build_key != record.identity.toolchain
    {
        return Err("search context module or toolchain identity mismatch".into());
    }
    let details =
        Details::read(opened)?.ok_or("search context requires retained choice identity")?;
    let choices = details
        .choices
        .as_ref()
        .ok_or("search context requires retained choice identity")?;
    choices.inspect(opened)?;
    if canonical::encode(&choices.identity)? != canonical::encode(&context.identity)? {
        return Err("search context choice identity mismatch".into());
    }
    let module_entry = opened
        .manifest()
        .files
        .get("target/module.wasm")
        .ok_or("search context module is missing")?;
    let module = opened.payload("target/module.wasm", module_entry.bytes)?;
    context.resolution.validate(&module)?;
    context
        .resolution
        .validate_fingerprints(&context.identity, &context.patterns)?;
    Ok(Some(context))
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use crate::evidence::{
        identity::Identity,
        inspection::{ChoiceEvidence, Payload},
        record::{Limits, Outcome, Record, Stream},
        store::{Publication, Store},
    };
    use sha2::{Digest as _, Sha256};
    use std::{collections::BTreeMap, os::unix::fs::DirBuilderExt as _};

    pub(crate) fn publication(round: u64) -> Publication {
        let module = b"\0asm\x01\0\0\0".to_vec();
        let choice = linked::ChoiceIdentity {
            target_sha256: Sha256::digest(&module).into(),
            toolchain_build_key: "1".repeat(64),
            goos: "wasip1".into(),
            goarch: "wasm".into(),
            implementation_sha256: [2; 32],
        };
        let details = Details {
            schema: "gomad.inspection-evidence/v1".into(),
            outputs: None,
            world: None,
            simulation: None,
            minimization: None,
            choices: Some(ChoiceEvidence {
                profile: "gomad-choice-trace/v4".into(),
                implementation_sha256: "2".repeat(64),
                identity: choice.clone(),
                trace: Payload {
                    schema: "gomad-choice-trace/v4".into(),
                    file: "evidence/choice".into(),
                    sha256: canonical::digest(&[]),
                    bytes: 0,
                    limit: 4096,
                },
                records: 0,
                terminal_state: "complete".into(),
                tape_sha256: Some(linked::inspect_choices(&[], &choice).unwrap().tape_sha256),
            }),
        };
        let details = canonical::encode(&details).unwrap();
        let identity = Identity {
            target: canonical::digest(&module),
            toolchain: choice.toolchain_build_key.clone(),
            platform: "3".repeat(64),
            implementation: "4".repeat(64),
            configuration: "5".repeat(64),
        };
        let record = Record {
            schema: "gomad.execution-record/v2".into(),
            execution_identity: identity.digest().unwrap(),
            identity,
            profile: "gomad.wasi-cooperative/v1".into(),
            seed: 7,
            arguments: Vec::new(),
            environment: BTreeMap::new(),
            limits: Limits {
                fuel: 1,
                memory_bytes: 65536,
                wall_millis: 1,
                output_bytes: 4096,
                evidence_bytes: 4096,
            },
            virtual_time_nanos: 0,
            outcome: Outcome {
                class: "success".into(),
                termination: "exit".into(),
                exit_code: Some(0),
                reason: String::new(),
            },
            streams: BTreeMap::from([
                (
                    "evidence/inspection.json".into(),
                    Stream {
                        bytes: details.len() as u64,
                        sha256: canonical::digest(&details),
                        records: 1,
                    },
                ),
                (
                    "evidence/choice".into(),
                    Stream {
                        bytes: 0,
                        sha256: canonical::digest(&[]),
                        records: 0,
                    },
                ),
            ]),
        };
        let context = Context::new(
            vec!["main".into()],
            site_scope::Resolution::default(),
            "6".repeat(64),
            round,
            choice,
        );
        Publication {
            record,
            payloads: BTreeMap::from([
                ("target/module.wasm".into(), module),
                ("evidence/inspection.json".into(), details),
                ("evidence/choice".into(), Vec::new()),
                (FILE.into(), context.bytes().unwrap()),
            ]),
        }
    }

    pub(crate) fn store() -> Store {
        static SEQUENCE: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);
        let root = std::env::temp_dir().join(format!(
            "gomad-search-context-{}-{}",
            std::process::id(),
            SEQUENCE.fetch_add(1, std::sync::atomic::Ordering::Relaxed)
        ));
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&root)
            .unwrap();
        Store {
            root,
            maximum_bytes: 0,
            target_pool: None,
            #[cfg(feature = "determinism-fixture")]
            fault: None,
        }
    }

    #[test]
    fn search_context_variants_preserve_record_and_failure_identity_and_retire() {
        let store = store();
        let first = publication(0);
        let mut second = publication(1);
        let mut other: Context = serde_json::from_slice(&second.payloads[FILE]).unwrap();
        other.patterns = vec!["runtime".into()];
        other.scope = other.resolution.scope(&other.patterns);
        second.payloads.insert(FILE.into(), other.bytes().unwrap());
        assert_eq!(first.record, second.record);
        assert_eq!(
            first.record.failure_signature().unwrap(),
            second.record.failure_signature().unwrap()
        );
        let first = store.publish(first).unwrap();
        let second = store.publish(second).unwrap();
        assert_ne!(first.path, second.path);
        assert_eq!(first.record_sha256, second.record_sha256);
        assert_eq!(
            read(&Opened::open(&first.path).unwrap())
                .unwrap()
                .unwrap()
                .round,
            0
        );
        assert_eq!(
            read(&Opened::open(&second.path).unwrap())
                .unwrap()
                .unwrap()
                .round,
            1
        );
        assert_eq!(store.publish(publication(0)).unwrap().path, first.path);
        Store::retire(
            &crate::evidence::store::open::pin(&store.root).unwrap(),
            &std::collections::BTreeSet::new(),
        )
        .unwrap();
        assert!(!first.path.exists() && !second.path.exists());
        std::fs::remove_dir_all(store.root).unwrap();
    }

    #[test]
    fn search_context_detached_opener_refuses_present_invalid_typed_payloads() {
        let store = store();
        let published = store.publish(publication(0)).unwrap();
        let original: serde_json::Value =
            serde_json::from_slice(&std::fs::read(published.path.join(FILE)).unwrap()).unwrap();
        let manifest_path = published.path.join("manifest.json");
        let original_manifest = std::fs::read(&manifest_path).unwrap();
        for (field, value) in [
            ("schema", serde_json::json!("unknown")),
            ("patterns", serde_json::json!(["invalid,"])),
            ("plan_sha256", serde_json::json!("invalid")),
            ("unknown", serde_json::json!(true)),
        ] {
            let mut malformed = original.clone();
            malformed[field] = value;
            let bytes = canonical::encode(&malformed).unwrap();
            std::fs::write(published.path.join(FILE), &bytes).unwrap();
            let mut manifest: super::super::store::Manifest =
                serde_json::from_slice(&original_manifest).unwrap();
            let entry = manifest.files.get_mut(FILE).unwrap();
            entry.bytes = bytes.len() as u64;
            entry.sha256 = canonical::digest(&bytes);
            std::fs::write(&manifest_path, canonical::encode(&manifest).unwrap()).unwrap();
            assert!(Opened::open(&published.path).is_err(), "{field}");
        }
        std::fs::remove_dir_all(store.root).unwrap();
    }

    #[test]
    fn search_context_malformed_present_is_refused_but_absence_is_legacy() {
        let store = store();
        let mut absent = publication(0);
        absent.payloads.remove(FILE);
        let published = store.publish(absent).unwrap();
        assert!(
            read(&Opened::open(&published.path).unwrap())
                .unwrap()
                .is_none()
        );
        for raw in [
            b"{}".to_vec(),
            b"null".to_vec(),
            vec![b' '; usize::try_from(MAX_BYTES).unwrap() + 1],
        ] {
            let mut malformed = publication(0);
            malformed.payloads.insert(FILE.into(), raw);
            assert!(store.publish(malformed).is_err());
        }
        std::fs::remove_dir_all(store.root).unwrap();
    }
}
