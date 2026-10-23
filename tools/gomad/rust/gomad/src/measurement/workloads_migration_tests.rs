use super::*;
use std::{
    os::unix::fs::PermissionsExt,
    sync::atomic::{AtomicU64, Ordering},
};

static NEXT: AtomicU64 = AtomicU64::new(0);

struct Baseline(PathBuf);
impl Baseline {
    fn new() -> Self {
        let root = std::env::temp_dir().join(format!(
            "gomad-migration-cost-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir(&root).unwrap();
        fs::set_permissions(&root, fs::Permissions::from_mode(0o700)).unwrap();
        let baseline = Self(root);
        let mut guests = BTreeMap::new();
        for name in ["choices", "runtime_probes", "time_probes"] {
            let mut runs = BTreeMap::new();
            for seed in [0, 1_000_003] {
                let label = format!("{name}-seed{seed}");
                let root = baseline.0.join(&label);
                fs::create_dir(&root).unwrap();
                fs::set_permissions(&root, fs::Permissions::from_mode(0o700)).unwrap();
                let request = json!({"kind":"standalone","guest":{"module":baseline.0.join("missing-module.wasm"),"model":{"configuration":{"seed":seed}}}});
                let raw = serde_json::to_vec(&request).unwrap();
                baseline.save(&root.join("request.json"), &request);
                baseline.save(&baseline.0.join(format!("{label}.input.json")), &request);
                let command = vec![
                    baseline
                        .0
                        .join("missing-helper")
                        .to_string_lossy()
                        .into_owned(),
                    "--linked-run".into(),
                    baseline
                        .0
                        .join(format!("{label}.input.json"))
                        .to_string_lossy()
                        .into_owned(),
                    root.to_string_lossy().into_owned(),
                ];
                baseline.save(&root.join("report.json"), &json!({
                    "schema":"gomad.linked-profile-run/v1","request_sha256":canonical::digest(&raw),
                    "invocation":command,"fresh_store_id":42,"host_sha256":"a".repeat(64),
                    "execution":{"Ok":{"termination":"exit"}}
                }));
                let capture = |suffix: &str| {
                    json!({
                        "path":baseline.0.join(format!("{label}-driver.{suffix}")),
                        "bytes":0,"captured_bytes":0,"sha256":canonical::digest(b""),
                        "truncated":false,"error":null
                    })
                };
                baseline.save(
                    &baseline.0.join(format!("{label}-driver.json")),
                    &json!({
                        "schema":"gomad.native-process-observation/v1","command":command,
                        "directory":baseline.0,"environment":{},"pid":42,
                        "reaped_monotonic_nanos":99,"exit_code":0,"signal":null,
                        "duration_nanos":1,"termination":"exit", "stdout":capture("stdout"),
                        "stderr":capture("stderr"),"stdout_sha256":canonical::digest(b""),
                        "stderr_sha256":canonical::digest(b""),"cleanup_errors":[],
                        "error":null,"resource_usage":null
                    }),
                );
                runs.insert(seed.to_string(), root);
            }
            guests.insert(name, json!({"provenance":baseline.0.join("missing-provenance.json"),"provenance_sha256":"b".repeat(64),"runs":runs}));
        }
        baseline.save(
            &baseline.0.join("native-migration-baseline.json"),
            &json!({
                "schema":"gomad.native-migration-baseline/v1","guests":guests
            }),
        );
        baseline
    }
    fn save(&self, path: &Path, value: &Value) {
        fs::write(path, serde_json::to_vec(value).unwrap()).unwrap();
        fs::set_permissions(path, fs::Permissions::from_mode(0o600)).unwrap();
    }
    fn change(&self, path: &Path, mutate: impl FnOnce(&mut Value)) {
        let mut value = crate::qualify::spec::strict(&fs::read(path).unwrap()).unwrap();
        mutate(&mut value);
        self.save(path, &value);
    }
    fn invoke(&self) -> Result<PathBuf, String> {
        let call = Invocation {
            name: "migration".into(),
            repo: self.0.clone(),
            go_command: "go".into(),
            inputs: BTreeMap::from([
                (
                    "GOMAD_WASM_MIGRATION_BASELINE".into(),
                    self.0.to_string_lossy().into_owned(),
                ),
                (
                    "GOMAD_WASM_TEST_HELPER".into(),
                    self.0.join("missing-helper").to_string_lossy().into_owned(),
                ),
            ]),
            environ: vec![],
            keep: false,
        };
        migration(
            &call,
            &super::super::inputs::fixture_checkout(self.0.clone()),
            &self.0,
        )
    }
}
impl Drop for Baseline {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.0).unwrap();
    }
}

#[test]
fn migration_costs_precede_module_and_role_access_for_every_selected_baseline() {
    for name in ["choices", "runtime_probes", "time_probes"] {
        for seed in [0, 1_000_003] {
            for (label, cost) in [
                ("missing", None),
                ("zero", Some(json!(0))),
                ("negative", Some(json!(-1))),
                ("overflow", Some(json!(u64::MAX))),
            ] {
                let baseline = Baseline::new();
                baseline.change(
                    &baseline.0.join(format!("{name}-seed{seed}-driver.json")),
                    |command| {
                        if let Some(cost) = cost {
                            command["duration_nanos"] = cost;
                        } else {
                            command.as_object_mut().unwrap().remove("duration_nanos");
                        }
                    },
                );
                let error = baseline.invoke().unwrap_err();
                assert!(
                    error.contains("baseline wall time must be positive"),
                    "{name}/{seed}/{label}: {error}"
                );
                assert!(!baseline.0.join("roles").exists());
                assert!(!baseline.0.join("compile-cache").exists());
                assert!(!baseline.0.join("baseline-costs.json").exists());
            }
        }
    }
    let baseline = Baseline::new();
    let error = baseline.invoke().unwrap_err();
    assert!(!error.contains("baseline wall time"), "{error}");
    assert!(
        baseline.0.join("roles").is_dir(),
        "positive 1ns baseline did not reach role admission: {error}"
    );
    let costs =
        crate::qualify::spec::strict(&fs::read(baseline.0.join("baseline-costs.json")).unwrap())
            .unwrap();
    assert_eq!(costs.as_array().unwrap().len(), 6);
    assert!(
        costs
            .as_array()
            .unwrap()
            .iter()
            .all(|row| row["wall_nanos"] == 1)
    );
    assert!(!baseline.0.join("missing-module.wasm").exists());
}

#[test]
fn migration_cost_preflight_requires_exact_six_selected_roots() {
    for mutation in 0..7 {
        let baseline = Baseline::new();
        baseline.change(
            &baseline.0.join("native-migration-baseline.json"),
            |value| {
                let guests = value["guests"].as_object_mut().unwrap();
                match mutation {
                    0 => {
                        guests.remove("time_probes");
                    }
                    1 => {
                        guests.insert("unselected".into(), guests["choices"].clone());
                    }
                    2 => {
                        let row = guests.remove("time_probes").unwrap();
                        guests.insert("unselected".into(), row);
                    }
                    3 => {
                        guests["time_probes"]["runs"]
                            .as_object_mut()
                            .unwrap()
                            .remove("1000003");
                    }
                    4 => {
                        guests["time_probes"]["runs"]["7"] =
                            guests["time_probes"]["runs"]["0"].clone();
                    }
                    5 => {
                        guests["time_probes"]["runs"]["1000003"] =
                            guests["time_probes"]["runs"]["0"].clone();
                    }
                    _ => {
                        guests["time_probes"]["runs"]["1000003"] = json!("relative");
                    }
                }
            },
        );
        let error = baseline.invoke().unwrap_err();
        assert!(
            error.contains("native migration baseline registry"),
            "mutation {mutation}: {error}"
        );
        assert!(!baseline.0.join("roles").exists());
        assert!(!baseline.0.join("baseline-costs.json").exists());
    }
}

#[test]
fn migration_cost_preflight_authenticates_actual_command_and_report_pair() {
    for mutation in 0..9 {
        let baseline = Baseline::new();
        let root = baseline.0.join("choices-seed0");
        if mutation < 6 {
            baseline.change(
                &baseline.0.join("choices-seed0-driver.json"),
                |command| match mutation {
                    0 => command["pid"] = json!(43),
                    1 => command["command"][1] = json!("--identity"),
                    2 => command["command"][2] = json!(baseline.0.join("other.input.json")),
                    3 => command["command"][3] = json!(baseline.0.join("other-output")),
                    4 => command["termination"] = json!("signal"),
                    _ => command["error"] = json!("wait failed"),
                },
            );
        } else {
            baseline.change(&root.join("report.json"), |report| match mutation {
                6 => report["request_sha256"] = json!("c".repeat(64)),
                7 => report["invocation"][0] = json!("/unrelated/helper"),
                _ => report["execution"] = json!({"Err":"transport failed"}),
            });
        }
        let error = baseline.invoke().unwrap_err();
        assert!(
            error.contains("native migration baseline cost provenance"),
            "mutation {mutation}: {error}"
        );
        assert!(!baseline.0.join("roles").exists());
        assert!(!baseline.0.join("baseline-costs.json").exists());
    }
}
