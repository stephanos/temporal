use super::*;
use std::{
    collections::BTreeMap, fs, os::unix::fs::PermissionsExt, path::PathBuf, process::Command,
};

struct Scratch(PathBuf);
impl Scratch {
    fn new() -> Self {
        let root =
            std::env::temp_dir().join(format!("gomad-published-rerun-{}", std::process::id()));
        fs::create_dir(&root).unwrap();
        Self(root)
    }
}
impl Drop for Scratch {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.0).unwrap();
    }
}

#[test]
fn four_original_published_reruns_execute_without_nesting_requested_roots() {
    let scratch = Scratch::new();
    let recorder = scratch.0.join("record arguments");
    fs::write(
        &recorder,
        b"#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$RERUN_ARGUMENTS\"\n",
    )
    .unwrap();
    fs::set_permissions(&recorder, fs::Permissions::from_mode(0o700)).unwrap();
    for (name, variable, shell) in [
        ("guest-profile", "GOMAD_WASM_GUEST_PROFILE_ROOT", true),
        ("guest-variants", "GOMAD_WASM_GUEST_PROFILE_ROOT", true),
        (
            "diagnostic-cluster-baseline",
            "GOMAD_WASM_TEMPORAL_RECOVERY_ROOT",
            true,
        ),
        (
            "matched-release-candidate",
            "GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT",
            false,
        ),
    ] {
        let requested = scratch.0.join(name).join("output root");
        let arguments = scratch.0.join(format!("{name}-arguments"));
        let mut current = requested.clone();
        for _ in 0..3 {
            let call = inputs::Invocation {
                name: name.into(),
                repo: scratch.0.clone(),
                go_command: "go".into(),
                inputs: BTreeMap::from([(variable.into(), current.to_string_lossy().into_owned())]),
                environ: Vec::new(),
                keep: false,
            };
            let proof = inputs::fixture_checkout(scratch.0.clone());
            dispatch_collector(
                &call,
                &proof,
                &mut std::io::sink(),
                |invocation, _, root| {
                    let record = inputs::rerun_record(
                        recorder.to_str().unwrap(),
                        scratch.0.to_str().unwrap(),
                        name,
                        &invocation.inputs,
                        shell,
                    );
                    let report = root.join("report.json");
                    fs::write(&report, canonical::encode(&json!({"rerun":record}))?)
                        .map_err(|error| error.to_string())?;
                    Ok(report)
                },
            )
            .unwrap();
            let store = requested.join(".gomad-measure").join(name);
            let latest: String =
                serde_json::from_slice(&fs::read(store.join("latest.json")).unwrap()).unwrap();
            let report: Value =
                serde_json::from_slice(&fs::read(store.join(latest).join("report.json")).unwrap())
                    .unwrap();
            let mut command = if let Some(script) = report["rerun"]["command"].as_str() {
                let mut command = Command::new("/bin/sh");
                command.args(["-c", script]);
                command
            } else {
                let argv = report["rerun"]["command"].as_array().unwrap();
                let mut command = Command::new(argv[0].as_str().unwrap());
                command.args(argv[1..].iter().map(|value| value.as_str().unwrap()));
                command
            };
            assert!(
                command
                    .env("RERUN_ARGUMENTS", &arguments)
                    .status()
                    .unwrap()
                    .success()
            );
            let recorded = fs::read_to_string(&arguments).unwrap();
            let argv = recorded.lines().collect::<Vec<_>>();
            let flag = format!("--{}", inputs::flag(variable));
            let index = argv.iter().position(|argument| *argument == flag).unwrap();
            current = argv[index + 1].into();
            assert_eq!(current, requested);
            assert_eq!(
                fs::read_dir(&store)
                    .unwrap()
                    .map(Result::unwrap)
                    .filter(|entry| entry.file_name().to_string_lossy().starts_with("run-"))
                    .count(),
                1
            );
        }
    }
}
