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
    time::Duration,
};

pub(super) struct Fault {
    pub kind: linked::FixtureFault,
    pub owner: Option<Handle>,
}

pub(super) fn run(
    prepared: &Prepared,
    job: &ledger::Job,
    cancellation: &linked::Cancellation,
    fault: Fault,
) -> Result<execution::Result, String> {
    if linked::Conditions::compiled() != linked::Conditions::DeterminismFixtureReplay {
        return Err("counterfactual producer requires the compiled fixture role".into());
    }
    validate_preparation(prepared)?;
    execution::run_counterfactual(prepared, job, Duration::from_secs(60), cancellation, fault)?
        .finalized()
        .map_err(|failure| failure.message())
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
    receipt: PathBuf,
    receipt_sha256: String,
    original_requested_choice_bytes: u64,
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

fn produce_pair(
    prepared: &Prepared,
    case: &Case,
    root: &Path,
    kind: linked::FixtureFault,
) -> Vec<Value> {
    let store = super::store(root.join(&case.name).join("artifacts"), u64::MAX);
    let mut pair = Vec::new();
    for fault in [linked::FixtureFault::None, kind] {
        let result = run(
            prepared,
            &ledger::Job {
                ordinal: 0,
                seed: 7,
            },
            &linked::Cancellation::default(),
            Fault {
                kind: fault,
                owner: None,
            },
        )
        .unwrap();
        if prepared.scenario.is_some() {
            let summary: Value =
                serde_json::from_slice(&result.publication.payloads["evidence/cluster_summary"])
                    .unwrap();
            let incarnations = summary["incarnations"].as_array().unwrap();
            assert_eq!(incarnations.len(), 2);
            assert!(
                incarnations
                    .iter()
                    .all(|incarnation| incarnation["termination"] == "exit")
            );
        } else {
            assert_eq!(result.execution.record.outcome.termination, "exit");
            assert_eq!(result.execution.record.outcome.exit_code, Some(1));
        }
        assert_eq!(result.execution.record.outcome.class, "target_failure");
        assert!(
            result
                .publication
                .payloads
                .contains_key("evidence/event-order")
        );
        let published = store.publish(result.publication).unwrap();
        let opened = crate::evidence::store::Opened::open(&published.path).unwrap();
        opened.authenticate_all().unwrap();
        pair.push(json!({"artifact":published.path,"record_sha256":published.record_sha256,"bytes":published.stored_bytes}));
    }
    assert_ne!(pair[0]["record_sha256"], pair[1]["record_sha256"]);
    pair
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
    assert_eq!(input.schema, "gomad.core-counterfactual-producer/v1");
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
    let mut original_module = None;
    for case in input.cases {
        assert_eq!(
            case.original_requested_choice_bytes,
            if case.name == "handoff" { 0 } else { 4 << 20 }
        );
        let (_, prepared) =
            super::preparation::open_receipt(&case.receipt, &case.receipt_sha256).unwrap();
        validate_preparation(&prepared).unwrap();
        let digest = canonical::digest(&prepared.module);
        if let Some(expected) = &original_module {
            assert_eq!(&digest, expected);
        } else {
            original_module = Some(digest);
        }
        assert_eq!(prepared.scenario.is_some(), case.name == "handoff");
        assert_eq!(
            prepared
                .configuration
                .environment
                .contains_key("GOMAD_CHOICE_PROFILE"),
            case.name != "handoff"
        );
        let kind = match case.name.as_str() {
            "draw" => linked::FixtureFault::Draw,
            "clock" => linked::FixtureFault::Clock,
            "handoff" => linked::FixtureFault::Handoff,
            "output-clock" => linked::FixtureFault::OutputClock,
            _ => unreachable!(),
        };
        let pair = produce_pair(&prepared, &case, &root, kind);
        rows.push(json!({"name":case.name,"receipt":case.receipt,"receipt_sha256":case.receipt_sha256,"original_requested_choice_bytes":case.original_requested_choice_bytes,"effective_runtime_trace_bytes":1 << 20,"diagnostics":true,"baseline":pair[0],"fault":pair[1]}));
    }
    write(
        &root.join("artifact-pairs.json"),
        &json!({"schema":"gomad.core-counterfactual-artifact-pairs/v1","qualification":false,"source_role":"determinism-fixture","helper":linked::IMPLEMENTATION_SHA256,"cli":input.cli,"cli_sha256":input.cli_sha256,"producer":std::env::current_exe().unwrap(),"fixture_runtime_sha256":FIXTURE_SHA256,"pairs":rows}),
    );
}
