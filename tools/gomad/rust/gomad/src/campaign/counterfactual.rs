#[path = "counterfactual_assertions.rs"]
mod assertions;
#[path = "counterfactual_recipe.rs"]
mod recipe;

use super::{execution, ledger, prepare::Prepared};
use crate::evidence::canonical;
use gomad_wasmhost::{linked, linked_cluster::Handle};
use serde::Deserialize;
use serde_json::{Value, json};
use std::{
    collections::BTreeSet,
    io::Write as _,
    os::unix::fs::{DirBuilderExt as _, OpenOptionsExt as _, PermissionsExt as _},
    path::{Path, PathBuf},
    time::{Duration, Instant},
};

pub(super) struct Fault {
    pub kind: linked::FixtureFault,
    pub owner: Option<Handle>,
}

const SEMANTICS_SHA256: &str = "5ed9d769d7d97109f79896368e40a0469d219bd6819fde247914bad140ab4266";
const WIRE_SHA256: &str = "bc11cf9093eac65c5a54e29494c35bb2ec23c8a93a1a88a33ec2a2e5e41a68ee";
const MAIN_SHA256: &str = "b1f66edccd8a9f0b2dcb714e12edf043fa71ded378b932068937b92430771998";
const FIXTURE_SHA256: &str = "1b3ad687b44742555d9b9b66e73ac1c30cc621c6d5cd2b95f4afbbe41a95ea18";

const INSERTION: &str = r"
	gomadDeterminismFixturePending = gomadDeterminismFixturePending || gomadSeed&(1<<63) != 0
	gomadDeterminismFixtureOutputPending = gomadDeterminismFixtureOutputPending || gomadSeed&(1<<62) != 0
	gomadSeed &= ^uint64((1<<63)|(1<<62))
";
const APPENDIX: &str = r"
var gomadDeterminismFixturePending bool
var gomadDeterminismFixtureOutputPending bool

//go:linkname gomadDeterminismFixtureOutput
func gomadDeterminismFixtureOutput() bool { return gomadDeterminismFixtureOutputPending }

//go:linkname gomadDeterminismFixtureDraw
func gomadDeterminismFixtureDraw() {
	if gomadDeterminismFixturePending {
		gomadDeterminismFixturePending = false
		gomadChoiceRunqSeeded(2)
	}
}
";

fn runtime_sources(semantics: &[u8], wire: &[u8]) -> Result<(Vec<u8>, Vec<u8>), String> {
    if canonical::digest(semantics) != SEMANTICS_SHA256 || canonical::digest(wire) != WIRE_SHA256 {
        return Err("original fixture runtime sources changed".into());
    }
    let semantics = std::str::from_utf8(semantics).map_err(|error| error.to_string())?;
    let anchor = "func gomadChoiceSeedRandom() {";
    if semantics.matches(anchor).count() != 1 {
        return Err("original fixture anchor is not unique".into());
    }
    let mut patched = semantics.replacen(anchor, &format!("{anchor}{INSERTION}"), 1);
    patched.push_str(APPENDIX);
    if canonical::digest(patched.as_bytes()) != FIXTURE_SHA256 {
        return Err("original fixture runtime recipe changed".into());
    }
    Ok((patched.into_bytes(), wire.to_vec()))
}

fn validate_preparation(prepared: &Prepared) -> Result<(), String> {
    let configuration = &prepared.configuration;
    let model = configuration
        .model
        .as_ref()
        .ok_or("missing fixture model")?;
    let namespace = configuration
        .namespace
        .as_ref()
        .ok_or("missing fixture namespace")?;
    if configuration.engine != linked::Conditions::DeterminismFixtureReplay.configuration()
        || configuration.profile != "gomad.wasi-cooperative/v1"
        || configuration.limits.memory_bytes != 256 << 20
        || configuration.limits.fuel != 1 << 60
        || configuration.limits.wall_millis != 60_000
        || configuration.limits.output_bytes != 1 << 20
        || configuration.world_transition_bytes != Some(4 << 20)
        || model["limit"] != 1 << 20
        || model["diagnostics"] != true
        || model["limits"]["calls"] != 1 << 20
        || model["limits"]["transcript_bytes"] != 64 << 20
        || model["limits"]["pending_events"] != 4096
        || model["epoch_nanos"] != 946_684_800_000_000_000_u64
        || model["read_step_nanos"] != 0
        || namespace.limits.output_bytes != 1 << 20
        || namespace.limits.filesystem_bytes != 8 << 20
        || namespace.limits.files != 1000
        || namespace.limits.descriptors != 64
    {
        return Err(
            "counterfactual effective request differs from the original fixture bounds".into(),
        );
    }
    let provenance: Value =
        serde_json::from_slice(&prepared.provenance).map_err(|error| error.to_string())?;
    for (field, expected) in [
        ("compiler", FIXTURE_SHA256),
        ("compiler", WIRE_SHA256),
        ("sources", MAIN_SHA256),
    ] {
        if !provenance["inputs"][field]
            .as_object()
            .is_some_and(|entries| entries.values().any(|value| value == expected))
        {
            return Err(format!(
                "missing authenticated original fixture {field} member {expected}"
            ));
        }
    }
    if provenance["inputs"]["tags"] != json!(["gomad", "test_dep"])
        || provenance["inputs"]["kind"] != "go-run"
        || provenance["inputs"]["source"] != "."
        || provenance["inputs"]["go_version"] != "go1.27.1"
    {
        return Err("original fixture compiler selection changed".into());
    }
    Ok(())
}

pub(super) fn validate_bound(bound: &super::scenario::Bound) -> Result<(), String> {
    let request = &bound.request;
    let expected = [
        ("a", gomad_wasmhost::linked_cluster::Action::Start),
        ("b", gomad_wasmhost::linked_cluster::Action::Start),
        ("a", gomad_wasmhost::linked_cluster::Action::Wait),
        ("b", gomad_wasmhost::linked_cluster::Action::Wait),
    ];
    if request.incarnations.len() != 2
        || request.steps.len() != expected.len()
        || !request.boundaries.is_empty()
        || !request.crash_groups.is_empty()
        || request
            .steps
            .iter()
            .zip(expected)
            .any(|(step, (node, action))| {
                step.handle.node != node || step.handle.incarnation != 1 || step.action != action
            })
        || request.incarnations.iter().any(|incarnation| {
            incarnation.handle.incarnation != 1
                || !matches!(incarnation.handle.node.as_str(), "a" | "b")
                || incarnation.request.model.as_ref().is_none_or(|model| {
                    model.configuration.limit != 1 << 20 || !model.configuration.diagnostics
                })
        })
    {
        return Err("counterfactual handoff differs from the original two-owner lifecycle or effective diagnostic bounds".into());
    }
    Ok(())
}

#[test]
fn original_fixture_recipe_is_byte_identical_and_refuses_other_runtime_sources() {
    let semantics =
        include_bytes!("../../../../internal/model/runtimesource/gomad_semantics.go.txt");
    let wire = include_bytes!(
        "../../../../internal/model/runtimesource/gomad_choicewire_generated.go.txt"
    );
    let (patched, copied) = runtime_sources(semantics, wire).unwrap();
    assert_eq!(canonical::digest(&patched), FIXTURE_SHA256);
    assert_eq!(copied, wire);
    let mut changed = semantics.to_vec();
    changed.push(b'\n');
    assert!(runtime_sources(&changed, wire).is_err());
    assert!(runtime_sources(semantics, b"other wire").is_err());
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Inputs {
    schema: String,
    cli: PathBuf,
    cli_sha256: String,
    cases: Vec<Case>,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Case {
    name: String,
    original_requested_choice_bytes: u64,
    campaign: crate::cli::explore::ExploreOptions,
}

fn validate_campaign(
    options: &crate::cli::explore::ExploreOptions,
    clustered: bool,
) -> Result<(), String> {
    if options.seeds != "7"
        || options.parallel != 1
        || options.execution_timeout != Duration::from_secs(60)
        || options.overall_timeout != Duration::from_secs(300)
        || options.terminate_grace != Duration::from_millis(100)
        || options.on_failure != "first"
        || options.failure_budget != 1
        || options.coverage != "none"
        || options.keep_successes != "none"
        || options.success_limit != 0
        || options.success_bytes != 0
        || options.output_limit != 1 << 20
        || options.world_transition_limit != 4 << 20
        || options.choice_bytes != if clustered { 0 } else { 4 << 20 }
        || options.choices == clustered
        || options.diagnostics == clustered
        || options.scenario.is_some() != clustered
        || options.soft_assertion_report_only
        || !options.environment.is_empty()
        || options.build_tags != ["gomad", "test_dep"]
        || options.target_kind != "go-run"
        || options.source != "."
        || !options.target_args.is_empty()
    {
        return Err("original Explore campaign options changed".into());
    }
    if let Some(scenario) = &options.scenario {
        let original = include_bytes!(
            "../../../../testdata/parity/executed-divergence-original-scenario.json"
        );
        if super::scenario::normalize(scenario)? != super::scenario::normalize(original)? {
            return Err("original handoff scenario changed".into());
        }
    }
    Ok(())
}

#[test]
fn original_campaign_options_refuse_deadline_policy_and_recording_relaxations() {
    let arguments = [
        "--seeds=7",
        "--parallel=1",
        "--execution-timeout=1m",
        "--overall-timeout=5m",
        "--terminate-grace=100ms",
        "--output-limit=1MiB",
        "--world-transition-limit=4MiB",
        "--diagnostics",
        "--choice-bytes=4MiB",
        "--build-tag=gomad",
        "--build-tag=test_dep",
        "go-run",
        ".",
    ]
    .map(str::to_owned);
    let options = crate::cli::explore::parse(&arguments, &mut false).unwrap();
    validate_campaign(&options, false).unwrap();
    let value = serde_json::to_value(&options).unwrap();
    for (field, changed) in [
        ("seeds", json!("8")),
        ("parallel", json!(2)),
        ("execution_timeout", json!({"secs":61,"nanos":0})),
        ("overall_timeout", json!({"secs":301,"nanos":0})),
        ("terminate_grace", json!({"secs":0,"nanos":101_000_000})),
        ("on_failure", json!("all")),
        ("failure_budget", json!(2)),
        ("keep_successes", json!("all")),
        ("choice_bytes", json!(1 << 20)),
        ("diagnostics", json!(false)),
        ("world_transition_limit", json!(8 << 20)),
        ("build_tags", json!(["gomad"])),
    ] {
        let mut changed_options = value.clone();
        changed_options[field] = changed;
        let changed_options = serde_json::from_value(changed_options).unwrap();
        assert!(
            validate_campaign(&changed_options, false).is_err(),
            "{field}"
        );
    }
    let mut clustered = options;
    clustered.scenario = Some(
        include_bytes!("../../../../testdata/parity/executed-divergence-original-scenario.json")
            .to_vec(),
    );
    clustered.choices = false;
    clustered.diagnostics = false;
    clustered.choice_bytes = 0;
    validate_campaign(&clustered, true).unwrap();
    assert!(validate_campaign(&clustered, false).is_err());
    let mut changed: Value = serde_json::from_slice(clustered.scenario.as_ref().unwrap()).unwrap();
    changed["boots"][0]["args"] = json!(["changed"]);
    clustered.scenario = Some(canonical::encode(&changed).unwrap());
    assert!(validate_campaign(&clustered, true).is_err());
}

fn authenticate_cli(path: &Path, expected: &str) -> Result<(), String> {
    let metadata = std::fs::symlink_metadata(path).map_err(|error| error.to_string())?;
    if !path.is_absolute()
        || !metadata.file_type().is_file()
        || metadata.permissions().mode() & 0o111 == 0
    {
        return Err(
            "counterfactual replay requires an absolute executable regular CLI file".into(),
        );
    }
    let bytes = std::fs::read(path).map_err(|error| error.to_string())?;
    if canonical::digest(&bytes) != expected {
        return Err("counterfactual CLI digest changed".into());
    }
    let producer = std::env::current_exe().map_err(|error| error.to_string())?;
    let producer = std::fs::read(producer).map_err(|error| error.to_string())?;
    if canonical::digest(&producer) == expected {
        return Err("libtest producer cannot stand in for the public replay CLI".into());
    }
    Ok(())
}

#[test]
fn replay_cli_binding_refuses_the_libtest_producer_and_wrong_digest() {
    let producer = std::env::current_exe().unwrap();
    let digest = canonical::digest(&std::fs::read(&producer).unwrap());
    assert!(
        authenticate_cli(&producer, &digest)
            .unwrap_err()
            .contains("libtest")
    );
    assert!(
        authenticate_cli(&producer, &"0".repeat(64))
            .unwrap_err()
            .contains("digest")
    );
    assert!(authenticate_cli(Path::new("relative-cli"), &digest).is_err());
}

fn write(path: &Path, value: &Value) {
    let mut file = std::fs::OpenOptions::new()
        .create_new(true)
        .write(true)
        .mode(0o600)
        .open(path)
        .unwrap();
    file.write_all(&canonical::encode(value).unwrap()).unwrap();
    file.sync_all().unwrap();
}

fn assert_preparation_matches(expected: &Prepared, actual: &Prepared) {
    assert_eq!(actual.module, expected.module);
    assert_eq!(
        canonical::encode(&actual.configuration).unwrap(),
        canonical::encode(&expected.configuration).unwrap()
    );
    assert_eq!(actual.scenario, expected.scenario);
    assert_eq!(actual.toolchain, expected.toolchain);
    assert_eq!(actual.provenance, expected.provenance);
    assert_eq!(
        actual.soft_assertion_report_only,
        expected.soft_assertion_report_only
    );
    assert_eq!(
        canonical::encode(&actual.choice_identity).unwrap(),
        canonical::encode(&expected.choice_identity).unwrap()
    );
    assert_eq!(
        canonical::encode(&actual.network).unwrap(),
        canonical::encode(&expected.network).unwrap()
    );
}

#[test]
fn repreparation_binds_scenario_without_reusing_admission_observations() {
    let fixture = crate::installed_lifecycle_tests::retained(false);
    let opened = crate::evidence::store::Opened::open(&fixture.published.path).unwrap();
    let raw = opened
        .payload(crate::replay::preflight::FILE, 1 << 20)
        .unwrap();
    let prepared = || {
        let inputs: crate::replay::preflight::Inputs = serde_json::from_slice(&raw).unwrap();
        Prepared {
            module: vec![1, 2, 3],
            configuration: inputs.configuration,
            toolchain: fixture.record.identity.toolchain.clone(),
            choice_identity: None,
            scenario: Some(br#"{"boots":[{"args":["original"]}]}"#.to_vec()),
            provenance: b"{}".to_vec(),
            network: None,
            soft_assertion_report_only: false,
            compile_cache_root: None,
            auxiliary: Default::default(),
        }
    };
    let expected = prepared();
    let mut actual = prepared();
    actual.auxiliary.insert(
        "target/preparation-inputs/admission/current".into(),
        b"fresh command PID and times".to_vec(),
    );
    assert_preparation_matches(&expected, &actual);
    actual.scenario = Some(br#"{"boots":[{"args":["changed"]}]}"#.to_vec());
    assert!(
        std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
            assert_preparation_matches(&expected, &actual);
        }))
        .is_err()
    );
}

fn campaign_outcome(campaign: &Path, artifact: &Path) -> (crate::evidence::store::Opened, u64) {
    let journal = ledger::Journal::open(campaign, false).unwrap();
    assert_eq!(journal.plan()["overall_nanos"], 300_000_000_000_u64);
    let outcomes = journal
        .events()
        .iter()
        .filter_map(|event| match event {
            ledger::Event::Outcome { executions, .. } => Some(executions),
            _ => None,
        })
        .flatten()
        .collect::<Vec<_>>();
    assert_eq!(outcomes.len(), 1);
    assert_eq!(outcomes[0].seed, 7);
    assert_eq!(outcomes[0].record.outcome.class, "target_failure");
    let opened = crate::evidence::store::Opened::open(artifact).unwrap();
    opened.authenticate_all().unwrap();
    assert_eq!(opened.manifest().record, outcomes[0].record);
    assert!(opened.manifest().files.contains_key("evidence/event-order"));
    (opened, outcomes[0].stored_bytes)
}

fn explore_result(output: &[u8]) -> Value {
    let report: Value = serde_json::from_slice(
        output
            .split(|byte| *byte == b'\n')
            .filter(|line| !line.is_empty())
            .last()
            .unwrap(),
    )
    .unwrap();
    assert_eq!(report["type"], "result");
    assert_eq!(report["failures"], 1);
    assert_eq!(report["attempted"], 1);
    let artifacts = report["artifacts"].as_array().unwrap();
    assert_eq!(artifacts.len(), 1);
    report
}

fn run_campaign(
    expected: Option<&Prepared>,
    case: &Case,
    root: &Path,
    fault: linked::FixtureFault,
    baseline: bool,
) -> Value {
    let cell = root.join(&case.name).join(if baseline {
        "baseline-campaign"
    } else {
        "fault-campaign"
    });
    let mut options = case.campaign.clone();
    validate_campaign(&options, case.name == "handoff").unwrap();
    options.artifacts = cell;
    let started = Instant::now();
    recipe::validate_installation(&options);
    // The original StressExecution override requested diagnostics even for
    // the cluster and capped its runtime trace independently of campaign retention.
    let mut effective = options.clone();
    effective.choice_bytes = 1 << 20;
    effective.diagnostics = true;
    effective.artifacts = options.artifacts.join("preparation");
    let preparation = super::preparation::seed(&effective).unwrap();
    validate_preparation(&preparation.prepared).unwrap();
    assert_eq!(
        preparation.prepared.scenario.is_some(),
        case.name == "handoff"
    );
    assert_eq!(
        preparation
            .prepared
            .configuration
            .environment
            .contains_key("GOMAD_CHOICE_PROFILE"),
        case.name != "handoff"
    );
    if let Some(expected) = expected {
        assert_preparation_matches(expected, &preparation.prepared);
    }
    let (preparation_receipt, _, preparation_sha256) = preparation
        .publish(
            &options.artifacts.join("plan.json"),
            vec![
                std::env::current_exe()
                    .unwrap()
                    .to_string_lossy()
                    .into_owned(),
                "--ignored".into(),
                "--exact".into(),
                "campaign::counterfactual::produce_the_four_original_counterfactual_artifact_pairs"
                    .into(),
            ],
        )
        .unwrap();
    let mut journal = super::seed_journal(&options, &preparation.prepared).unwrap();
    let remaining = options
        .overall_timeout
        .checked_sub(started.elapsed())
        .unwrap();
    let mut output = Vec::new();
    let exit = super::run_observed_using(
        &mut journal,
        &preparation.prepared,
        remaining,
        true,
        &mut output,
        None,
        |prepared, job, timeout, cancellation| {
            execution::run_counterfactual(
                prepared,
                job,
                timeout,
                cancellation,
                Fault {
                    kind: fault,
                    owner: None,
                },
            )
        },
    )
    .unwrap();
    assert_eq!(exit, 1);
    let wall = started.elapsed();
    assert!(wall <= options.overall_timeout);
    let report = explore_result(&output);
    let artifacts = report["artifacts"].as_array().unwrap();
    let artifact = PathBuf::from(artifacts[0]["path"].as_str().unwrap());
    let campaign = journal.path().to_path_buf();
    drop(journal);
    let (opened, stored_bytes) = campaign_outcome(&campaign, &artifact);
    write(&campaign.join("original-explore-result.json"), &report);
    json!({"artifact":artifact,"record_sha256":opened.manifest().record_sha256,
        "bytes":stored_bytes,"campaign":campaign,"campaign_result":report,
        "preparation_receipt":preparation_receipt,"preparation_sha256":preparation_sha256,
        "module_sha256":canonical::digest(&preparation.prepared.module),
        "wall_nanos":wall.as_nanos()})
}

fn produce_pair(case: &Case, root: &Path, kind: linked::FixtureFault) -> (Vec<Value>, Vec<Value>) {
    let mut pair = Vec::new();
    let mut failures = Vec::new();
    let mut prepared = None;
    for fault in [linked::FixtureFault::None, kind] {
        let row = run_campaign(prepared.as_ref(), case, root, fault, pair.is_empty());
        if prepared.is_none() {
            let (_, baseline) = super::preparation::open_receipt(
                Path::new(row["preparation_receipt"].as_str().unwrap()),
                row["preparation_sha256"].as_str().unwrap(),
            )
            .unwrap();
            prepared = Some(baseline);
        }
        let artifact = PathBuf::from(row["artifact"].as_str().unwrap());
        let opened = crate::evidence::store::Opened::open(&artifact).unwrap();
        opened.authenticate_all().unwrap();
        if case.name == "handoff" {
            let summary: Value = serde_json::from_slice(
                &opened
                    .payload("evidence/cluster_summary", 16 << 20)
                    .unwrap(),
            )
            .unwrap();
            let incarnations = summary["incarnations"].as_array().unwrap();
            assert_eq!(incarnations.len(), 2);
            for incarnation in incarnations {
                let handle = &incarnation["handle"];
                let diagnostic = opened
                    .payload(
                        &format!(
                            "evidence/cluster/{}/{}/diagnostic",
                            handle["node"].as_str().unwrap(),
                            handle["incarnation"].as_u64().unwrap()
                        ),
                        1 << 20,
                    )
                    .unwrap();
                assert!(
                    !diagnostic.is_empty(),
                    "original fixture requires a nonempty diagnostic trace for {handle}"
                );
                if incarnation["termination"] != "exit" {
                    failures.push(json!({
                        "original_root":"TestDeterminismExecutedDivergenceArtifacts/handoff",
                        "predicate":"every reported incarnation termination equals exit",
                        "artifact":artifact,
                        "handle":incarnation["handle"],
                        "expected":"exit",
                        "actual":incarnation["termination"],
                        "observation":incarnation
                    }));
                }
            }
        } else {
            assert_eq!(opened.manifest().record.outcome.termination, "exit");
            assert_eq!(opened.manifest().record.outcome.exit_code, Some(1));
            assert!(
                !opened
                    .payload("evidence/diagnostic", 1 << 20)
                    .unwrap()
                    .is_empty(),
                "original fixture requires a nonempty diagnostic trace"
            );
        }
        pair.push(row);
    }
    assert_ne!(pair[0]["record_sha256"], pair[1]["record_sha256"]);
    write(
        &root
            .join(&case.name)
            .join("original-outcome-predicates.json"),
        &json!({"status":if failures.is_empty() {"PASS"} else {"FAIL"},"failures":failures}),
    );
    (pair, failures)
}

fn verify_pair(input: &Inputs, case: &Case, root: &Path, pair: &[Value]) -> Value {
    let cell = |artifact: &Value, axis: &str| {
        json!({
            "fresh":true,"workload":"original-determinism-fixture","seed":7,
            "identities":{"module":MAIN_SHA256,"runtime":FIXTURE_SHA256},
            "artifact":artifact["artifact"],"binary":input.cli,"axis":axis,
            "command":[std::env::current_exe().unwrap(),"--ignored","--exact",
                "campaign::counterfactual::produce_the_four_original_counterfactual_artifact_pairs"]
        })
    };
    let report = crate::stress::compare_counterfactual(
        &cell(&pair[0], "baseline"),
        &cell(&pair[1], &case.name),
    )
    .unwrap();
    write(
        &root.join(&case.name).join("first-divergence.json"),
        &report,
    );
    assert_eq!(report["status"], "diverged", "{report}");
    assert_eq!(report["ordering"], "helper-event");
    assertions::original_predicates(&case.name, &report, pair);
    let expected_kind = match case.name.as_str() {
        "draw" => "draw-counter",
        "clock" => "clock",
        "handoff" => "world-order",
        "output-clock" => "output",
        _ => unreachable!(),
    };
    assert_eq!(report["first"]["kind"], expected_kind, "{report}");
    assert!(report["first"]["coordinate"]["Shared"].as_u64().unwrap() > 0);
    assert_ne!(report["first"]["instance"].as_str().unwrap(), "");
    assert_eq!(report["first"]["incarnation"], 1);
    assert_ne!(report["first"]["expected"], report["first"]["actual"]);
    let command = report["reproducer"]["replay_command"]
        .as_array()
        .unwrap()
        .iter()
        .map(|value| value.as_str().unwrap().to_owned())
        .collect::<Vec<_>>();
    assert_eq!(command[0], input.cli.to_str().unwrap());
    assert!(report["reproducer"]["prefix_unavailable"].is_null());
    assert!(command.contains(&pair[0]["artifact"].as_str().unwrap().to_owned()));
    let mut provider_output = Vec::new();
    let mut provider_arguments = command[2..].to_vec();
    provider_arguments.splice(
        0..0,
        [
            "--debug-output".into(),
            root.join(&case.name)
                .join("provider-output")
                .to_string_lossy()
                .into_owned(),
        ],
    );
    assert_eq!(
        crate::cli::replay::run(&provider_arguments, &mut provider_output)
            .map_err(|error| error.message)
            .unwrap(),
        0
    );
    let inspection: Value = serde_json::from_slice(&provider_output).unwrap();
    assertions::inspected_prefix(&inspection, &report);
    write(
        &root.join(&case.name).join("provider-inspection.json"),
        &inspection,
    );
    authenticate_cli(&input.cli, &input.cli_sha256).unwrap();
    let output = std::process::Command::new(&command[0])
        .args(&command[1..])
        .output()
        .unwrap();
    let directory = root.join(&case.name);
    std::fs::write(directory.join("public-replay.stdout"), &output.stdout).unwrap();
    std::fs::write(directory.join("public-replay.stderr"), &output.stderr).unwrap();
    let receipt = json!({"command":command,"exit_code":output.status.code(),
        "stdout_sha256":canonical::digest(&output.stdout),"stderr_sha256":canonical::digest(&output.stderr),
        "cli_sha256":input.cli_sha256});
    write(&directory.join("public-replay.json"), &receipt);
    assert!(
        output.status.success(),
        "{receipt}: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    let replay: Value = serde_json::from_slice(&output.stdout).unwrap();
    assert_eq!(replay["status"], "inspected_prefix", "{replay}");
    assert_eq!(replay["prefix_verified"], true, "{replay}");
    assert_eq!(replay["witness_match"], true, "{replay}");
    assert_eq!(
        replay["resolved"]["choice"], report["reproducer"]["stop"]["stop"]["value"],
        "{replay}"
    );
    json!({"report":report,"public_replay":receipt,"result":replay})
}

struct PreparationInputs {
    installation_arguments: Vec<Vec<String>>,
    cli: PathBuf,
    cli_sha256: String,
    cases: Vec<PreparationCase>,
    output: PathBuf,
}
struct PreparationCase {
    name: String,
    options: crate::cli::explore::ExploreOptions,
}

#[test]
#[ignore = "requires the exact original fixture installation, compiler lease and private diagnostic request bounds"]
fn prepare_the_four_original_counterfactual_requests() {
    let path = PathBuf::from(std::env::var_os("GOMAD_CORE_COUNTERFACTUAL_PREPARATION").unwrap());
    let input = recipe::materialize(&path);
    authenticate_cli(&input.cli, &input.cli_sha256).unwrap();
    for arguments in input.installation_arguments {
        assert_eq!(
            crate::cli::install::run(&arguments, &mut Vec::new())
                .map_err(|error| error.message)
                .unwrap(),
            0
        );
    }
    let mut cases = Vec::new();
    for case in input.cases {
        let campaign = case.options;
        validate_campaign(&campaign, case.name == "handoff").unwrap();
        cases.push(json!({"name":case.name,
            "original_requested_choice_bytes":if case.name == "handoff" {0} else {4 << 20},
            "campaign": campaign}));
    }
    write(
        &input.output,
        &json!({"schema":"gomad.core-counterfactual-producer/v2",
        "cli":input.cli,"cli_sha256":input.cli_sha256,"cases":cases}),
    );
}

#[test]
#[ignore = "requires original authenticated fixture-runtime preparations and an explicitly granted genuine Go guest slot"]
fn produce_the_four_original_counterfactual_artifact_pairs() {
    let path = PathBuf::from(
        std::env::var_os("GOMAD_CORE_COUNTERFACTUAL_INPUT")
            .expect("missing original fixture input"),
    );
    let root = PathBuf::from(
        std::env::var_os("GOMAD_CORE_COUNTERFACTUAL_ROOT").expect("missing fresh fixture root"),
    );
    assert!(path.is_absolute() && root.is_absolute());
    let input: Inputs = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    assert_eq!(input.schema, "gomad.core-counterfactual-producer/v2");
    authenticate_cli(&input.cli, &input.cli_sha256).unwrap();
    let expected = BTreeSet::from(["draw", "clock", "handoff", "output-clock"]);
    assert_eq!(input.cases.len(), 4);
    assert_eq!(
        input
            .cases
            .iter()
            .map(|case| case.name.as_str())
            .collect::<BTreeSet<_>>(),
        expected
    );
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(&root)
        .unwrap();
    let mut rows = Vec::new();
    let mut original_failures = Vec::new();
    let mut original_module = None;
    for case in &input.cases {
        assert_eq!(
            case.original_requested_choice_bytes,
            if case.name == "handoff" { 0 } else { 4 << 20 }
        );
        let kind = match case.name.as_str() {
            "draw" => linked::FixtureFault::Draw,
            "clock" => linked::FixtureFault::Clock,
            "handoff" => linked::FixtureFault::Handoff,
            "output-clock" => linked::FixtureFault::OutputClock,
            _ => unreachable!(),
        };
        let (pair, failures) = produce_pair(case, &root, kind);
        let digest = pair[0]["module_sha256"].as_str().unwrap().to_owned();
        if let Some(expected) = &original_module {
            assert_eq!(&digest, expected);
        } else {
            original_module = Some(digest);
        }
        original_failures.extend(failures);
        let verification = verify_pair(&input, case, &root, &pair);
        rows.push(json!({"name":case.name,"original_requested_choice_bytes":case.original_requested_choice_bytes,"effective_runtime_trace_bytes":1 << 20,"diagnostics":true,"baseline":pair[0],"fault":pair[1],"verification":verification}));
    }
    write(
        &root.join("artifact-pairs.json"),
        &json!({"schema":"gomad.core-counterfactual-artifact-pairs/v1","status":if original_failures.is_empty() {"PASS"} else {"FAIL_ORIGINAL_PREDICATE"},"original_predicate_failures":original_failures,"qualification":false,"source_role":"determinism-fixture","helper":linked::IMPLEMENTATION_SHA256,"cli":input.cli,"cli_sha256":input.cli_sha256,"producer":std::env::current_exe().unwrap(),"fixture_runtime_sha256":FIXTURE_SHA256,"pairs":rows}),
    );
    assert!(
        original_failures.is_empty(),
        "original outcome predicates failed; diagnostic evidence remains at {}: {original_failures:?}",
        root.display()
    );
}
