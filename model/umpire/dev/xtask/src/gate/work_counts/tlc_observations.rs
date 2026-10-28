//! Integrity-linked TLC process and assertion observations.
use serde::Deserialize;
use std::{
    collections::{BTreeMap, BTreeSet},
    fs, io::{self, Read},
    os::unix::{ffi::OsStrExt, fs::OpenOptionsExt},
    path::{Component, Path, PathBuf},
};
use umpire_backend_process::{bytes_sha256, structural_sha256};

const MAX_DOCUMENT: u64 = 4 * 1024 * 1024;
const RENDERER: &[u8] = include_bytes!("../../../../../backends/tlc/src/runner/fixture.rs");

fn invalid() -> io::Error { io::Error::other("invalid TLC observation authority") }
fn require(valid: bool) -> io::Result<()> { if valid { Ok(()) } else { Err(invalid()) } }
fn digest(value: &str) -> bool {
    value.len() == 64 && value.bytes().all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
}
fn attempt_id(value: &str) -> bool {
    value.split_once('-').is_some_and(|(pid, counter)| {
        !pid.is_empty() && !counter.is_empty()
            && pid.bytes().all(|b| b.is_ascii_digit())
            && counter.bytes().all(|b| b.is_ascii_digit())
            && pid.parse::<u64>().is_ok() && counter.parse::<u64>().is_ok()
    })
}
fn configuration(value: &str) -> bool { matches!(value, "probe" | "-") || digest(value) }
fn cause(value: &str) -> bool { matches!(value, "completed" | "refused" | "nonutf8" | "spawn-error") }
fn hex(bytes: &[u8]) -> String { bytes.iter().map(|b| format!("{b:02x}")).collect() }
fn unhex(value: &str) -> io::Result<Vec<u8>> {
    require(!value.is_empty() && value.len().is_multiple_of(2)
        && value.bytes().all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b)))?;
    value.as_bytes().chunks_exact(2).map(|pair| {
        u8::from_str_radix(std::str::from_utf8(pair).map_err(io::Error::other)?, 16).map_err(io::Error::other)
    }).collect()
}
fn path(value: &str) -> io::Result<PathBuf> {
    use std::os::unix::ffi::OsStringExt;
    let bytes = unhex(value)?;
    require(!bytes.contains(&0))?;
    let path = PathBuf::from(std::ffi::OsString::from_vec(bytes));
    require(path.is_absolute() && path.components().all(|c| matches!(c, Component::RootDir | Component::Normal(_))))?;
    let normalized: PathBuf = path.components().collect();
    require(normalized.as_os_str().as_bytes() == path.as_os_str().as_bytes())?;
    Ok(path)
}
fn regular_bytes(path: &Path, limit: u64) -> io::Result<Vec<u8>> {
    let mut current = PathBuf::new();
    for component in path.components() {
        require(matches!(component, Component::RootDir | Component::Normal(_)))?;
        current.push(component);
        let metadata = fs::symlink_metadata(&current)?;
        require(!metadata.is_symlink())?;
    }
    let file = fs::OpenOptions::new().read(true).custom_flags(libc::O_NOFOLLOW).open(path)?;
    let metadata = file.metadata()?;
    require(metadata.is_file() && metadata.len() <= limit)?;
    let mut bytes = Vec::new();
    file.take(limit + 1).read_to_end(&mut bytes)?;
    require(bytes.len() as u64 <= limit)?;
    Ok(bytes)
}

#[derive(Clone)]
struct Reference { path: String, sha: String }
impl Reference {
    fn new(path: &str, sha: &str) -> io::Result<Self> {
        self::path(path)?;
        require(digest(sha))?;
        Ok(Self { path: path.into(), sha: sha.into() })
    }
    fn document<T: serde::de::DeserializeOwned>(&self, scope: &Scope, basename: &str) -> io::Result<T> {
        let path = path(&self.path)?;
        require(path == scope.authorities.join(basename))?;
        let bytes = regular_bytes(&path, MAX_DOCUMENT)?;
        require(bytes_sha256(&bytes) == self.sha)?;
        serde_json::from_slice(&bytes).map_err(io::Error::other)
    }
}
struct Scope { authorities: PathBuf, sha: String }
impl Scope {
    fn from_work(directory: &Path) -> io::Result<Self> {
        require(directory.is_absolute() && directory.canonicalize()? == directory)?;
        let stage = directory.file_name().and_then(|s| s.to_str()).ok_or_else(invalid)?;
        let parent = directory.parent().ok_or_else(invalid)?;
        let scope = parent.file_name().and_then(|s| s.to_str()).ok_or_else(invalid)?;
        let work = parent.parent().ok_or_else(invalid)?;
        require(work.file_name().is_some_and(|s| s == "work"))?;
        let receipts = work.parent().ok_or_else(invalid)?;
        let sha = structural_sha256(&(hex(receipts.as_os_str().as_bytes()), scope, stage));
        Ok(Self { authorities: receipts.join("authorities").join(scope).join(stage), sha })
    }
}
struct Attempt { version: u8, origin: String, configuration: String, identity: String }
struct Terminal { version: u8, outcome: String, cause: String, reference: Option<Reference> }

#[derive(Default)]
pub(super) struct Collector {
    attempts: BTreeMap<String, Attempt>,
    terminals: BTreeMap<String, Terminal>,
    completions: BTreeMap<String, Reference>,
    passed: BTreeSet<(String, String)>,
}
impl Collector {
    pub(super) fn ingest(&mut self, fields: &[&str]) -> io::Result<bool> {
        if !fields.get(1).is_some_and(|name| name.starts_with("tlc-")) { return Ok(false); }
        match fields {
            ["1", "tlc-attempt", id, config] if attempt_id(id) && configuration(config) => {
                require(self.attempts.insert((*id).into(), Attempt {
                    version: 1, origin: "legacy".into(), configuration: (*config).into(), identity: String::new(),
                }).is_none())?;
            }
            ["2", "tlc-attempt", id, origin, config, identity]
                if attempt_id(id) && matches!(*origin, "admitted" | "fixture") && configuration(config) && digest(identity) => {
                require(self.attempts.insert((*id).into(), Attempt {
                    version: 2, origin: (*origin).into(), configuration: (*config).into(), identity: (*identity).into(),
                }).is_none())?;
            }
            ["1", "tlc-finish", id, started] if attempt_id(id) && matches!(*started, "0" | "1") => {
                require(self.terminals.insert((*id).into(), Terminal {
                    version: 1, outcome: if *started == "1" { "started" } else { "unknown" }.into(), cause: String::new(), reference: None,
                }).is_none())?;
            }
            ["2", "tlc-finish", id, outcome, why, reference, sha]
                if attempt_id(id) && matches!(*outcome, "started" | "unknown" | "fixture") && cause(why) => {
                let reference = if *outcome == "fixture" {
                    Some(Reference::new(reference, sha)?)
                } else { require(*reference == "-" && *sha == "-")?; None };
                require(self.terminals.insert((*id).into(), Terminal {
                    version: 2, outcome: (*outcome).into(), cause: (*why).into(), reference,
                }).is_none())?;
            }
            ["2", "tlc-fixture-complete", id, reference, sha] if digest(id) => {
                require(self.completions.insert((*id).into(), Reference::new(reference, sha)?).is_none())?;
            }
            _ => return Err(invalid()),
        }
        Ok(true)
    }
    pub(super) fn record_pass(&mut self, name: String, binary_sha: String) -> io::Result<()> {
        require(digest(&binary_sha) && name.split_once('$').is_some_and(|(binary, test)| !binary.is_empty() && !test.is_empty()))?;
        require(self.passed.insert((name, binary_sha)))
    }
    pub(super) fn finish(&self, directory: &Path) -> io::Result<(Option<u64>, Option<u64>)> {
        require(self.attempts.keys().eq(self.terminals.keys()))?;
        let scope = if self.completions.is_empty() && !self.attempts.values().any(|a| a.origin == "fixture") {
            None
        } else { Some(Scope::from_work(directory)?) };
        let mut fixtures = BTreeMap::new();
        let mut configurations = BTreeSet::new();
        let mut launches = 0;
        let mut unknown = false;
        let mut configuration_unknown = false;
        for (id, attempt) in &self.attempts {
            let terminal = &self.terminals[id];
            require(attempt.version == terminal.version)?;
            if attempt.origin == "fixture" {
                require(terminal.outcome == "fixture")?;
                let scope = scope.as_ref().ok_or_else(invalid)?;
                let reference = terminal.reference.as_ref().ok_or_else(invalid)?;
                let doc: Fixture = reference.document(scope, &format!("{id}.json"))?;
                doc.validate(id, attempt, terminal, scope, &self.passed)?;
                fixtures.insert(id.clone(), (doc, reference));
            } else {
                require(terminal.outcome != "fixture" && terminal.reference.is_none())?;
                if terminal.outcome == "unknown" { unknown = true; }
                else {
                    launches += 1;
                    match attempt.configuration.as_str() {
                        "probe" => (), "-" => configuration_unknown = true,
                        config => { configurations.insert(config); }
                    }
                }
            }
        }
        let mut assigned = BTreeSet::new();
        for (id, reference) in &self.completions {
            let scope = scope.as_ref().ok_or_else(invalid)?;
            let doc: Completion = reference.document(scope, &format!("assertions-{id}.json"))?;
            require(doc.version == 1 && doc.kind == "tlc-fixture-assertions" && doc.assertion_id == *id
                && doc.scope_sha256 == scope.sha && doc.outcome == "assertions-completed" && !doc.attempts.is_empty())?;
            for entry in &doc.attempts {
                require(assigned.insert(entry.attempt.clone()))?;
                let (fixture, terminal_reference) = fixtures.get(&entry.attempt).ok_or_else(invalid)?;
                require(fixture.assertion_id == *id && fixture.test_hex == doc.test_hex
                    && fixture.test_executable_sha256 == doc.test_executable_sha256
                    && entry.path_hex == terminal_reference.path && entry.sha256 == terminal_reference.sha)?;
            }
        }
        require(assigned.len() == fixtures.len())?;
        if self.attempts.is_empty() || unknown { return Ok((None, None)); }
        Ok((Some(launches), (!configuration_unknown).then_some(configurations.len() as u64)))
    }
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Artifact { path_hex: String, sha256: String }
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Data { path_hex: String, sha256: String, bytes_hex: String }
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Fixture {
    version: u8, kind: String, attempt: String, assertion_id: String, scope_sha256: String,
    configuration: String, family: String, renderer_sha256: String, case_hex: String,
    script_path_hex: String, script_sha256: String, script_hex: String,
    interpreter_path_hex: String, interpreter_sha256: String, tools: Vec<Artifact>, data: Vec<Data>,
    test_hex: String, test_executable_path_hex: String, test_executable_sha256: String,
    environment_policy: String, cause: String,
    #[serde(deserialize_with = "required_exit")]
    exit: Option<i32>,
}
fn required_exit<'de, D: serde::Deserializer<'de>>(deserializer: D) -> Result<Option<i32>, D::Error> {
    Option::<i32>::deserialize(deserializer)
}
impl Fixture {
    fn validate(&self, id: &str, attempt: &Attempt, terminal: &Terminal, scope: &Scope,
        passed: &BTreeSet<(String, String)>) -> io::Result<()> {
        require(self.version == 1 && self.kind == "closed-tlc-protocol-fixture" && self.attempt == id
            && digest(&self.assertion_id) && self.scope_sha256 == scope.sha
            && self.configuration == attempt.configuration && self.cause == terminal.cause
            && matches!(self.family.as_str(), "tlc-batch-v1" | "tlc-cache-v1")
            && self.renderer_sha256 == bytes_sha256(RENDERER)
            && self.environment_policy == "clean-path-c-v1")?;
        let script = unhex(&self.script_hex)?;
        let case = String::from_utf8(unhex(&self.case_hex)?).map_err(io::Error::other)?;
        require(script.starts_with(b"#!/bin/bash\nset -eu\n") && self.script_sha256 == bytes_sha256(&script))?;
        require(if self.family == "tlc-cache-v1" { case == "Cache" } else {
            ["Protocol { ", "Resource { ", "WorkerPeak { ", "Recovery { "].iter().any(|prefix| case.starts_with(prefix)) && case.ends_with(" }")
        })?;
        let script_path = path(&self.script_path_hex)?;
        require(script_path.file_name().is_some_and(|s| s == "java"))?;
        let owner = script_path.parent().ok_or_else(invalid)?;
        let interpreter = path(&self.interpreter_path_hex)?;
        require(interpreter == Path::new("/bin/bash").canonicalize()?
            && digest(&self.interpreter_sha256)
            && bytes_sha256(&regular_bytes(&interpreter, 256 * 1024 * 1024)?) == self.interpreter_sha256)?;
        require(self.tools.len() == 3)?;
        for (tool, expected) in self.tools.iter().zip(["/usr/bin/dirname", "/usr/bin/cat", "/usr/bin/grep"]) {
            let tool_path = path(&tool.path_hex)?;
            require(tool_path == Path::new(expected).canonicalize()? && digest(&tool.sha256)
                && bytes_sha256(&regular_bytes(&tool_path, 256 * 1024 * 1024)?) == tool.sha256)?;
        }
        let expected_data: &[&str] = if self.family == "tlc-cache-v1" { &["mode", "java-version", "tlc.jar"] } else { &[] };
        require(self.data.len() == expected_data.len())?;
        for (data, name) in self.data.iter().zip(expected_data) {
            let bytes = if data.bytes_hex.is_empty() { Vec::new() } else { unhex(&data.bytes_hex)? };
            require(path(&data.path_hex)? == owner.join(name) && data.sha256 == bytes_sha256(&bytes))?;
            if *name == "mode" {
                require([b"verified".as_slice(), b"identity-failure", b"failure", b"module-failure", b"bad-dump", b"evaluation", b"refuted", b"mixed"].contains(&bytes.as_slice()))?;
            }
        }
        let test = String::from_utf8(unhex(&self.test_hex)?).map_err(io::Error::other)?;
        path(&self.test_executable_path_hex)?;
        require(digest(&self.test_executable_sha256) && passed.iter().filter(|(name, sha)| {
            sha == &self.test_executable_sha256 && name.split_once('$').is_some_and(|(_, observed)| observed == test)
        }).count() == 1)?;
        require(attempt.identity == structural_sha256(&(&self.assertion_id, &self.family,
            &self.renderer_sha256, &self.case_hex, &self.script_sha256, &self.interpreter_sha256)))?;
        require(self.exit.is_none_or(|code| (0..=255).contains(&code)))
    }
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct CompletionEntry { attempt: String, path_hex: String, sha256: String }
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Completion {
    version: u8, kind: String, assertion_id: String, scope_sha256: String,
    test_hex: String, test_executable_sha256: String, outcome: String, attempts: Vec<CompletionEntry>,
}

#[cfg(test)]
mod tests {
    use super::*;

    fn observe(lines: &[&str]) -> io::Result<(Option<u64>, Option<u64>)> {
        let mut collector = Collector::default();
        for line in lines {
            assert!(collector.ingest(&line.split('\t').collect::<Vec<_>>())?);
        }
        collector.finish(Path::new("/unused"))
    }

    #[test]
    fn counts_confirmed_v1_launches_and_deduplicates_configurations() -> io::Result<()> {
        let sha = "a".repeat(64);
        assert_eq!(observe(&[
            &format!("1\ttlc-attempt\t100-0\t{sha}"), "1\ttlc-finish\t100-0\t1",
            &format!("1\ttlc-attempt\t100-1\t{sha}"), "1\ttlc-finish\t100-1\t1",
        ])?, (Some(2), Some(1)));
        Ok(())
    }

    #[test]
    fn counts_admitted_v2_but_preserves_unknown_legacy_completion() -> io::Result<()> {
        let identity = "b".repeat(64);
        assert_eq!(observe(&[
            &format!("2\ttlc-attempt\t100-0\tadmitted\tprobe\t{identity}"),
            "2\ttlc-finish\t100-0\tstarted\tcompleted\t-\t-",
        ])?, (Some(1), Some(0)));
        assert_eq!(observe(&["1\ttlc-attempt\t100-0\t-", "1\ttlc-finish\t100-0\t0"])?, (None, None));
        Ok(())
    }

    #[test]
    fn rejects_missing_duplicate_mixed_and_unknown_records() {
        for lines in [
            vec!["1\ttlc-attempt\t100-0\tprobe"],
            vec!["1\ttlc-finish\t100-0\t1"],
            vec!["1\ttlc-attempt\t100-0\tprobe", "1\ttlc-attempt\t100-0\tprobe"],
            vec!["1\ttlc-attempt\t100-0\tprobe", "2\ttlc-finish\t100-0\tstarted\tcompleted\t-\t-"],
            vec!["3\ttlc-attempt\t100-0\tprobe"],
            vec!["2\ttlc-forged\t100-0"],
        ] {
            assert!(observe(&lines).is_err(), "accepted {lines:?}");
        }
    }

    #[test]
    fn empty_observations_do_not_prove_zero() -> io::Result<()> {
        assert_eq!(Collector::default().finish(Path::new("/unused"))?, (None, None));
        assert!(!Collector::default().ingest(&["1", "tests", "other"])?);
        Ok(())
    }
    #[test]
    fn refuses_fixture_without_assertions_and_observed_parent_pass() -> io::Result<()> {
        let mut collector = Collector::default();
        assert!(collector.ingest(&["2", "tlc-attempt", "100-0", "fixture", "probe", &"b".repeat(64)])?);
        assert!(collector.ingest(&["2", "tlc-finish", "100-0", "fixture", "completed", "2f746d702f3130302d302e6a736f6e", &"c".repeat(64)])?);
        assert!(collector.finish(Path::new("/tmp/receipts/work/scope/tests")).is_err());
        Ok(())
    }

    #[test]
    fn rejects_invalid_record_id_configuration_origin_cause_and_identity() {
        for fields in [
            vec!["1", "tlc-attempt", "bad", "probe"],
            vec!["1", "tlc-attempt", "100-0", "bad"],
            vec!["2", "tlc-attempt", "100-0", "other", "probe", "bad"],
            vec!["2", "tlc-finish", "100-0", "started", "other", "-", "-"],
            vec!["2", "tlc-fixture-complete", "bad", "bad", "bad"],
            vec!["2", "tlc-attempt"],
        ] {
            assert!(Collector::default().ingest(&fields).is_err(), "accepted {fields:?}");
        }
    }

    fn fixture_counts(family: &str, pass: bool, unknown: bool,
        modify: impl FnOnce(&mut serde_json::Value, &mut serde_json::Value)) -> io::Result<(Option<u64>, Option<u64>)> {
        use serde_json::json;
        let scratch = tempfile::tempdir()?;
        let receipts = scratch.path().canonicalize()?;
        let directory = receipts.join("work/scope/tests");
        fs::create_dir_all(&directory)?;
        let scope = Scope::from_work(&directory)?;
        fs::create_dir_all(&scope.authorities)?;
        let owner = receipts.join("cleaned-scratch");
        let artifact = |name: &str| -> io::Result<serde_json::Value> {
            let resolved = Path::new(name).canonicalize()?;
            Ok(json!({"path_hex": hex(resolved.as_os_str().as_bytes()), "sha256": bytes_sha256(&fs::read(resolved)?)}))
        };
        let executable = std::env::current_exe()?.canonicalize()?;
        let executable_sha = bytes_sha256(&fs::read(&executable)?);
        let interpreter = artifact("/bin/bash")?;
        let assertion = "a".repeat(64);
        let renderer = bytes_sha256(RENDERER);
        let script = b"#!/bin/bash\nset -eu\nprintf fixture\n";
        let test_name = if family == "tlc-cache-v1" { "runner::cache_tests::populated_cache_hits_and_authoritative_disable_runs_fresh" } else { "runner::batch_tests::batch_protocol_requires_complete_selected_evidence" };
        let case = if family == "tlc-cache-v1" { "Cache" } else { "Protocol { output: \"fixture\", status: 0 }" };
        let data: Vec<_> = if family == "tlc-cache-v1" {
            ["mode", "java-version", "tlc.jar"].into_iter().map(|name| {
                let bytes = if name == "mode" { b"verified".as_slice() } else { b"fixture" };
                json!({"path_hex":hex(owner.join(name).as_os_str().as_bytes()), "sha256":bytes_sha256(bytes), "bytes_hex":hex(bytes)})
            }).collect()
        } else { vec![] };
        let mut authority = json!({
            "version":1,"kind":"closed-tlc-protocol-fixture","attempt":"100-0","assertion_id":assertion,
            "scope_sha256":scope.sha,"configuration":"probe","family":family,"renderer_sha256":renderer,
            "case_hex":hex(case.as_bytes()),"script_path_hex":hex(owner.join("java").as_os_str().as_bytes()),
            "script_sha256":bytes_sha256(script),"script_hex":hex(script),
            "interpreter_path_hex":interpreter["path_hex"],"interpreter_sha256":interpreter["sha256"],
            "tools":[artifact("/usr/bin/dirname")?,artifact("/usr/bin/cat")?,artifact("/usr/bin/grep")?],"data":data,
            "test_hex":hex(test_name.as_bytes()),"test_executable_path_hex":hex(executable.as_os_str().as_bytes()),
            "test_executable_sha256":executable_sha,"environment_policy":"clean-path-c-v1","cause":"completed","exit":0
        });
        let authority_path = scope.authorities.join("100-0.json");
        let mut completion = json!({"version":1,"kind":"tlc-fixture-assertions","assertion_id":assertion,
            "scope_sha256":scope.sha,"test_hex":hex(test_name.as_bytes()),"test_executable_sha256":executable_sha,
            "outcome":"assertions-completed","attempts":[]});
        modify(&mut authority, &mut completion);
        let bytes = serde_json::to_vec(&authority)?;
        fs::write(&authority_path, &bytes)?;
        let authority_sha = bytes_sha256(&bytes);
        if completion["attempts"] == json!([]) {
            completion["attempts"] = json!([{"attempt":"100-0","path_hex":hex(authority_path.as_os_str().as_bytes()),"sha256":authority_sha}]);
        }
        let completion_path = scope.authorities.join(format!("assertions-{assertion}.json"));
        let bytes = serde_json::to_vec(&completion)?;
        fs::write(&completion_path, &bytes)?;
        let identity = structural_sha256(&(&assertion, &family, &renderer, &hex(case.as_bytes()),
            &bytes_sha256(script), interpreter["sha256"].as_str().unwrap()));
        let mut collector = Collector::default();
        collector.ingest(&["2","tlc-attempt","100-0","fixture","probe",&identity])?;
        collector.ingest(&["2","tlc-finish","100-0","fixture","completed",&hex(authority_path.as_os_str().as_bytes()),&authority_sha])?;
        collector.ingest(&["2","tlc-fixture-complete",&assertion,&hex(completion_path.as_os_str().as_bytes()),&bytes_sha256(&bytes)])?;
        if pass {
            let observed = String::from_utf8(unhex(authority["test_hex"].as_str().unwrap())?).map_err(io::Error::other)?;
            collector.record_pass(format!("umpire-backend-tlc::umpire_backend_tlc${observed}"), executable_sha)?;
        }
        if unknown {
            collector.ingest(&["1","tlc-attempt","101-0","probe"])?;
            collector.ingest(&["1","tlc-finish","101-0","0"])?;
        }
        collector.finish(&directory)
    }

    #[test]
    fn closed_fixtures_require_positive_proof_and_never_hide_real_unknown() -> io::Result<()> {
        for family in ["tlc-batch-v1", "tlc-cache-v1"] {
            assert_eq!(fixture_counts(family, true, false, |_, _| {})?, (Some(0), Some(0)));
            assert!(fixture_counts(family, false, false, |_, _| {}).is_err());
            assert_eq!(fixture_counts(family, true, true, |_, _| {})?, (None, None));
        }
        Ok(())
    }

    #[test]
    fn rejects_corrupted_crossed_or_unrecognized_fixture_proof() {
        for field in ["renderer_sha256", "scope_sha256", "script_sha256", "test_executable_sha256", "interpreter_sha256", "environment_policy", "cause", "family"] {
            assert!(fixture_counts("tlc-batch-v1", true, false, |doc, _| {
                doc[field] = "bad".into();
            }).is_err(), "accepted {field}");
        }
        assert!(fixture_counts("tlc-batch-v1", true, false, |doc, _| {
            doc["extra"] = true.into();
        }).is_err());
        assert!(fixture_counts("tlc-batch-v1", true, false, |doc, _| {
            doc.as_object_mut().unwrap().remove("exit");
        }).is_err());
        assert!(fixture_counts("tlc-batch-v1", true, false, |_, completion| {
            completion["attempts"] = serde_json::json!([{"attempt":"other","path_hex":"00","sha256":"bad"}]);
        }).is_err());
        assert!(fixture_counts("tlc-cache-v1", true, false, |doc, _| {
            doc["data"][0]["bytes_hex"] = hex(b"changed").into();
        }).is_err());
    }

    #[test]
    fn rejects_other_binary_pass_and_crossed_fixture_parent_family() {
        assert!(Collector::default().record_pass(
            "other-package::other_binary$runner::batch_tests::batch_protocol_requires_complete_selected_evidence".into(),
            "a".repeat(64),
        ).is_err());
        assert!(fixture_counts("tlc-batch-v1", true, false, |doc, completion| {
            let test = hex(b"runner::cache_tests::populated_cache_hits_and_authoritative_disable_runs_fresh");
            doc["test_hex"] = test.clone().into();
            completion["test_hex"] = test.into();
        }).is_err());
    }

    #[test]
    fn refuses_unrecorded_or_partial_fixture_authority_files() {
        for name in ["extra.json", "100-1.json", "partial"] {
            assert!(fixture_counts("tlc-batch-v1", true, false, |doc, _| {
                let script = path(doc["script_path_hex"].as_str().unwrap()).unwrap();
                let receipts = script.parent().unwrap().parent().unwrap();
                fs::write(receipts.join("authorities/scope/tests").join(name), b"{}").unwrap();
            }).is_err(), "accepted {name}");
        }
    }

    #[test]
    fn refuses_authority_documents_when_all_fixture_events_are_missing() -> io::Result<()> {
        let scratch = tempfile::tempdir()?;
        let receipts = scratch.path().canonicalize()?;
        let directory = receipts.join("work/scope/tests");
        fs::create_dir_all(&directory)?;
        fs::create_dir_all(receipts.join("authorities/scope/tests"))?;
        fs::write(receipts.join("authorities/scope/tests/100-0.json"), b"{}")?;
        assert!(Collector::default().finish(&directory).is_err());
        Ok(())
    }

}
