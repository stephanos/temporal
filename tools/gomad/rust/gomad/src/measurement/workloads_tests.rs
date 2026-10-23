use super::*;
fn call(name: &str) -> Invocation {
    Invocation {
        name: name.into(),
        repo: "/repo".into(),
        go_command: "go".into(),
        inputs: BTreeMap::new(),
        environ: vec![],
        keep: false,
    }
}
#[test]
fn producer_matrices_preserve_seeds_counts_and_default_frontend_exclusion() {
    for (name, count) in [
        ("runtime-cost-profile", 2),
        ("host-call-profile", 10),
        ("host-owner-profile", 17),
        ("guest-profile", 13),
        ("guest-variants", 192),
        ("failing-suite-profile", 17),
        ("legacy-workflow-profile", 3),
        ("matched-suite", 55),
    ] {
        let rows = plan(&call(name)).unwrap();
        assert_eq!(rows.len(), count, "{name}");
        let names = rows
            .iter()
            .map(|row| row.name.as_str())
            .collect::<std::collections::BTreeSet<_>>();
        assert_eq!(names.len(), rows.len());
    }
    let matched = plan(&call("matched-suite")).unwrap();
    assert!(
        matched
            .iter()
            .all(|row| row.selection != "^TestFrontendSystemInfo$")
    );
}
#[test]
fn matched_order_alternates_and_preserves_original_helper_tape() {
    let mut call = call("matched-suite");
    call.inputs
        .insert("GOMAD_WASM_MATCHED_SUITE".into(), " activity ".into());
    let rows = plan(&call).unwrap();
    assert_eq!(rows.len(), 11);
    let roles = rows[2..10]
        .iter()
        .map(|row| row.role.as_str())
        .collect::<Vec<_>>();
    assert_eq!(
        roles,
        vec![
            "baseline",
            "candidate",
            "candidate",
            "baseline",
            "baseline",
            "candidate",
            "candidate",
            "baseline"
        ]
    );
    assert_eq!(
        rows.last().unwrap().replay_from.as_deref(),
        Some("activity-seed0-baseline")
    );
}
#[test]
fn guest_variant_rotation_preserves_five_measured_samples_per_seed() {
    let rows = plan(&call("guest-variants")).unwrap();
    for (_, selection) in selections() {
        for seed in SEEDS {
            let rows = rows
                .iter()
                .filter(|row| row.selection == selection && row.seed == seed && !row.prepare)
                .collect::<Vec<_>>();
            assert_eq!(rows.len(), 30);
            for repetition in 0..5 {
                let turn = rows
                    .iter()
                    .filter(|row| row.repetition == repetition)
                    .collect::<Vec<_>>();
                assert_eq!(turn[0].variant, "baseline");
                for (slot, row) in turn.into_iter().enumerate().skip(1) {
                    assert_eq!(
                        row.variant,
                        ["baseline", "gc200", "gc400", "gc-off", "gowasm", "fuel-off"]
                            [1 + (slot - 1 + usize::try_from(repetition).unwrap()) % 5]
                    );
                }
            }
        }
    }
}
#[test]
fn profile_sampling_and_host_owner_warmup_are_explicit() {
    let rows = plan(&call("guest-profile")).unwrap();
    assert_eq!(rows.iter().filter(|row| row.sampling).count(), 6);
    assert!(rows[0].prepare);
    for pair in rows[1..].as_chunks::<2>().0 {
        assert!(!pair[0].sampling);
        assert!(pair[1].sampling);
        assert_eq!(pair[1].compare_to.as_deref(), Some(pair[0].name.as_str()));
    }
    let rows = plan(&call("host-owner-profile")).unwrap();
    assert_eq!(rows.iter().filter(|row| row.repetition == -1).count(), 3);
}
#[test]
fn invalid_repetitions_and_scratch_grammar_refuse_before_any_native_run() {
    for value in ["0", "21", "NaN"] {
        let mut call = call("host-call-profile");
        call.inputs.insert(
            "GOMAD_WASM_HOST_CALL_PROFILE_REPETITIONS".into(),
            value.into(),
        );
        assert!(plan(&call).is_err());
    }
    for value in ["", "diag=relative", "../escape=/binary", "missing-equals"] {
        let mut call = call("scratch-smoke-profile");
        call.inputs
            .insert("GOMAD_SCRATCH_HELPERS".into(), value.into());
        assert!(plan(&call).is_err());
    }
}
#[test]
fn request_transformations_preserve_unrelated_arguments_and_bound_gc_off() {
    let mut row = RunSpec::new("fixture", ACTIVITY, 0, "diagnostic");
    row.prepare = true;
    row.gc = "off".into();
    row.sampling = true;
    let input = json!({"kind":"standalone","guest":{"limits":{"memory_bytes":4_u64<<30},"model":{"arguments":["temporal.test","-test.v",format!("-test.run={ACTIVITY}"),"literal $arg"],"environment":["PWD=/workspace","TZ=UTC"]}}});
    let transformed = apply(&row, input, Path::new("/cache")).unwrap();
    assert_eq!(transformed["compile_cache"], "/cache");
    assert_eq!(
        transformed["guest"]["model"]["arguments"],
        json!(["temporal.test", "-test.v", "-test.run=^$", "literal $arg"])
    );
    assert_eq!(
        transformed["guest"]["model"]["environment"],
        json!([
            "PWD=/workspace",
            "TZ=UTC",
            "GOGC=off",
            "GOMEMLIMIT=3221225472B"
        ])
    );
    assert_eq!(
        transformed["guest"]["guest_profile"],
        json!({"interval_micros":1000,"max_samples":10000})
    );
    assert!(apply(&row, transformed, Path::new("/cache")).is_err());
}

#[test]
fn created_native_cache_is_private_and_existing_unsafe_cache_is_refused() {
    use std::os::unix::fs::PermissionsExt;
    let path =
        std::env::temp_dir().join(format!("gomad-private-native-cache-{}", std::process::id()));
    cache_directory(&path).unwrap();
    assert_eq!(
        fs::metadata(&path).unwrap().permissions().mode() & 0o777,
        0o700
    );
    fs::set_permissions(&path, fs::Permissions::from_mode(0o755)).unwrap();
    assert!(cache_directory(&path).is_err());
    fs::remove_dir(&path).unwrap();
}

#[test]
fn variant_recommendations_fold_all_seeds_without_losing_per_seed_decisions() {
    let root = std::env::temp_dir().join(format!(
        "gomad-variant-decisions-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    fs::create_dir(&root).unwrap();
    let mut samples = Vec::new();
    for row in plan(&call("guest-variants")).unwrap() {
        if row.variant == "fuel-off" && row.seed == 2_000_006 && row.repetition == 4 {
            continue;
        }
        let index = usize::try_from(row.repetition).unwrap();
        let times = match (row.variant.as_str(), row.seed) {
            ("baseline", _) => [98, 99, 100, 101, 102],
            ("gc400" | "gc-off", 0) => [80, 89, 90, 91, 100],
            ("gc-off", 1_000_003) => [108, 109, 110, 111, 112],
            _ => [78, 79, 80, 81, 82],
        };
        let failed = row.variant == "gowasm" && row.seed == 2_000_006 && row.repetition == 2;
        samples.push(json!({"sample":row,"status":if failed {"failed"} else {"executed"},"profile":{"phases":{"execute":{"wall_nanos":times[index]}}}}));
    }
    variant_recommendations(&root, &samples).unwrap();
    let measurements: Vec<Value> =
        serde_json::from_slice(&fs::read(root.join("variant-recommendations.json")).unwrap())
            .unwrap();
    let summary: Vec<Value> = serde_json::from_slice(
        &fs::read(root.join("variant-recommendations-summary.json")).unwrap(),
    )
    .unwrap();
    assert_eq!(measurements.len(), 36);
    let mut expected_summary = Vec::new();
    for selection in ["activity", "workflow"] {
        for (variant, per_seed, decision) in [
            ("baseline", ["control", "control", "control"], "control"),
            ("fuel-off", ["adopt", "adopt", "reject"], "reject"),
            ("gc-off", ["investigate", "reject", "adopt"], "reject"),
            ("gc200", ["adopt", "adopt", "adopt"], "adopt"),
            ("gc400", ["investigate", "adopt", "adopt"], "investigate"),
            ("gowasm", ["adopt", "adopt", "reject"], "reject"),
        ] {
            let actual: Vec<_> = measurements
                .iter()
                .filter(|row| row["selection"] == selection && row["variant"] == variant)
                .map(|row| {
                    (
                        row["seed"].as_u64().unwrap(),
                        row["recommendation"].as_str().unwrap(),
                    )
                })
                .collect();
            assert_eq!(
                actual,
                vec![
                    (0, per_seed[0]),
                    (1_000_003, per_seed[1]),
                    (2_000_006, per_seed[2])
                ],
                "{selection}/{variant}"
            );
            expected_summary.push(json!({"selection":selection,"variant":variant,"recommendation":decision,"adoption_requires_separate_qualification":true}));
        }
    }
    assert_eq!(summary, expected_summary);
    fs::remove_dir_all(root).unwrap();
}
/// Original `TestMatchedSuiteSelectionCacheAndOrder` cache half: the matched
/// baseline cache must be an explicit absolute directory distinct from the
/// candidate cache, with a trailing separator not hiding equality.
#[test]
fn matched_baseline_cache_must_be_absolute_and_distinct_from_candidate() {
    let candidate = Path::new("/scratch/candidate");
    for baseline in ["", "relative", "/scratch/candidate", "/scratch/candidate/"] {
        assert!(
            matched_baseline_cache(baseline, candidate).is_err(),
            "baseline cache {baseline:?} admitted"
        );
    }
    matched_baseline_cache("/scratch/baseline", candidate).unwrap();
}
