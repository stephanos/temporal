use super::{PreparationCase, PreparationInputs, canonical};
use crate::{cli::explore::ExploreOptions, install::config};
use serde::Deserialize;
use serde_json::json;
use std::{
    io::Write as _,
    os::unix::fs::{DirBuilderExt as _, OpenOptionsExt as _},
    path::{Path, PathBuf},
};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Recipe {
    schema: String,
    root: PathBuf,
    compiler: PathBuf,
    compiler_sha256: String,
    cli: PathBuf,
    cli_sha256: String,
}

fn bytes(path: &Path, raw: &[u8]) {
    let mut file = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .unwrap();
    file.write_all(raw).unwrap();
    file.sync_all().unwrap();
}

fn configuration() -> (serde_json::Value, serde_json::Value) {
    let mut original = config::build("gomad.wasi-cooperative/v1", None, None)
        .unwrap()
        .original;
    original["Clock"]["ReadStepNanos"] = json!(0);
    original["Limits"]["OutputBytes"] = json!(1 << 20);
    original["Limits"]["FilesystemBytes"] = json!(8 << 20);
    original["Limits"]["Calls"] = json!(1 << 20);
    original["Limits"]["TranscriptBytes"] = json!(8 << 20);
    let mut effective = original.clone();
    // The original campaign's IOTranscriptLimit overrides the provider's base policy.
    effective["Limits"]["TranscriptBytes"] = json!(64 << 20);
    config::convert(original.clone(), "gomad.wasi-cooperative/v1").unwrap();
    config::convert(effective.clone(), "gomad.wasi-cooperative/v1").unwrap();
    assert_eq!(
        canonical::digest(&canonical::encode(&original).unwrap()),
        "f56de9ae094fbb7934b7b44f5e6a165ef265c860aaced107c40f510209bf7535"
    );
    assert_eq!(
        canonical::digest(&canonical::encode(&effective).unwrap()),
        "30c739aefce01c762f5feef379c270a2d91bb666b3af94234f43070fa9e9fe1a"
    );
    (original, effective)
}

fn options(root: &Path, source: &Path, name: &str) -> ExploreOptions {
    let mut arguments = vec![
        "--json".into(),
        "--seeds=7".into(),
        "--parallel=1".into(),
        "--execution-timeout=1m".into(),
        "--overall-timeout=5m".into(),
        "--terminate-grace=100ms".into(),
        "--on-failure=first".into(),
        "--failure-budget=1".into(),
        "--coverage=none".into(),
        "--output-limit=1MiB".into(),
        "--world-transition-limit=4MiB".into(),
        "--build-tag=gomad".into(),
        "--build-tag=test_dep".into(),
        "--working-dir".into(),
        source.to_string_lossy().into_owned(),
        "--artifacts".into(),
        root.join(format!("{name}-preparation"))
            .to_string_lossy()
            .into_owned(),
    ];
    if name == "handoff" {
        arguments.extend([
            "--scenario".into(),
            root.join("scenario.json").to_string_lossy().into_owned(),
        ]);
    } else {
        arguments.extend(["--diagnostics".into(), "--choice-bytes=4MiB".into()]);
    }
    arguments.extend(["go-run".into(), ".".into()]);
    let mut options = crate::cli::explore::parse(&arguments, &mut false).unwrap();
    options.installation = Some(root.join("installation.json"));
    super::validate_campaign(&options, name == "handoff").unwrap();
    super::write(
        &root.join(format!("{name}-original-options.json")),
        &json!({"argv":arguments,"options":options}),
    );
    options.choice_bytes = 1 << 20;
    options.diagnostics = true;
    options
}

#[test]
fn original_recipe_uses_parser_and_preserves_raw_options_beside_effective_overrides() {
    let root = crate::campaign::prepare::process::Temporary::new(&std::env::temp_dir()).unwrap();
    bytes(
        &root.0.join("scenario.json"),
        include_bytes!("../../../../testdata/parity/executed-divergence-original-scenario.json"),
    );
    let (original, effective) = configuration();
    assert_eq!(original["Limits"]["TranscriptBytes"], 8 << 20);
    assert_eq!(effective["Limits"]["TranscriptBytes"], 64 << 20);
    for name in ["draw", "clock", "handoff", "output-clock"] {
        let options = options(&root.0, Path::new("/original-source"), name);
        assert_eq!(options.choice_bytes, 1 << 20);
        assert!(options.diagnostics);
        let raw: serde_json::Value = serde_json::from_slice(
            &std::fs::read(root.0.join(format!("{name}-original-options.json"))).unwrap(),
        )
        .unwrap();
        let original: ExploreOptions = serde_json::from_value(raw["options"].clone()).unwrap();
        super::validate_campaign(&original, name == "handoff").unwrap();
        assert_eq!(
            original.choice_bytes,
            if name == "handoff" { 0 } else { 4 << 20 }
        );
        assert_eq!(original.diagnostics, name != "handoff");
    }
}

pub(super) fn materialize(path: &Path) -> PreparationInputs {
    let recipe: Recipe = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    assert_eq!(recipe.schema, "gomad.original-counterfactual-recipe/v1");
    assert!(recipe.root.is_absolute() && recipe.compiler.is_absolute());
    assert_eq!(
        canonical::digest(&std::fs::read(&recipe.compiler).unwrap()),
        recipe.compiler_sha256
    );
    super::authenticate_cli(&recipe.cli, &recipe.cli_sha256).unwrap();
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(&recipe.root)
        .unwrap();
    let source = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../../../common/gomaddebug/testdata/determinism")
        .canonicalize()
        .unwrap();
    assert_eq!(
        canonical::digest(&std::fs::read(source.join("main.go")).unwrap()),
        super::MAIN_SHA256
    );
    let runtime = recipe.root.join("runtime");
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(&runtime)
        .unwrap();
    let (semantics, wire) = super::runtime_sources(
        include_bytes!("../../../../internal/model/runtimesource/gomad_semantics.go.txt"),
        include_bytes!(
            "../../../../internal/model/runtimesource/gomad_choicewire_generated.go.txt"
        ),
    )
    .unwrap();
    bytes(&runtime.join("gomad_semantics.go.txt"), &semantics);
    bytes(&runtime.join("gomad_choicewire_generated.go.txt"), &wire);
    let scenario =
        include_bytes!("../../../../testdata/parity/executed-divergence-original-scenario.json");
    assert_eq!(
        canonical::digest(scenario),
        "db02fea5e0fe20d46bb9786c5ee37f59802c4c7f6af9f6744cff48d8819c939e"
    );
    bytes(&recipe.root.join("scenario.json"), scenario);
    let (original, effective) = configuration();
    super::write(
        &recipe.root.join("original-provider-config.json"),
        &original,
    );
    super::write(&recipe.root.join("effective-config.json"), &effective);
    let cases = ["draw", "clock", "handoff", "output-clock"]
        .into_iter()
        .map(|name| PreparationCase {
            name: name.into(),
            options: options(&recipe.root, &source, name),
            plan: recipe.root.join(format!("{name}-plan.json")),
        })
        .collect();
    let values = [
        ("compiler", recipe.compiler),
        ("runtime-root", runtime),
        ("cache", recipe.root.join("cache")),
        ("output", recipe.root.join("installation.json")),
        ("config", recipe.root.join("effective-config.json")),
    ];
    let mut installation_arguments = values
        .into_iter()
        .flat_map(|(name, path)| [format!("--{name}"), path.to_string_lossy().into_owned()])
        .collect::<Vec<_>>();
    installation_arguments.extend(
        [
            "--profile",
            "gomad.wasi-cooperative/v1",
            "--go-version",
            "go1.27.1",
            "--memory-bytes",
            "268435456",
            "--fuel",
            "1152921504606846976",
        ]
        .map(str::to_owned),
    );
    PreparationInputs {
        installation_arguments,
        cli: recipe.cli,
        cli_sha256: recipe.cli_sha256,
        cases,
        output: recipe.root.join("producer-input.json"),
    }
}
