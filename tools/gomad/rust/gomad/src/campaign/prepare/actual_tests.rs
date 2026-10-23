use super::*;
use std::os::unix::fs::DirBuilderExt as _;

#[test]
#[expect(
    clippy::too_many_lines,
    reason = "Retain the actual compiler/cache/custody probe in one bounded transaction."
)]
#[ignore = "requires an explicitly selected pinned Go compiler and an empty retained probe root"]
fn actual_scoped_compiler_and_cached_output_preserve_raw_proofs() {
    let compiler = std::env::var("GOMAD_TEST_PREPARATION_COMPILER").expect("pinned compiler");
    let root = PathBuf::from(std::env::var_os("GOMAD_TEST_PREPARATION_ROOT").expect("probe root"));
    fs::create_dir(&root).unwrap();
    let source = root.join("source");
    fs::create_dir(&source).unwrap();
    fs::create_dir(root.join("tmp")).unwrap();
    fs::write(
        source.join("go.mod"),
        "module preparation.probe\n\ngo 1.27.1\n",
    )
    .unwrap();
    fs::write(
        source.join("main.go"),
        "package main\nfunc main() { println(\"preparation\") }\n",
    )
    .unwrap();
    let installation = root.join("installation.json");
    let arguments = vec![
        "--compiler".into(),
        compiler,
        "--cache".into(),
        root.join("cache").to_string_lossy().into_owned(),
        "--go-version=go1.27.1".into(),
        "--output".into(),
        installation.to_string_lossy().into_owned(),
    ];
    crate::cli::install::run(&arguments, &mut Vec::new())
        .unwrap_or_else(|error| panic!("{}", error.message));
    let mut options =
        crate::cli::explore::parse(&["go-run".into(), ".".into()], &mut false).unwrap();
    options.installation = Some(installation);
    options.working_dir = source.clone();
    let mut controls = CompilerControls {
        parallelism: Some(1),
        environment: BTreeMap::from([
            ("GOMAXPROCS".into(), "1".into()),
            (
                "GOTMPDIR".into(),
                root.join("tmp").to_string_lossy().into_owned(),
            ),
            ("GOWASM".into(), String::new()),
        ]),
    };
    let first = prepare_with_compiler_controls(&options, &controls).unwrap();
    options.target_args = vec!["fresh argument".into()];
    let second = prepare_with_compiler_controls(&options, &controls).unwrap();
    assert_eq!(first.module, second.module);
    assert_ne!(first.toolchain, second.toolchain);
    for (name, raw) in first
        .auxiliary
        .iter()
        .filter(|(name, _)| name.contains("/compile/"))
    {
        assert_eq!(second.auxiliary.get(name), Some(raw));
    }
    assert_ne!(
        first.auxiliary["target/preparation-inputs/admission/metadata.json"],
        second.auxiliary["target/preparation-inputs/admission/metadata.json"]
    );
    let provenance: Value = serde_json::from_slice(&second.provenance).unwrap();
    assert!(
        provenance["inputs"]["build_flags"]
            .as_array()
            .unwrap()
            .contains(&json!("-p=1"))
    );
    for (name, value) in &controls.environment {
        assert_eq!(provenance["inputs"]["environment"][name], json!(value));
    }
    controls
        .environment
        .insert("GOWASM".into(), "satconv,signext".into());
    let features = prepare_with_compiler_controls(&options, &controls).unwrap();
    assert_ne!(features.toolchain, second.toolchain);
    fs::write(
        source.join("main.go"),
        "package main\nfunc main() { println(\"changed\") }\n",
    )
    .unwrap();
    let changed = prepare_with_compiler_controls(&options, &controls).unwrap();
    assert_ne!(features.module, changed.module);
    assert_ne!(features.toolchain, changed.toolchain);
    fs::write(
        source.join("main.go"),
        "package main\nfunc main() { println(\"preparation\") }\n",
    )
    .unwrap();
    for (name, prepared) in [("first", &first), ("second", &second)] {
        let campaign = root.join(name);
        fs::DirBuilder::new().mode(0o700).create(&campaign).unwrap();
        let mut plan = json!({});
        crate::campaign::retain_prepared(&campaign, prepared, &mut plan).unwrap();
        fs::write(
            campaign.join("plan.json"),
            canonical::encode(&plan).unwrap(),
        )
        .unwrap();
        let measurement = root.join(format!("{name}-measurement"));
        fs::DirBuilder::new()
            .mode(0o700)
            .create(&measurement)
            .unwrap();
        crate::measurement::module::retain_preparation_observations(&measurement, prepared)
            .unwrap();
        let record = canonical::digest(&prepared.provenance);
        assert_eq!(
            open_observations(&measurement, prepared, &record).unwrap(),
            prepared.auxiliary
        );
        let descriptor: Value = serde_json::from_slice(
            &fs::read(measurement.join("preparation-observations.json")).unwrap(),
        )
        .unwrap();
        assert_eq!(
            descriptor["artifact_root"],
            json!(measurement.join("preparation-observations"))
        );
        assert_eq!(descriptor["record_sha256"], record);
        for (path, raw) in &prepared.auxiliary {
            assert_eq!(
                fs::read(measurement.join("preparation-observations").join(path)).unwrap(),
                *raw
            );
        }
    }
    println!("retained compiler/cache probe: {}", root.display());
}
