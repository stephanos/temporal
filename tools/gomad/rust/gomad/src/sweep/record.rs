#[cfg(test)]
mod tests {
    use super::super::enumerate::{SuiteMethod, SuiteRegistration};
    use super::*;
    use serde_json::json;
    use std::sync::atomic::{AtomicU64, Ordering};

    fn header() -> Header {
        Header {
            statement: OBSERVATION_STATEMENT.into(),
            candidate: {
                let mut candidate = BTreeMap::new();
                for key in [
                    "git_sha",
                    "source_sha256",
                    "gomad_sha256",
                    "helper_sha256",
                    "compiler_sha256",
                    "go_version",
                    "runtime_implementation_sha256",
                    "model_sha256",
                    "guest_config_sha256",
                    "engine",
                    "profile",
                    "config_source",
                ] {
                    candidate.insert(
                        key.into(),
                        json!(if key == "config_source" {
                            "throwaway".to_owned()
                        } else if key.ends_with("_sha256") {
                            format!("sha256:{}", "a".repeat(64))
                        } else {
                            "identity".to_owned()
                        }),
                    );
                }
                for key in [
                    "memory_bytes",
                    "fuel",
                    "seed",
                    "per_test_timeout_nanos",
                    "prepare_timeout_nanos",
                ] {
                    candidate.insert(key.into(), json!(1));
                }
                for key in ["build_tags", "explore_flags", "test_flags"] {
                    candidate.insert(key.into(), json!(["flag"]));
                }
                candidate
            },
            host: json!({"hostname":"test"}),
            packages: vec![Package {
                pattern: "./tests".into(),
                tests: vec!["TestA".into(), "TestB".into()],
                top_level_tests: vec!["TestA".into(), "TestB".into()],
                list_count: 2,
                parsed_count: 2,
                import_path: "tests".into(),
                module_directory: ".".into(),
                target: "./tests".into(),
                guest_excluded: BTreeMap::new(),
                methods: Vec::new(),
                registrations: Vec::new(),
                units: BTreeMap::from([
                    ("TestA".into(), vec!["TestA".into()]),
                    ("TestB".into(), vec!["TestB".into()]),
                ]),
            }],
            options: json!({"max_tests":2,"retain_bytes":0,"overall_budget_nanos":1,"config_source":"throwaway"}),
            harness_sha256: format!("sha256:{}", "b".repeat(64)),
        }
    }

    fn suite_header() -> Header {
        let mut header = header();
        let package = &mut header.packages[0];
        package.top_level_tests = vec!["TestPlain".into(), "TestSuite".into()];
        package.tests = vec![
            "TestPlain".into(),
            "TestSuite/TestOne".into(),
            "TestSuite/TestTwo".into(),
        ];
        package.units = BTreeMap::from([
            ("TestPlain".into(), vec!["TestPlain".into()]),
            (
                "TestSuite".into(),
                vec!["TestSuite/TestOne".into(), "TestSuite/TestTwo".into()],
            ),
        ]);
        package.methods = ["TestOne", "TestTwo"]
            .map(|name| SuiteMethod {
                suite: "S".into(),
                name: name.into(),
                file: "suite_test.go".into(),
                units: vec![format!("TestSuite/{name}")],
            })
            .to_vec();
        package.registrations = vec![SuiteRegistration {
            wrapper: "TestSuite".into(),
            suite: "S".into(),
            path: "TestSuite".into(),
            file: "suite_test.go".into(),
            methods: vec!["TestOne".into(), "TestTwo".into()],
        }];
        header
    }

    fn rekey(sidecar: &mut Sidecar) {
        sidecar.candidate_key = Header {
            statement: sidecar.statement.clone(),
            candidate: sidecar.candidate.clone(),
            host: sidecar.hosts.first().cloned().unwrap_or(Value::Null),
            packages: sidecar.packages.clone(),
            options: sidecar.options.clone(),
            harness_sha256: sidecar.harness_sha256.clone(),
        }
        .key()
        .unwrap();
    }

    #[test]
    fn verifier_checks_source_method_inventory_independently() {
        for case in [
            "valid",
            "missing-method-unit",
            "unknown-method-unit",
            "unknown-wrapper",
            "bulk-suite-unit",
            "unsorted-methods",
            "duplicate-method",
            "wrong-prefix",
        ] {
            let path = path();
            let journal = Journal::open(&path, &suite_header()).unwrap();
            let mut sidecar = build_sidecar(journal.rows()).unwrap();
            let package = &mut sidecar.packages[0];
            let expected = match case {
                "valid" => "",
                "missing-method-unit" => {
                    package.units.get_mut("TestSuite").unwrap().clear();
                    "method has no units"
                }
                "unknown-method-unit" => {
                    package
                        .units
                        .get_mut("TestSuite")
                        .unwrap()
                        .push("TestSuite/TestUnknown".into());
                    "method units do not exactly match"
                }
                "unknown-wrapper" => {
                    let units = package.units.remove("TestSuite").unwrap();
                    package.units.insert("TestLost".into(), units);
                    "registration wrappers do not match"
                }
                "bulk-suite-unit" => {
                    package.tests = vec!["TestPlain".into(), "TestSuite".into()];
                    "method units do not exactly match"
                }
                "unsorted-methods" => {
                    package.units.get_mut("TestSuite").unwrap().reverse();
                    "method units are not sorted and unique"
                }
                "duplicate-method" => {
                    package
                        .units
                        .get_mut("TestSuite")
                        .unwrap()
                        .push("TestSuite/TestTwo".into());
                    "method units are not sorted and unique"
                }
                _ => {
                    package.units.get_mut("TestSuite").unwrap()[0] = "TestOther/TestOne".into();
                    "method unit has wrong wrapper"
                }
            };
            rekey(&mut sidecar);
            let findings = verify(&sidecar);
            if expected.is_empty() {
                assert!(findings.is_empty(), "{findings:?}");
            } else {
                assert!(
                    findings.iter().any(|finding| finding.contains(expected)),
                    "{case}: {findings:?}"
                );
            }
            drop(journal);
            fs::remove_file(path).unwrap();
        }
    }

    #[test]
    fn declarations_reject_consistently_mutated_output_inventory() {
        for case in [
            "bulk-wrapper",
            "missing-declared-method",
            "empty-declared-coverage",
            "unknown-declared-unit",
            "unknown-registration",
            "undeclared-selected-unit",
        ] {
            let path = path();
            let journal = Journal::open(&path, &suite_header()).unwrap();
            let mut sidecar = build_sidecar(journal.rows()).unwrap();
            let package = &mut sidecar.packages[0];
            let expected = match case {
                "bulk-wrapper" => {
                    package.tests = vec!["TestPlain".into(), "TestSuite".into()];
                    package
                        .units
                        .insert("TestSuite".into(), vec!["TestSuite".into()]);
                    "suite wrapper is a runnable unit"
                }
                "missing-declared-method" => {
                    package.tests.pop();
                    package.units.get_mut("TestSuite").unwrap().pop();
                    "method unit not enumerated"
                }
                "empty-declared-coverage" => {
                    package.tests.pop();
                    package.units.get_mut("TestSuite").unwrap().pop();
                    package.methods[1].units.clear();
                    "declared method has no units"
                }
                "unknown-declared-unit" => {
                    package.methods[1].units = vec!["TestSuite/TestLost".into()];
                    "method unit not enumerated"
                }
                "unknown-registration" => {
                    package.registrations[0].wrapper = "TestLost".into();
                    "registration wrapper not enumerated"
                }
                _ => {
                    package.tests.push("TestSuite/TestUnknown".into());
                    package
                        .units
                        .get_mut("TestSuite")
                        .unwrap()
                        .push("TestSuite/TestUnknown".into());
                    "unit has no source declaration"
                }
            };
            let structural = inventory(&sidecar.packages).unwrap_err();
            assert!(structural.contains(expected), "{case}: {structural}");
            rekey(&mut sidecar);
            let findings = verify(&sidecar);
            assert!(
                findings.iter().any(|finding| finding.contains(expected)),
                "{case}: {findings:?}"
            );
            drop(journal);
            fs::remove_file(path).unwrap();
        }
    }

    #[test]
    fn nested_suite_declarations_cover_each_parent_and_leaf() {
        let mut header = suite_header();
        let package = &mut header.packages[0];
        let leaf = "TestSuite/TestNested/sub/TestLeaf".to_owned();
        package.tests = vec!["TestPlain".into(), leaf.clone()];
        package.units.insert("TestSuite".into(), vec![leaf.clone()]);
        package.methods = vec![
            SuiteMethod {
                suite: "Outer".into(),
                name: "TestNested".into(),
                file: "outer_test.go".into(),
                units: vec![leaf.clone()],
            },
            SuiteMethod {
                suite: "Inner".into(),
                name: "TestLeaf".into(),
                file: "inner_test.go".into(),
                units: vec![leaf],
            },
        ];
        package.registrations = vec![
            SuiteRegistration {
                wrapper: "TestSuite".into(),
                suite: "Outer".into(),
                path: "TestSuite".into(),
                file: "outer_test.go".into(),
                methods: vec!["TestNested".into()],
            },
            SuiteRegistration {
                wrapper: "TestSuite".into(),
                suite: "Inner".into(),
                path: "TestSuite/TestNested/sub".into(),
                file: "inner_test.go".into(),
                methods: vec!["TestLeaf".into()],
            },
        ];
        let path = path();
        let journal = Journal::open(&path, &header).unwrap();
        let sidecar = build_sidecar(journal.rows()).unwrap();
        assert_eq!(verify(&sidecar), Vec::<String>::new());
        for method in 0..2 {
            let mut changed = sidecar.clone();
            changed.packages[0].methods[method].units.clear();
            rekey(&mut changed);
            assert!(
                verify(&changed)
                    .iter()
                    .any(|finding| finding.contains("declared method has no units"))
            );
        }
        let mut changed = sidecar;
        changed.packages[0].tests.pop();
        changed.packages[0]
            .units
            .get_mut("TestSuite")
            .unwrap()
            .clear();
        rekey(&mut changed);
        assert!(!verify(&changed).is_empty());
        drop(journal);
        fs::remove_file(path).unwrap();
    }

    #[test]
    fn registration_effective_methods_have_independent_selected_coverage() {
        let path = path();
        let journal = Journal::open(&path, &suite_header()).unwrap();
        let mut sidecar = build_sidecar(journal.rows()).unwrap();
        let package = &mut sidecar.packages[0];
        package.tests.pop();
        package.units.get_mut("TestSuite").unwrap().pop();
        package.methods.pop();
        rekey(&mut sidecar);
        assert!(
            verify(&sidecar)
                .iter()
                .any(|finding| finding
                    .contains("registration effective method has no selected unit"))
        );
        drop(journal);
        fs::remove_file(path).unwrap();
    }

    fn test_entry(name: &str, class: Class, reason: &str) -> TestEntry {
        let mut entry = TestEntry::new("./tests", name, class, reason);
        entry.config_source = "throwaway".into();
        entry
    }

    fn path() -> std::path::PathBuf {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        std::env::temp_dir().join(format!(
            "gomad-native-sweep-journal-{}-{}.jsonl",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ))
    }

    #[test]
    fn resumes_only_matching_headers_and_repairs_torn_tail() {
        let path = path();
        let header = header();
        let mut journal = Journal::open(&path, &header).unwrap();
        journal
            .append_test(test_entry("TestA", Class::Pass, ""))
            .unwrap();
        drop(journal);
        OpenOptions::new()
            .append(true)
            .open(&path)
            .unwrap()
            .write_all(b"{\"kind\":")
            .unwrap();
        let mut journal = Journal::open(&path, &header).unwrap();
        assert_eq!(journal.latest().len(), 1);
        journal
            .append_test(test_entry("TestB", Class::Pass, ""))
            .unwrap();
        let sidecar = build_sidecar(journal.rows()).unwrap();
        assert_eq!(verify(&sidecar), Vec::<String>::new());
        drop(journal);
        let before = fs::read(&path).unwrap();
        let mut changed = header.clone();
        changed.options["max_tests"] = json!(3);
        assert!(Journal::open(&path, &changed).is_err());
        assert_eq!(fs::read(&path).unwrap(), before);
        let mut changed = header.clone();
        changed
            .candidate
            .insert("source_sha256".into(), json!("sha256:changed"));
        assert!(Journal::open(&path, &changed).is_err());
        changed = header.clone();
        changed.packages[0].tests.pop();
        assert!(Journal::open(&path, &changed).is_err());
        changed = header;
        changed.harness_sha256.push('x');
        assert!(Journal::open(&path, &changed).is_err());
        assert_eq!(fs::read(&path).unwrap(), before);
        fs::remove_file(path).unwrap();
    }

    #[test]
    fn preserves_complete_tail_without_newline_and_rejects_invalid_append() {
        let path = path();
        let header = header();
        let journal = Journal::open(&path, &header).unwrap();
        drop(journal);
        let mut bytes = fs::read(&path).unwrap();
        assert_eq!(bytes.pop(), Some(b'\n'));
        fs::write(&path, &bytes).unwrap();
        let mut journal = Journal::open(&path, &header).unwrap();
        assert_eq!(journal.rows().len(), 2);
        let before = fs::read(&path).unwrap();
        assert!(
            journal
                .append_test(test_entry("TestLost", Class::Pass, ""))
                .is_err()
        );
        assert!(
            journal
                .append_test(test_entry("TestA", Class::NotRun, " "))
                .is_err()
        );
        assert_eq!(fs::read(&path).unwrap(), before);
        drop(journal);
        fs::remove_file(path).unwrap();
    }

    #[test]
    fn warmup_records_do_not_complete_test_inventory() {
        let path = path();
        let mut journal = Journal::open(&path, &header()).unwrap();
        journal
            .append_warmup(test_entry(
                "(warm-up)",
                Class::NotRun,
                "warm-up only; not a test row",
            ))
            .unwrap();
        assert!(journal.latest().is_empty());
        let before = fs::read(&path).unwrap();
        for entry in [
            test_entry("TestA", Class::NotRun, "warm-up only; not a test row"),
            test_entry("(warm-up)", Class::Pass, "warm-up only; not a test row"),
            test_entry("(warm-up)", Class::NotRun, "another reason"),
        ] {
            assert!(journal.append_warmup(entry).is_err());
        }
        let mut wrong_source =
            test_entry("(warm-up)", Class::NotRun, "warm-up only; not a test row");
        wrong_source.config_source = "temporal-install".into();
        assert!(journal.append_warmup(wrong_source).is_err());
        assert_eq!(fs::read(&path).unwrap(), before);
        let sidecar = build_sidecar(journal.rows()).unwrap();
        assert_eq!(sidecar.by_class.get("not-run"), Some(&2));
        assert_eq!(verify(&sidecar), Vec::<String>::new());
        drop(journal);
        let journal = Journal::open(&path, &header()).unwrap();
        assert!(journal.latest().is_empty());
        drop(journal);
        fs::remove_file(path).unwrap();
    }

    #[test]
    fn rejects_old_schema_and_malformed_interior_without_mutation() {
        for bytes in [
            b"{\"schema\":\"gomad-wasm.sweep-journal/v1\"}\n".as_slice(),
            b"garbage\n{\"kind\":".as_slice(),
            b"{\"schema\":\"gomad-wasm.sweep-journal/v1\"}".as_slice(),
        ] {
            let path = path();
            fs::write(&path, bytes).unwrap();
            assert!(Journal::open(&path, &header()).is_err());
            assert_eq!(fs::read(&path).unwrap(), bytes);
            fs::remove_file(path).unwrap();
        }
    }

    #[test]
    fn provisional_native_v1_is_refused_before_append() {
        let path = path();
        let journal = Journal::open(&path, &header()).unwrap();
        let mut sidecar = build_sidecar(journal.rows()).unwrap();
        let mut encoded = serde_json::to_value(&journal.rows()[0]).unwrap();
        drop(journal);
        encoded["schema"] = json!("gomad.rust-wasm-sweep/v1");
        let mut bytes = serde_json::to_vec(&encoded).unwrap();
        bytes.push(b'\n');
        fs::write(&path, &bytes).unwrap();
        assert!(Journal::open(&path, &header()).is_err());
        assert_eq!(fs::read(&path).unwrap(), bytes);
        sidecar.schema = "gomad.rust-wasm-sweep/v1".into();
        assert!(
            verify(&sidecar)
                .iter()
                .any(|finding| finding == "incompatible sidecar schema")
        );
        fs::remove_file(path).unwrap();
    }

    #[test]
    fn verifier_rejects_bad_rows_inventory_counts_and_unknown_fields() {
        let path = path();
        let journal = Journal::open(&path, &header()).unwrap();
        let valid = build_sidecar(journal.rows()).unwrap();
        assert_eq!(verify(&valid), Vec::<String>::new());
        for index in 0..19 {
            let mut altered = valid.clone();
            match index {
                0 => {
                    altered.rows.pop();
                }
                1 => altered.rows.push(altered.rows[0].clone()),
                2 => altered.rows[0].name = "TestUnknown".into(),
                3 => altered.rows[0].reason.clear(),
                4 => {
                    altered.by_class.clear();
                }
                5 => {
                    altered.reasons.clear();
                }
                6 => altered.packages[0].parsed_count += 1,
                7 => {
                    altered.candidate.remove("helper_sha256");
                    altered.candidate_key = Header {
                        candidate: altered.candidate.clone(),
                        ..header()
                    }
                    .key()
                    .unwrap();
                }
                8 => altered.rows[0].skipped = true,
                9 => {
                    altered.by_package.clear();
                }
                10 => altered.statement = "qualified".into(),
                11 => altered.rows[0].config_source.clear(),
                12 => {
                    altered
                        .candidate
                        .insert("helper_sha256".into(), json!("sha256:abc"));
                }
                13 => altered.harness_sha256 = format!("sha256:{}", "A".repeat(64)),
                14 => altered.hosts.clear(),
                15 => altered.candidate_key = "sha256:stale".into(),
                16 => {
                    altered.candidate.remove("git_sha");
                    rekey(&mut altered);
                }
                17 => {
                    altered.candidate.remove("runtime_implementation_sha256");
                    rekey(&mut altered);
                }
                _ => altered.schema = "gomad-wasm.sweep/v2".into(),
            }
            assert!(!verify(&altered).is_empty(), "mutation {index}");
        }
        for class in ["", "flaky"] {
            let mut encoded = serde_json::to_value(&valid).unwrap();
            encoded["rows"][0]["class"] = json!(class);
            fs::write(&path, serde_json::to_vec(&encoded).unwrap()).unwrap();
            assert!(verify_file(&path).is_err(), "invalid class {class}");
        }
        let mut encoded = serde_json::to_value(&valid).unwrap();
        encoded["support"] = json!(true);
        fs::write(&path, serde_json::to_vec(&encoded).unwrap()).unwrap();
        assert!(verify_file(&path).is_err());
        fs::remove_file(path).unwrap();
    }
}

use super::classify::{Class, Subtests};
use super::enumerate::Package;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, BTreeSet};
use std::fs::{self, File, OpenOptions};
use std::io::Write;
use std::path::Path;

pub const SCHEMA: &str = "gomad.rust-wasm-sweep/v2";
pub const OBSERVATION_STATEMENT: &str = "Observation on one candidate only. A pass is an observed outcome of one seed under a bounded budget, not support or qualification.";
pub type Candidate = BTreeMap<String, Value>;

#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct Header {
    pub statement: String,
    pub candidate: Candidate,
    pub host: Value,
    pub packages: Vec<Package>,
    pub options: Value,
    pub harness_sha256: String,
}

impl Header {
    pub fn key(&self) -> Result<String, String> {
        let bytes = serde_json::to_vec(&(
            &self.candidate,
            &self.packages,
            &self.options,
            &self.harness_sha256,
        ))
        .map_err(|e| e.to_string())?;
        Ok(format!("sha256:{:x}", Sha256::digest(bytes)))
    }
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct TestEntry {
    pub package: String,
    pub name: String,
    pub class: Class,
    pub reason: String,
    pub skipped: bool,
    pub subtests: Option<Subtests>,
    pub config_source: String,
    pub module_sha256: String,
    pub attempts: usize,
    pub first_attempt_reason: String,
    pub gomad_exit: i32,
    pub wall_seconds: f64,
    pub preparation_bytes: u64,
    pub artifact_bytes: u64,
    pub engine_cache_bytes: u64,
    pub engine_entries_added: usize,
    pub engine_cache_hit: bool,
    pub artifact_retained: bool,
    pub artifact_path: String,
    pub retention_note: String,
}

impl TestEntry {
    pub fn new(
        package: impl Into<String>,
        name: impl Into<String>,
        class: Class,
        reason: impl Into<String>,
    ) -> Self {
        Self {
            package: package.into(),
            name: name.into(),
            class,
            reason: reason.into(),
            skipped: false,
            subtests: None,
            config_source: String::new(),
            module_sha256: String::new(),
            attempts: 0,
            first_attempt_reason: String::new(),
            gomad_exit: 0,
            wall_seconds: 0.0,
            preparation_bytes: 0,
            artifact_bytes: 0,
            engine_cache_bytes: 0,
            engine_entries_added: 0,
            engine_cache_hit: false,
            artifact_retained: false,
            artifact_path: String::new(),
            retention_note: String::new(),
        }
    }
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(try_from = "EncodedRow", into = "EncodedRow")]
pub struct Row {
    pub schema: String,
    pub candidate: String,
    pub entry: RowEntry,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(tag = "kind", rename_all = "snake_case", deny_unknown_fields)]
pub enum RowEntry {
    Header { header: Header },
    Test { test: TestEntry },
    Warmup { test: TestEntry },
}

#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct EncodedRow {
    schema: String,
    candidate: String,
    kind: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    header: Option<Header>,
    #[serde(skip_serializing_if = "Option::is_none")]
    test: Option<TestEntry>,
}

impl TryFrom<EncodedRow> for Row {
    type Error = String;
    fn try_from(row: EncodedRow) -> Result<Self, String> {
        let entry = match (row.kind.as_str(), row.header, row.test) {
            ("header", Some(header), None) => RowEntry::Header { header },
            ("test", None, Some(test)) => RowEntry::Test { test },
            ("warmup", None, Some(test)) => RowEntry::Warmup { test },
            _ => return Err("journal row must contain exactly its kind's payload".into()),
        };
        Ok(Self {
            schema: row.schema,
            candidate: row.candidate,
            entry,
        })
    }
}

impl From<Row> for EncodedRow {
    fn from(row: Row) -> Self {
        let (kind, header, test) = match row.entry {
            RowEntry::Header { header } => ("header", Some(header), None),
            RowEntry::Test { test } => ("test", None, Some(test)),
            RowEntry::Warmup { test } => ("warmup", None, Some(test)),
        };
        Self {
            schema: row.schema,
            candidate: row.candidate,
            kind: kind.into(),
            header,
            test,
        }
    }
}

pub struct Journal {
    file: File,
    rows: Vec<Row>,
    header: Header,
    key: String,
}

fn decode_rows(data: &[u8]) -> Result<(Vec<Row>, usize, bool), String> {
    let mut rows = Vec::new();
    let mut start = 0;
    for (index, line) in data.split_inclusive(|byte| *byte == b'\n').enumerate() {
        let complete = line.last() == Some(&b'\n');
        let value: Value = match serde_json::from_slice(line) {
            Ok(value) => value,
            Err(error) if !complete && error.is_eof() => return Ok((rows, start, true)),
            Err(error) => return Err(format!("journal line {} is malformed: {error}", index + 1)),
        };
        if value.get("schema").and_then(Value::as_str) != Some(SCHEMA) {
            return Err(format!(
                "journal line {} has an incompatible schema",
                index + 1
            ));
        }
        let row: Row = serde_json::from_value(value)
            .map_err(|e| format!("journal line {}: {e}", index + 1))?;
        rows.push(row);
        start += line.len();
    }
    Ok((rows, start, false))
}

fn validate_rows(rows: &[Row], expected: Option<&Header>) -> Result<Header, String> {
    let Some(RowEntry::Header { header: first }) = rows.first().map(|row| &row.entry) else {
        return Err("journal has no initial header".into());
    };
    if first.statement != OBSERVATION_STATEMENT
        || candidate_missing(&first.candidate).is_some()
        || !valid_options(&first.options)
        || !valid_digest(&first.harness_sha256)
        || first.candidate.get("config_source") != first.options.get("config_source")
        || first.host.as_object().is_none_or(serde_json::Map::is_empty)
    {
        return Err("journal header has incomplete identity or observation statement".into());
    }
    let key = first.key()?;
    if let Some(expected) = expected
        && (expected.key()? != key || expected.statement != first.statement)
    {
        return Err(
            "journal candidate, inventory, options or harness does not match this invocation"
                .into(),
        );
    }
    let inventory = inventory(&first.packages)?;
    for row in rows {
        if row.schema != SCHEMA || row.candidate != key {
            return Err("journal row candidate or schema mismatch".into());
        }
        match &row.entry {
            RowEntry::Header { header } => {
                if header.key()? != key
                    || header.statement != first.statement
                    || header
                        .host
                        .as_object()
                        .is_none_or(serde_json::Map::is_empty)
                {
                    return Err("journal repeated header mismatch".into());
                }
            }
            RowEntry::Test { test } => {
                validate_test(test, &inventory)?;
                validate_config_source(test, first)?;
            }
            RowEntry::Warmup { test } => validate_warmup(test, first)?,
        }
    }
    Ok(first.clone())
}

impl Journal {
    pub fn open(path: impl AsRef<Path>, header: &Header) -> Result<Self, String> {
        let data = match fs::read(path.as_ref()) {
            Ok(data) => data,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => Vec::new(),
            Err(error) => return Err(error.to_string()),
        };
        let (rows, retained, torn) = decode_rows(&data)?;
        if !rows.is_empty() {
            validate_rows(&rows, Some(header))?;
        } else if !data.is_empty() {
            return Err("journal has no complete native header".into());
        }
        let key = header.key()?;
        let row = Row {
            schema: SCHEMA.into(),
            candidate: key.clone(),
            entry: RowEntry::Header {
                header: header.clone(),
            },
        };
        let mut checked = rows.clone();
        checked.push(row.clone());
        validate_rows(&checked, Some(header))?;
        let file = OpenOptions::new()
            .create(true)
            .append(true)
            .open(path.as_ref())
            .map_err(|error| error.to_string())?;
        if torn {
            file.set_len(retained as u64)
                .map_err(|error| error.to_string())?;
            file.sync_all().map_err(|error| error.to_string())?;
        }
        let mut journal = Self {
            file,
            rows,
            header: header.clone(),
            key,
        };
        if !torn && !data.is_empty() && data.last() != Some(&b'\n') {
            journal
                .file
                .write_all(b"\n")
                .map_err(|error| error.to_string())?;
        }
        journal.append(row)?;
        Ok(journal)
    }

    fn append(&mut self, row: Row) -> Result<(), String> {
        let mut bytes = serde_json::to_vec(&row).map_err(|e| e.to_string())?;
        bytes.push(b'\n');
        self.file
            .write_all(&bytes)
            .map_err(|error| error.to_string())?;
        self.file.sync_all().map_err(|error| error.to_string())?;
        self.rows.push(row);
        Ok(())
    }

    pub fn append_test(&mut self, test: TestEntry) -> Result<(), String> {
        validate_test(&test, &inventory(&self.header.packages)?)?;
        validate_config_source(&test, &self.header)?;
        self.append(Row {
            schema: SCHEMA.into(),
            candidate: self.key.clone(),
            entry: RowEntry::Test { test },
        })
    }

    pub fn append_warmup(&mut self, test: TestEntry) -> Result<(), String> {
        validate_warmup(&test, &self.header)?;
        self.append(Row {
            schema: SCHEMA.into(),
            candidate: self.key.clone(),
            entry: RowEntry::Warmup { test },
        })
    }

    pub fn rows(&self) -> &[Row] {
        &self.rows
    }

    pub fn latest(&self) -> BTreeMap<(String, String), TestEntry> {
        latest(&self.rows)
    }
}

fn latest(rows: &[Row]) -> BTreeMap<(String, String), TestEntry> {
    let mut entries = BTreeMap::new();
    for row in rows {
        if let RowEntry::Test { test } = &row.entry {
            entries.insert((test.package.clone(), test.name.clone()), test.clone());
        }
    }
    entries
}

fn inventory(packages: &[Package]) -> Result<BTreeSet<(String, String)>, String> {
    let mut names = BTreeSet::new();
    let mut patterns = BTreeSet::new();
    for package in packages {
        if package.pattern.is_empty() || !patterns.insert(&package.pattern) {
            return Err("duplicate or empty package inventory".into());
        }
        if package.list_count != package.parsed_count
            || package.parsed_count != package.top_level_tests.len()
        {
            return Err(format!("{}: enumeration not reconciled", package.pattern));
        }
        let top: BTreeSet<&str> = package.top_level_tests.iter().map(String::as_str).collect();
        if top.len() != package.top_level_tests.len() {
            return Err("duplicate top-level inventory".into());
        }
        let wrappers: BTreeSet<&str> = package.units.keys().map(String::as_str).collect();
        if wrappers != top {
            return Err(format!(
                "{}: registration wrappers do not match top-level inventory",
                package.pattern
            ));
        }
        let mut flattened = Vec::new();
        for (wrapper, units) in &package.units {
            if units.is_empty() {
                return Err(format!(
                    "{} {wrapper}: method has no units",
                    package.pattern
                ));
            }
            if units.windows(2).any(|pair| pair[0] >= pair[1]) {
                return Err(format!(
                    "{} {wrapper}: method units are not sorted and unique",
                    package.pattern
                ));
            }
            for unit in units {
                if unit != wrapper && !unit.starts_with(&format!("{wrapper}/")) {
                    return Err(format!(
                        "{} {unit}: method unit has wrong wrapper",
                        package.pattern
                    ));
                }
                if unit == wrapper && units.len() != 1 {
                    return Err(format!(
                        "{} {wrapper}: suite wrapper is a runnable unit",
                        package.pattern
                    ));
                }
                flattened.push(unit);
            }
        }
        if flattened != package.tests.iter().collect::<Vec<_>>() {
            return Err(format!(
                "{}: method units do not exactly match enumerated tests",
                package.pattern
            ));
        }
        declared_coverage(package, &top)?;
        for name in &package.tests {
            if name.is_empty() || !names.insert((package.pattern.clone(), name.clone())) {
                return Err(format!(
                    "{} {name}: duplicate or empty enumerated unit",
                    package.pattern
                ));
            }
        }
        if package
            .guest_excluded
            .keys()
            .any(|name| !package.tests.contains(name))
        {
            return Err("guest exclusion references unknown unit".into());
        }
    }
    Ok(names)
}

fn declared_coverage(package: &Package, top: &BTreeSet<&str>) -> Result<(), String> {
    let mut wrappers = BTreeSet::new();
    let mut registrations = BTreeSet::new();
    for registration in &package.registrations {
        if !top.contains(registration.wrapper.as_str()) {
            return Err(format!(
                "{} {}: registration wrapper not enumerated",
                package.pattern, registration.wrapper
            ));
        }
        if registration.suite.is_empty()
            || registration.file.is_empty()
            || (registration.path != registration.wrapper
                && !registration
                    .path
                    .starts_with(&format!("{}/", registration.wrapper)))
            || !registrations.insert((registration.wrapper.as_str(), registration.path.as_str()))
        {
            return Err(format!("{}: invalid suite registration", package.pattern));
        }
        wrappers.insert(registration.wrapper.as_str());
        if package.tests.contains(&registration.wrapper) {
            return Err(format!(
                "{} {}: suite wrapper is a runnable unit",
                package.pattern, registration.wrapper
            ));
        }
    }
    let mut covered = BTreeSet::new();
    let mut declarations = BTreeSet::new();
    for method in &package.methods {
        if method.suite.is_empty()
            || !method.name.starts_with("Test")
            || method.name.contains('/')
            || method.file.is_empty()
            || !declarations.insert((method.suite.as_str(), method.name.as_str()))
        {
            return Err(format!(
                "{}: invalid or duplicate method declaration",
                package.pattern
            ));
        }
        if method.units.is_empty() {
            return Err(format!(
                "{} {}.{}: declared method has no units",
                package.pattern, method.suite, method.name
            ));
        }
        if method.units.windows(2).any(|pair| pair[0] >= pair[1]) {
            return Err(format!(
                "{}: declared method units are not sorted and unique",
                package.pattern
            ));
        }
        for unit in &method.units {
            if !package.tests.contains(unit) {
                return Err(format!(
                    "{} {unit}: method unit not enumerated",
                    package.pattern
                ));
            }
            if !package
                .registrations
                .iter()
                .any(|registration| unit.starts_with(&format!("{}/", registration.path)))
                || !unit.split('/').any(|segment| segment == method.name)
            {
                return Err(format!(
                    "{} {unit}: method unit has no registration or declared method selector",
                    package.pattern
                ));
            }
            covered.insert(unit.as_str());
        }
    }
    registration_coverage(package)?;
    for name in &package.top_level_tests {
        if !wrappers.contains(name.as_str()) {
            covered.insert(name.as_str());
        }
    }
    for unit in &package.tests {
        if !covered.contains(unit.as_str()) {
            return Err(format!(
                "{} {unit}: unit has no source declaration",
                package.pattern
            ));
        }
    }
    for name in &package.top_level_tests {
        if !wrappers.contains(name.as_str())
            && package.units[name].as_slice() != std::slice::from_ref(name)
        {
            return Err(format!(
                "{} {name}: standalone test has no unit",
                package.pattern
            ));
        }
    }
    Ok(())
}

fn registration_coverage(package: &Package) -> Result<(), String> {
    for registration in &package.registrations {
        if registration.methods.is_empty()
            || registration
                .methods
                .windows(2)
                .any(|pair| pair[0] >= pair[1])
        {
            return Err(format!(
                "{}: registration has empty or unordered effective methods",
                package.pattern
            ));
        }
        for method in &registration.methods {
            let selector = format!("{}/{method}", registration.path);
            if !method.starts_with("Test")
                || method.contains('/')
                || !package
                    .tests
                    .iter()
                    .any(|unit| unit == &selector || unit.starts_with(&format!("{selector}/")))
            {
                return Err(format!(
                    "{} {selector}: registration effective method has no selected unit",
                    package.pattern
                ));
            }
        }
    }
    Ok(())
}

fn valid_digest(value: &str) -> bool {
    value.strip_prefix("sha256:").is_some_and(|hex| {
        hex.len() == 64
            && hex
                .bytes()
                .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
    })
}

fn validate_config_source(test: &TestEntry, header: &Header) -> Result<(), String> {
    if header
        .candidate
        .get("config_source")
        .and_then(Value::as_str)
        != Some(test.config_source.as_str())
    {
        return Err(format!(
            "{} {}: config_source differs from candidate",
            test.package, test.name
        ));
    }
    Ok(())
}

fn validate_warmup(test: &TestEntry, header: &Header) -> Result<(), String> {
    if !header
        .packages
        .iter()
        .any(|package| package.pattern == test.package)
        || test.name != "(warm-up)"
        || test.class != Class::NotRun
        || !(test.reason == "warm-up only; not a test row"
            || test.reason.starts_with("warm-up only; not a test row: "))
        || test.skipped
        || test.subtests.is_some()
        || !test.wall_seconds.is_finite()
        || test.wall_seconds < 0.0
    {
        return Err("invalid warm-up observation".into());
    }
    validate_config_source(test, header)
}

fn valid_options(options: &Value) -> bool {
    let Some(options) = options.as_object() else {
        return false;
    };
    options.len() == 4
        && options.get("max_tests").and_then(Value::as_u64).is_some()
        && options
            .get("retain_bytes")
            .and_then(Value::as_u64)
            .is_some()
        && options
            .get("overall_budget_nanos")
            .and_then(Value::as_u64)
            .is_some_and(|value| value > 0)
        && options
            .get("config_source")
            .and_then(Value::as_str)
            .is_some_and(|value| !value.is_empty())
}

fn candidate_missing(candidate: &Candidate) -> Option<String> {
    if candidate.len() != 20 {
        return Some("candidate fields".into());
    }
    for field in [
        "git_sha",
        "source_sha256",
        "gomad_sha256",
        "helper_sha256",
        "compiler_sha256",
        "go_version",
        "runtime_implementation_sha256",
        "model_sha256",
        "guest_config_sha256",
        "engine",
        "profile",
        "config_source",
    ] {
        if candidate
            .get(field)
            .and_then(Value::as_str)
            .is_none_or(|value| value.trim().is_empty())
        {
            return Some(field.into());
        }
    }
    for (field, value) in candidate {
        if field.ends_with("_sha256") && !value.as_str().is_some_and(valid_digest) {
            return Some(field.clone());
        }
    }
    for field in [
        "memory_bytes",
        "fuel",
        "per_test_timeout_nanos",
        "prepare_timeout_nanos",
    ] {
        if candidate
            .get(field)
            .and_then(Value::as_u64)
            .is_none_or(|value| value == 0)
        {
            return Some(field.into());
        }
    }
    if candidate.get("seed").and_then(Value::as_u64).is_none() {
        return Some("seed".into());
    }
    for field in ["build_tags", "explore_flags", "test_flags"] {
        if !candidate
            .get(field)
            .and_then(Value::as_array)
            .is_some_and(|values| {
                !values.is_empty()
                    && values
                        .iter()
                        .all(|value| value.as_str().is_some_and(|value| !value.is_empty()))
            })
        {
            return Some(field.into());
        }
    }
    None
}

fn class_name(class: Class) -> Result<String, String> {
    serde_json::to_value(class)
        .map_err(|e| e.to_string())?
        .as_str()
        .map(str::to_owned)
        .ok_or_else(|| "class must be a string".into())
}

fn validate_test(test: &TestEntry, names: &BTreeSet<(String, String)>) -> Result<(), String> {
    if !names.contains(&(test.package.clone(), test.name.clone())) {
        return Err(format!(
            "{} {}: row not enumerated",
            test.package, test.name
        ));
    }
    let class = class_name(test.class)?;
    if !matches!(
        class.as_str(),
        "pass"
            | "guest-failure"
            | "refused-import"
            | "capacity-exceeded"
            | "timeout"
            | "build-failure"
            | "infrastructure"
            | "replay-divergence"
            | "not-run"
    ) {
        return Err("invalid class".into());
    }
    if class != "pass" && test.reason.trim().is_empty() {
        return Err(format!(
            "{} {}: non-pass without a reason",
            test.package, test.name
        ));
    }
    if test.skipped && class != "pass" {
        return Err("skipped is only valid on pass".into());
    }
    if !test.wall_seconds.is_finite() || test.wall_seconds < 0.0 {
        return Err("invalid wall_seconds".into());
    }
    Ok(())
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Sidecar {
    pub schema: String,
    pub statement: String,
    pub candidate: Candidate,
    pub candidate_key: String,
    pub hosts: Vec<Value>,
    pub packages: Vec<Package>,
    pub options: Value,
    pub harness_sha256: String,
    pub rows: Vec<TestEntry>,
    pub by_class: BTreeMap<String, usize>,
    pub by_package: BTreeMap<String, BTreeMap<String, usize>>,
    pub reasons: Vec<ReasonCount>,
}

#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub struct ReasonCount {
    pub class: String,
    pub reason: String,
    pub count: usize,
}

#[expect(
    clippy::type_complexity,
    reason = "The tuple mirrors the three independently reconciled sidecar summaries."
)]
fn counters(
    rows: &[TestEntry],
) -> Result<
    (
        BTreeMap<String, usize>,
        BTreeMap<String, BTreeMap<String, usize>>,
        Vec<ReasonCount>,
    ),
    String,
> {
    let mut by_class = BTreeMap::new();
    let mut by_package: BTreeMap<String, BTreeMap<String, usize>> = BTreeMap::new();
    let mut reasons = BTreeMap::new();
    for row in rows {
        let class = class_name(row.class)?;
        *by_class.entry(class.clone()).or_default() += 1;
        *by_package
            .entry(row.package.clone())
            .or_default()
            .entry(class.clone())
            .or_default() += 1;
        if class != "pass" {
            *reasons.entry((class, row.reason.clone())).or_insert(0) += 1;
        }
    }
    let mut reasons: Vec<_> = reasons
        .into_iter()
        .map(|((class, reason), count)| ReasonCount {
            class,
            reason,
            count,
        })
        .collect();
    reasons.sort_by(|a, b| {
        b.count
            .cmp(&a.count)
            .then_with(|| a.class.cmp(&b.class))
            .then_with(|| a.reason.cmp(&b.reason))
    });
    Ok((by_class, by_package, reasons))
}

pub fn build_sidecar(rows: &[Row]) -> Result<Sidecar, String> {
    let header = validate_rows(rows, None)?;
    let mut latest = latest(rows);
    let mut entries = Vec::new();
    for (package, name) in inventory(&header.packages)? {
        entries.push(
            latest
                .remove(&(package.clone(), name.clone()))
                .unwrap_or_else(|| {
                    let mut entry = TestEntry::new(
                        package,
                        name,
                        Class::NotRun,
                        "not reached by this sweep (budget or --max-tests)",
                    );
                    entry.config_source = header.candidate["config_source"]
                        .as_str()
                        .unwrap_or_default()
                        .into();
                    entry
                }),
        );
    }
    let (by_class, by_package, reasons) = counters(&entries)?;
    let mut hosts = Vec::new();
    for row in rows {
        if let RowEntry::Header { header } = &row.entry
            && !hosts.contains(&header.host)
        {
            hosts.push(header.host.clone());
        }
    }
    Ok(Sidecar {
        schema: SCHEMA.into(),
        statement: OBSERVATION_STATEMENT.into(),
        candidate_key: header.key()?,
        candidate: header.candidate,
        hosts,
        packages: header.packages,
        options: header.options,
        harness_sha256: header.harness_sha256,
        rows: entries,
        by_class,
        by_package,
        reasons,
    })
}

pub fn verify(sidecar: &Sidecar) -> Vec<String> {
    let mut findings = Vec::new();
    if sidecar.schema != SCHEMA {
        findings.push("incompatible sidecar schema".into());
    }
    if sidecar.statement != OBSERVATION_STATEMENT {
        findings.push("observation statement missing or altered".into());
    }
    if sidecar.hosts.is_empty()
        || sidecar.hosts.iter().any(|host| {
            !host.is_object() || host.as_object().is_some_and(serde_json::Map::is_empty)
        })
    {
        findings.push("host identity missing".into());
    }
    let header = Header {
        statement: sidecar.statement.clone(),
        candidate: sidecar.candidate.clone(),
        host: sidecar.hosts.first().cloned().unwrap_or(Value::Null),
        packages: sidecar.packages.clone(),
        options: sidecar.options.clone(),
        harness_sha256: sidecar.harness_sha256.clone(),
    };
    if candidate_missing(&header.candidate).is_some()
        || !valid_digest(&header.harness_sha256)
        || header.candidate.get("config_source") != header.options.get("config_source")
        || !valid_options(&header.options)
    {
        findings.push("candidate identity or options missing".into());
    }
    if header.key().ok().as_ref() != Some(&sidecar.candidate_key) {
        findings.push("candidate key mismatch".into());
    }
    match inventory(&sidecar.packages) {
        Err(error) => findings.push(error),
        Ok(names) => {
            let mut seen = BTreeSet::new();
            for row in &sidecar.rows {
                if let Err(error) = validate_test(row, &names) {
                    findings.push(error);
                }
                if let Err(error) = validate_config_source(row, &header) {
                    findings.push(error);
                }
                if !seen.insert((row.package.clone(), row.name.clone())) {
                    findings.push("duplicate test row".into());
                }
            }
            if seen != names {
                findings.push("rows do not exactly cover enumerated inventory".into());
            }
        }
    }
    match counters(&sidecar.rows) {
        Ok((classes, packages, reasons)) => {
            if classes != sidecar.by_class {
                findings.push("by_class counters do not reconcile".into());
            }
            if packages != sidecar.by_package {
                findings.push("by_package counters do not reconcile".into());
            }
            if reasons != sidecar.reasons {
                findings.push("reason counters do not reconcile".into());
            }
        }
        Err(error) => findings.push(error),
    }
    findings
}

pub fn verify_file(path: impl AsRef<Path>) -> Result<Vec<String>, String> {
    let bytes = fs::read(path).map_err(|error| error.to_string())?;
    let sidecar: Sidecar =
        serde_json::from_slice(&bytes).map_err(|e| format!("decode sidecar: {e}"))?;
    Ok(verify(&sidecar))
}

pub fn write_sidecar(path: impl AsRef<Path>, sidecar: &Sidecar) -> Result<(), String> {
    let mut bytes = serde_json::to_vec_pretty(sidecar).map_err(|e| e.to_string())?;
    bytes.push(b'\n');
    let temporary = path.as_ref().with_extension("json.tmp");
    let mut file = File::create(&temporary).map_err(|error| error.to_string())?;
    file.write_all(&bytes).map_err(|error| error.to_string())?;
    file.sync_all().map_err(|error| error.to_string())?;
    fs::rename(temporary, path).map_err(|error| error.to_string())
}
