use super::*;
use crate::campaign::prepare::process::Temporary;
use crate::cycle::{
    custody,
    model::{Mode, PlanConfig},
    plan,
};
use serde_json::json;
use std::fs;

struct Fixture {
    temporary: Temporary,
    repo: PathBuf,
    path: PathBuf,
    value: Value,
}

impl Fixture {
    fn new() -> Self {
        let temporary = Temporary::new(&std::env::temp_dir().canonicalize().unwrap()).unwrap();
        let repo = temporary.0.join("repo");
        fs::create_dir(&repo).unwrap();
        let path = temporary.0.join("config.json");
        let endpoint = temporary.0.join("final-reader");
        fs::write(&endpoint, "actual sealed final reader input").unwrap();
        let stat = ["/usr/bin/stat", "/usr/bin/gnustat"]
            .into_iter()
            .find(|name| fs::symlink_metadata(name).is_ok_and(|m| m.file_type().is_file()))
            .unwrap();
        let contract = plan::derive(
            &PlanConfig {
                binary: endpoint.clone(),
                config_path: path.clone(),
                config_sha256: "a".repeat(64),
            },
            Mode::Parallel,
        )
        .unwrap();
        let stages: serde_json::Map<String, Value> = contract
            .stages
            .into_iter()
            .map(|stage| {
                (
                    stage.contract.label,
                    json!({"adapter": stage.contract.adapter}),
                )
            })
            .collect();
        let scratch = temporary.0.join("scratch");
        let value = json!({
            "schema": "gomad.rust-qualification-driver/v1", "repo": repo,
            "root": temporary.0.join("evidence"), "scratch": scratch,
            "temporary_paths": {"TMPDIR": scratch.join("tmp"), "GOTMPDIR": scratch.join("go-tmp")},
            "stages": stages, "final_reader": [endpoint],
            "final_sections": {"section": {"argv": [endpoint], "required": ["actual-proof"]}},
            "identity": {"host": "a".repeat(64), "helper": "b".repeat(64)},
            "cores": 2, "mem_gib": 4, "sent_epoch": 10, "deadline_s": 3600,
            "files": {stat: custody::digest(&fs::read(stat).unwrap()),
                endpoint.to_str().unwrap(): custody::digest(&fs::read(&endpoint).unwrap())}
        });
        Self {
            temporary,
            repo,
            path,
            value,
        }
    }

    fn load(&self, value: &Value) -> Result<Configured, String> {
        let bytes = serde_json::to_vec(value).unwrap();
        fs::write(&self.path, &bytes).unwrap();
        self.load_bytes(&bytes, &custody::digest(&bytes))
    }

    fn load_bytes(&self, bytes: &[u8], sha: &str) -> Result<Configured, String> {
        fs::write(&self.path, bytes).unwrap();
        Configured::read_with_context(
            &self.path,
            sha,
            &TrustedContext {
                repo_root: &self.repo,
                now_epoch: 20.0,
                available_cores: 4,
            },
        )
    }
}

#[test]
#[expect(
    clippy::too_many_lines,
    reason = "One table keeps all original configuration predicates and controls together"
)]
fn configured_catching_controls() {
    let fixture = Fixture::new();
    fixture.load(&fixture.value).unwrap();
    let cases = vec![
        (
            "driver:490",
            "/schema",
            json!("foreign"),
            "unknown driver config",
        ),
        (
            "driver:491",
            "/repo",
            json!(fixture.temporary.0),
            "wrong candidate checkout",
        ),
        (
            "driver:492",
            "/stages",
            json!({}),
            "missing/unexpected configured stages",
        ),
        (
            "driver:493",
            "/stages/validate/adapter",
            json!("pending"),
            "unbound original stage endpoints",
        ),
        (
            "driver:495-reader",
            "/final_reader",
            Value::Null,
            "missing actual final readers",
        ),
        (
            "driver:495-sections",
            "/final_sections",
            json!({}),
            "missing actual final readers",
        ),
        (
            "driver:496",
            "/synthetic",
            json!(true),
            "synthetic fixture cannot qualify",
        ),
        (
            "driver:497",
            "/identity",
            json!({"host": "a".repeat(64)}),
            "frozen host/helper identities required",
        ),
        (
            "driver:498-cores",
            "/cores",
            json!(0),
            "invalid host resource caps",
        ),
        (
            "driver:498-mem",
            "/mem_gib",
            json!(-1),
            "invalid host resource caps",
        ),
        (
            "driver:498-bool",
            "/cores",
            json!(true),
            "invalid host resource caps",
        ),
        ("driver:500", "/cores", json!(5), "core cap exceeds host"),
        (
            "driver:501-future",
            "/sent_epoch",
            json!(21),
            "invalid send epoch",
        ),
        (
            "driver:501-zero",
            "/sent_epoch",
            json!(0),
            "invalid send epoch",
        ),
        (
            "driver:105",
            "/deadline_s",
            json!(3601),
            "deadline must be positive",
        ),
        (
            "host_admission:19",
            "/root",
            json!("relative"),
            "evidence root must be an absolute canonical path",
        ),
        (
            "host_admission:21",
            "/scratch",
            json!("relative"),
            "scratch must be an absolute canonical path",
        ),
        (
            "host_admission:23",
            "/scratch",
            json!(fixture.temporary.0.join("missing/child")),
            "scratch parent must already exist",
        ),
        (
            "host_admission:24",
            "/scratch",
            json!(fixture.repo.join("scratch")),
            "scratch overlaps candidate source",
        ),
        (
            "host_admission:26-scratch-ancestor",
            "/root",
            json!(fixture.temporary.0.join("scratch/evidence")),
            "scratch overlaps evidence root",
        ),
        (
            "host_admission:26-evidence-ancestor",
            "/root",
            json!(fixture.temporary.0),
            "scratch overlaps evidence root",
        ),
        (
            "host_admission:29",
            "/temporary_paths",
            json!({"TMPDIR": "/tmp"}),
            "explicit TMPDIR and GOTMPDIR",
        ),
        (
            "host_admission:33",
            "/temporary_paths/TMPDIR",
            json!(fixture.temporary.0.join("foreign")),
            "must be a direct canonical scratch child",
        ),
        (
            "host_admission:33-relative",
            "/temporary_paths/TMPDIR",
            json!("relative"),
            "must be a direct canonical scratch child",
        ),
        (
            "host_admission:35",
            "/environment",
            json!({"TMPDIR": "/foreign"}),
            "conflicting temporary environment binding",
        ),
        (
            "host_admission:37",
            "/temporary_paths/GOTMPDIR",
            fixture.value["temporary_paths"]["TMPDIR"].clone(),
            "temporary paths must be distinct",
        ),
        (
            "internal-freezing",
            "/_freezing",
            json!(true),
            "reserved configuration field",
        ),
        (
            "invalid-endpoint",
            "/final_reader",
            json!(["/unsealed-endpoint"]),
            "unsealed actual final endpoint",
        ),
        (
            "invalid-section",
            "/final_sections/section/argv",
            json!(["/unsealed-endpoint"]),
            "unsealed actual final endpoint",
        ),
        (
            "identity-digest",
            "/identity/helper",
            json!("B".repeat(64)),
            "malformed identity",
        ),
    ];
    for (id, pointer, replacement, reason) in cases {
        let mut changed = fixture.value.clone();
        if let Some(target) = changed.pointer_mut(pointer) {
            *target = replacement;
        } else {
            changed[pointer.trim_start_matches('/')] = replacement;
        }
        let error = fixture.load(&changed).unwrap_err();
        assert!(error.contains(reason), "{id}: {error:?}, wanted {reason}");
        fixture
            .load(&fixture.value)
            .unwrap_or_else(|e| panic!("{id} control: {e}"));
        println!("configuration predicate {id}: refusal={error:?}; control=accepted");
    }
    let mut changed = fixture.value.clone();
    for stat in ["/usr/bin/stat", "/usr/bin/gnustat"] {
        changed["files"].as_object_mut().unwrap().remove(stat);
    }
    assert!(
        fixture
            .load(&changed)
            .unwrap_err()
            .contains("unsealed native filesystem probe")
    );
    fixture.load(&fixture.value).unwrap();
}

#[test]
fn configured_authenticates_original_bounded_strict_bytes() {
    let fixture = Fixture::new();
    let bytes = serde_json::to_vec(&fixture.value).unwrap();
    let mut duplicate = bytes.clone();
    duplicate.pop();
    duplicate.extend_from_slice(br#","schema":"gomad.rust-qualification-driver/v1"}"#);
    let error = fixture
        .load_bytes(&duplicate, &custody::digest(&duplicate))
        .unwrap_err();
    assert!(error.contains("duplicate"), "driver:51: {error}");
    fixture.load(&fixture.value).unwrap();
    println!("orchestration:driver:51 duplicate-buffer refusal={error:?}; control=accepted");
    let original = String::from_utf8(bytes.clone()).unwrap();
    for token in ["NaN", "Infinity", "-Infinity", "1e999"] {
        let raw = original.replace("\"cores\":2", &format!("\"cores\":{token}"));
        assert_ne!(raw, original);
        let error = fixture
            .load_bytes(raw.as_bytes(), &custody::digest(raw.as_bytes()))
            .unwrap_err();
        assert!(
            error.contains("qualification JSON"),
            "driver:58 {token}: {error}"
        );
        fixture.load(&fixture.value).unwrap();
        println!("orchestration:driver:58 token={token} refusal={error:?}; control=accepted");
    }
    let error = fixture
        .load_bytes(b"[", &custody::digest(b"["))
        .unwrap_err();
    assert!(error.contains("qualification JSON"), "malformed: {error}");
    fixture.load(&fixture.value).unwrap();
    let error = fixture
        .load_bytes(b"[", &custody::digest(&bytes))
        .unwrap_err();
    assert!(
        error.contains("configuration SHA256 mismatch"),
        "authenticate before decode: {error}"
    );
    let raw = vec![b' '; (16 << 20) + 1];
    assert!(
        fixture
            .load_bytes(&raw, &custody::digest(&raw))
            .unwrap_err()
            .contains("bound")
    );
    assert!(
        fixture
            .load_bytes(&bytes, "bad")
            .unwrap_err()
            .contains("malformed configuration SHA256")
    );
    let configured = fixture.load(&fixture.value).unwrap();
    assert_eq!(configured.bytes(), bytes);
    assert_eq!(configured.path(), fixture.path);
    assert_eq!(configured.sha256(), custody::digest(&bytes));
    assert_eq!(configured.value(), &fixture.value);
    let error = Configured::read(&fixture.path, configured.sha256()).unwrap_err();
    assert!(
        error.contains("wrong candidate checkout"),
        "production checkout anchor: {error}"
    );
    configured.recheck().unwrap();
    let mut changed = bytes.clone();
    changed.push(b' ');
    fs::write(&fixture.path, changed).unwrap();
    assert!(
        configured
            .recheck()
            .unwrap_err()
            .contains("driver config changed")
    );
    fs::write(&fixture.path, &bytes).unwrap();
    configured.recheck().unwrap();
}

#[test]
fn release_identity_binding_catching_controls() {
    let identity = json!({"host": "a".repeat(64), "helper": "b".repeat(64)});
    bind_identity("release", &identity, "release", &identity).unwrap();
    for (expected_role, role) in [("fixture", "release"), ("release", "fixture")] {
        assert!(
            bind_identity(expected_role, &identity, role, &identity)
                .unwrap_err()
                .contains("qualification requires admitted release role")
        );
        bind_identity("release", &identity, "release", &identity).unwrap();
    }
    let foreign = json!({"host": "c".repeat(64), "helper": "b".repeat(64)});
    assert!(
        bind_identity("release", &identity, "release", &foreign)
            .unwrap_err()
            .contains("candidate identity mismatch")
    );
    let empty = json!({});
    assert!(
        bind_identity("release", &empty, "release", &empty)
            .unwrap_err()
            .contains("candidate identity mismatch")
    );
    for malformed in [
        json!({"host": 3}),
        json!({"host": "A".repeat(64)}),
        json!({"host": "abc"}),
    ] {
        assert!(
            bind_identity("release", &malformed, "release", &malformed)
                .unwrap_err()
                .contains("malformed identity")
        );
        bind_identity("release", &identity, "release", &identity).unwrap();
    }
}

#[test]
fn deadline_catching_controls() {
    for invalid in [
        json!(0),
        json!(-1),
        json!(3601),
        json!(true),
        json!("1"),
        Value::Null,
    ] {
        assert!(deadline(&invalid).is_err());
        assert_eq!(
            deadline(&json!(3600)).unwrap().to_bits(),
            3600.0_f64.to_bits()
        );
    }
    assert_eq!(
        deadline(&json!(0.25)).unwrap().to_bits(),
        0.25_f64.to_bits()
    );
}

#[test]
fn canonical_temporary_bindings_reject_symlink_aliases() {
    use std::os::unix::fs::symlink;
    let fixture = Fixture::new();
    let alias = fixture.temporary.0.join("alias");
    symlink(&fixture.repo, &alias).unwrap();
    for (pointer, path, reason) in [
        (
            "/root",
            alias.join("evidence"),
            "evidence root must be an absolute canonical path",
        ),
        (
            "/scratch",
            alias.join("scratch"),
            "scratch must be an absolute canonical path",
        ),
        (
            "/temporary_paths/TMPDIR",
            alias.join("tmp"),
            "must be a direct canonical scratch child",
        ),
    ] {
        let mut value = fixture.value.clone();
        *value.pointer_mut(pointer).unwrap() = json!(path);
        assert!(fixture.load(&value).unwrap_err().contains(reason));
        fixture.load(&fixture.value).unwrap();
    }
    let mut value = fixture.value.clone();
    value["scratch"] = json!(fixture.temporary.0);
    assert!(
        fixture
            .load(&value)
            .unwrap_err()
            .contains("scratch overlaps candidate source")
    );
    fixture.load(&fixture.value).unwrap();
}
