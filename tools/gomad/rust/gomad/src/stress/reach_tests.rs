use super::*;
use serde_json::{Value, json};
use std::fmt::Write as _;

fn event(order: u64, kind: &str) -> Value {
    json!({"type":"event","order":order,"kind":kind,"instance":"a","incarnation":1,"ordinal":order,"trace":17,"decision":8,"time_nanos":101,"value":"early"})
}
pub(super) fn framed(rows: &[Value]) -> Vec<u8> {
    let mut raw = format!(
        "gomad-wasm-event-order-v1 {}\n",
        json!({"type":"header","schema":"gomad.wasm-event-order/v1"})
    );
    for row in rows {
        writeln!(raw, "gomad-wasm-event-order-v1 {row}").unwrap();
    }
    let complete = json!({"type":"complete","events":rows.iter().filter(|row|row["type"]=="event").count(),"bytes":raw.len()});
    writeln!(raw, "gomad-wasm-event-order-v1 {complete}").unwrap();
    raw.into_bytes()
}

#[test]
fn diagnostic_extraction_preserves_partial_status_and_strict_artifact_reader() {
    let full = framed(&[event(1, "clock")]);
    let mut mixed = b"other helper telemetry\n".to_vec();
    mixed.extend(&full);
    assert!(report::parse(&mixed).is_err());
    let diagnostic = report::extract_diagnostic(&mixed).unwrap();
    assert_eq!(diagnostic["complete"], true);
    assert_eq!(diagnostic["events"].as_array().unwrap().len(), 1);
    let end = std::str::from_utf8(&full)
        .unwrap()
        .rfind("gomad-wasm-event-order-v1 ")
        .unwrap();
    for cut in [0, full.len() - 1, end] {
        let partial = report::extract_diagnostic(&full[..cut]).unwrap();
        assert_eq!(partial["complete"], false);
        assert_ne!(partial["reason"].as_str().unwrap(), "");
        assert!(report::parse(&full[..cut]).is_err());
    }
    let source = String::from_utf8(full).unwrap();
    for broken in [
        source.replace("\"kind\":\"clock\"", "\"kind\":\"clock\",\"extra\":true"),
        source.replacen("\"order\":1", "\"order\":2", 1),
        source.replace("\"instance\":\"a\"", "\"instance\":\"\""),
        source.replace("\"incarnation\":1", "\"incarnation\":0"),
        source.replace("\"events\":1", "\"events\":2"),
        format!(
            "{source}gomad-wasm-event-order-v1 {{\"type\":\"header\",\"schema\":\"gomad.wasm-event-order/v1\"}}\n"
        ),
        format!("{source}gomad-wasm-event-order-v1 {{"),
    ] {
        assert!(
            report::extract_diagnostic(broken.as_bytes()).is_err(),
            "{broken}"
        );
    }
}

#[test]
fn controller_zero_incarnation_is_reserved_and_never_a_guest_boundary() {
    for owner in ["guest", "a", "Controller", "controller ", ""] {
        let mut row = event(1, "model-transition");
        row["instance"] = owner.into();
        row["incarnation"] = 0.into();
        assert!(report::parse(&framed(&[row])).is_err(), "{owner}");
    }
    let mut controller = event(2, "model-transition");
    controller["instance"] = "controller".into();
    controller["incarnation"] = 0.into();
    controller["time_nanos"] = 1.into();
    controller["value"] = json!({"action":"controller_effect"});
    let full = framed(&[event(1, "choice"), controller]);
    let log = report::parse(&full).unwrap();
    assert!(report::first(&log, &log).is_none());
    let boundary = json!({"type":"boundary","order":0,"instance":"controller","incarnation":0,"trace":0,"decision":0,"time_nanos":0,"inspection_order":1});
    assert!(report::parse(&framed(&[boundary, event(1, "clock")])).is_err());
}

#[test]
fn native_first_difference_preserves_causes_coordinates_and_presence() {
    for (first_kind, later_kind) in [
        ("clock", "choice"),
        ("draw-counter", "choice"),
        ("world-order", "choice"),
        ("output", "clock"),
    ] {
        let a = vec![event(1, first_kind), event(2, later_kind)];
        let mut b = a.clone();
        b[0]["value"] = "changed".into();
        b[1]["value"] = "later".into();
        let difference = report::first(
            &report::parse(&framed(&a)).unwrap(),
            &report::parse(&framed(&b)).unwrap(),
        )
        .unwrap();
        assert_eq!(difference["kind"], first_kind);
        assert_eq!(difference["instance"], "a");
        assert_eq!(difference["incarnation"], 1);
        assert_eq!(difference["ordinal"], 1);
        assert_eq!(
            difference["coordinate"],
            json!({"Shared":1,"ChoiceOrdinal":17,"VirtualTime":101})
        );
        assert_ne!(difference["expected"], difference["actual"]);
        let prefix = report::parse(&framed(&a[..1])).unwrap();
        let full = report::parse(&framed(&a)).unwrap();
        let trailing = report::first(&prefix, &full).unwrap();
        assert_eq!(trailing["expected"], "absent");
        assert_eq!(trailing["ordinal"], 2);
        assert_eq!(trailing["coordinate"]["Shared"], 2);
    }
    let base = json!({"fresh":true,"workload":"runtime","seed":7,"identities":{"module":"a","helper":"b"}});
    for (field, value) in [
        ("workload", json!("cluster")),
        ("seed", json!(8)),
        ("identities", json!({"module":"a","helper":"c"})),
    ] {
        let mut changed = base.clone();
        changed[field] = value;
        assert!(comparable(&base, &changed).is_err());
    }
}

#[test]
fn failed_process_diagnostic_binds_raw_stderr_and_command_without_clobbering() {
    use std::fs;
    let root = std::env::temp_dir().join(format!("gomad-stress-diagnostic-{}", std::process::id()));
    fs::create_dir(&root).unwrap();
    let binary = root.join("gomad");
    fs::write(&binary, b"binary source").unwrap();
    fs::write(root.join("request.json"), b"request source").unwrap();
    let stderr = framed(&[event(1, "output")]);
    fs::write(root.join("stderr.log"), &stderr).unwrap();
    let path = process::failure_diagnostic(&root, &binary, 79, &json!({"exit":1})).unwrap();
    let report: Value = read(&path).unwrap();
    assert_eq!(report["diagnostic_only"], true);
    assert_eq!(report["process_id"], 79);
    assert_eq!(report["outcome"], json!({"exit":1}));
    assert_eq!(
        report["stderr"]["sha256"],
        crate::evidence::canonical::digest(&stderr)
    );
    assert_eq!(report["stderr"]["bytes"], stderr.len());
    assert_eq!(
        report["request_sha256"],
        crate::evidence::canonical::digest(b"request source")
    );
    assert_eq!(
        report["binary_sha256"],
        crate::evidence::canonical::digest(b"binary source")
    );
    assert_eq!(report["command"][0], binary.to_string_lossy().as_ref());
    assert_eq!(report["event_order"]["complete"], true);
    assert!(process::failure_diagnostic(&root, &binary, 79, &json!({"exit":2})).is_err());
    assert_eq!(fs::read(root.join("stderr.log")).unwrap(), stderr);
    assert_eq!(read::<Value>(&path).unwrap(), report);
    fs::remove_dir_all(root).unwrap();
}

#[test]
fn actual_controller_command_frames_preserve_controller_coordinates() {
    let raw = include_bytes!("testdata/controller-event-order.stderr");
    let diagnostic = report::extract_diagnostic(raw).unwrap();
    assert_eq!(diagnostic["complete"], true);
    let events = diagnostic["events"].as_array().unwrap();
    assert_eq!(events.len(), 2);
    assert_eq!(events[0]["instance"], "a");
    assert_eq!(events[0]["incarnation"], 1);
    assert_eq!(events[1]["instance"], "controller");
    assert_eq!(events[1]["incarnation"], 0);
    assert_eq!(events[1]["kind"], "model-transition");
    assert_eq!(events[1]["time_nanos"], 1);
    assert_eq!(events[1]["value"], json!({"action":"controller_effect"}));
}

#[test]
fn foreign_singleton_inspection_cannot_authorize_a_retained_reproducer() {
    let root = std::env::temp_dir().join(format!("gomad-stress-prefix-{}", std::process::id()));
    std::fs::create_dir(&root).unwrap();
    let marker = json!({"type":"boundary","order":0,"instance":"a","incarnation":1,"trace":0,"decision":0,"time_nanos":100,"inspection_order":2});
    let a = vec![marker, event(1, "clock")];
    let mut b = a.clone();
    b[1]["value"] = "shifted".into();
    let metadata = include_bytes!("testdata/native-inspection-v3.json").to_vec();
    let left = super::tests::publish_stress_fixture(
        &root,
        "left",
        1,
        b"target",
        framed(&a),
        Some(metadata.clone()),
        true,
    );
    let right = super::tests::publish_stress_fixture(
        &root,
        "right",
        1,
        b"target",
        framed(&b),
        Some(metadata),
        true,
    );
    let report = report::compare(&left, &right).unwrap();
    assert_eq!(report["status"], "undiagnosable");
    assert!(
        report["reproducer"]["prefix_unavailable"]
            .as_str()
            .unwrap()
            .contains("canonical retained inspection stream identity")
    );
    assert!(report["reproducer"].get("replay_command").is_none());
    std::fs::remove_dir_all(root).unwrap();
}

#[test]
fn five_native_perturbation_fixtures_report_the_first_semantic_field() {
    for (name, owner, kind, ordinal, field, left, right) in [
        (
            "single-perturbed-draw",
            "guest",
            "draw-counter",
            4,
            "runq_draws",
            json!({"runq_draws":8}),
            json!({"runq_draws":9}),
        ),
        (
            "single-shifted-clock",
            "guest",
            "clock",
            2,
            "operation",
            json!({"timestamp":30}),
            json!({"timestamp":31}),
        ),
        (
            "cluster-perturbed-draw",
            "b",
            "draw-counter",
            4,
            "runq_draws",
            json!({"runq_draws":8}),
            json!({"runq_draws":9}),
        ),
        (
            "cluster-shifted-clock",
            "a",
            "clock",
            2,
            "operation",
            json!({"timestamp":30}),
            json!({"timestamp":31}),
        ),
        (
            "cluster-reordered-handoff",
            "b",
            "world-order",
            2,
            "step",
            json!({"node":"b"}),
            json!({"node":"a"}),
        ),
    ] {
        let mut a = event(1, kind);
        a["instance"] = owner.into();
        a["ordinal"] = ordinal.into();
        a["value"] = left;
        let mut b = a.clone();
        b["value"] = right;
        let report = report::first(
            &report::parse(&framed(&[a])).unwrap(),
            &report::parse(&framed(&[b])).unwrap(),
        )
        .unwrap();
        assert_eq!(report["kind"], kind, "{name}");
        assert_eq!(report["instance"], owner, "{name}");
        assert_eq!(report["incarnation"], 1, "{name}");
        assert_eq!(report["ordinal"], ordinal, "{name}");
        assert_eq!(report["field"], field, "{name}");
        assert_ne!(report["expected"], report["actual"], "{name}");
    }
    for field in [
        "runq_draws",
        "scheduler_draws",
        "select_draws",
        "runtime_rand_draws",
        "runtime_cheaprand_draws",
        "timer_draws",
        "clock_tick_draws",
    ] {
        let mut a = event(1, "draw-counter");
        a["value"] = json!({field:0});
        let mut b = a.clone();
        b["value"][field] = 1.into();
        let report = report::first(
            &report::parse(&framed(&[a])).unwrap(),
            &report::parse(&framed(&[b])).unwrap(),
        )
        .unwrap();
        assert_eq!(report["field"], field);
    }
}

#[test]
fn diagnostic_only_virtual_time_difference_is_a_clock_before_later_effects() {
    let mut witness = event(1, "model-transition");
    witness["ordinal"] = 3.into();
    witness["trace"] = 3.into();
    witness["time_nanos"] = 100.into();
    witness["value"] = json!({"model":"runtime-witness","virtual_time":100,"state":{"bytes":128,"sha256":"expected"}});
    let mut changed = witness.clone();
    changed["value"]["virtual_time"] = 101.into();
    changed["value"]["state"]["sha256"] = "changed".into();
    let left = report::parse(&framed(&[witness, event(2, "choice")])).unwrap();
    let right = report::parse(&framed(&[changed, event(2, "choice")])).unwrap();
    let first = report::first(&left, &right).unwrap();
    assert_eq!(first["kind"], "clock");
    assert_eq!(first["field"], "virtual_time");
    assert_eq!(first["ordinal"], 3);
    assert_eq!(first["coordinate"]["ChoiceOrdinal"], 3);
    assert_eq!(first["coordinate"]["VirtualTime"], 100);
}

#[test]
fn stress_host_perturbations_do_not_change_declared_guest_inputs() {
    use std::{fs, os::unix::fs::PermissionsExt};
    let root =
        std::env::temp_dir().join(format!("gomad-stress-host-inputs-{}", std::process::id()));
    fs::create_dir(&root).unwrap();
    let binary = root.join("probe");
    fs::write(&binary,r"#!/usr/bin/python3
import json,os,pathlib,sys
request=json.loads(pathlib.Path(sys.argv[3]).read_text())
root=pathlib.Path(request['root'])
(root/'observation.json').write_text(json.dumps({'process_id':os.getpid(),'stress':os.getenv('GOMAD_WASM_STRESS'),'jitter':os.getenv('GOMAD_WASM_STRESS_JITTER'),'padding':os.getenv('GOMAD_STRESS_PADDING'),'tmp':os.getenv('TMPDIR'),'request':request}))
").unwrap();
    fs::set_permissions(&binary, fs::Permissions::from_mode(0o700)).unwrap();
    let mut json = false;
    let options = crate::cli::explore::parse(
        &[
            "--env".into(),
            "DECLARED=unchanged".into(),
            "go-run".into(),
            "./unused".into(),
        ],
        &mut json,
    )
    .unwrap();
    let before = serde_json::to_value(&options).unwrap();
    for (index, axis) in ["cpu", "concurrent", "layout", "handoff"]
        .into_iter()
        .enumerate()
    {
        let cell = root.join(axis);
        fs::create_dir(&cell).unwrap();
        let request = Request {
            workload: "runtime".into(),
            seed: 7,
            axis: axis.into(),
            options: options.clone(),
            root: cell,
            source_sha256: crate::evidence::canonical::digest(b"source"),
            target_pool: root.join("targets"),
            supervision: Supervision {
                cpu_workers: 0,
                cohort_size: 1,
                repetition: index,
                member: 0,
            },
        };
        let observed = process::execute(&binary, &request, index).unwrap();
        assert_eq!(observed["request"]["options"], before, "{axis}");
        assert_eq!(
            serde_json::to_value(&request.options).unwrap(),
            before,
            "{axis}"
        );
        assert_eq!(observed["stress"], "1");
        assert_eq!(
            observed["jitter"],
            if axis == "handoff" {
                json!("1")
            } else {
                Value::Null
            }
        );
        if axis == "layout" {
            assert_eq!(
                observed["padding"].as_str().unwrap().len(),
                1 + index % 16 * 4096
            );
            assert!(
                observed["tmp"]
                    .as_str()
                    .unwrap()
                    .starts_with(request.root.to_str().unwrap())
            );
        } else {
            assert!(observed["padding"].is_null());
        }
    }
    fs::remove_dir_all(root).unwrap();
}

#[test]
fn empty_or_binary_telemetry_is_diagnostic_only_and_never_a_complete_artifact() {
    let empty = framed(&[]);
    assert!(report::parse(&empty).is_err());
    let diagnostic = report::extract_diagnostic(&empty).unwrap();
    assert_eq!(diagnostic["complete"], true);
    assert_eq!(diagnostic["events"], json!([]));
    let mut mixed = vec![255, b'\n'];
    mixed.extend(framed(&[event(1, "clock")]));
    assert!(report::parse(&mixed).is_err());
    assert_eq!(
        report::extract_diagnostic(&mixed).unwrap()["complete"],
        true
    );
}

#[test]
fn retained_runs_without_declared_diagnostics_are_undiagnosable() {
    let root = std::env::temp_dir().join(format!(
        "gomad-stress-no-diagnostics-{}",
        std::process::id()
    ));
    std::fs::create_dir(&root).unwrap();
    let left = super::tests::publish_stress_fixture(
        &root,
        "left",
        1,
        b"target",
        framed(&[event(1, "clock")]),
        None,
        false,
    );
    let right = super::tests::publish_stress_fixture(
        &root,
        "right",
        1,
        b"target",
        framed(&[event(1, "clock")]),
        None,
        false,
    );
    let report = report::compare(&left, &right).unwrap();
    assert_eq!(report["status"], "undiagnosable");
    assert!(report["reason"].as_str().unwrap().contains("--diagnostics"));
    assert!(report.get("first").is_none());
    std::fs::remove_dir_all(root).unwrap();
}

fn multi_inspection(change: &str) -> Vec<u8> {
    use gomad_wasmhost::linked_inspection::Evidence;
    let mut evidence =
        Evidence::decode(include_bytes!("testdata/multi-inspection-v3.json")).unwrap();
    let first_bytes =
        u32::from_le_bytes(evidence.boundary_frames[12..16].try_into().unwrap()) as usize;
    let second = 16 + first_bytes + 4;
    let cut = second + 65;
    assert_eq!(&evidence.boundary_frames[cut + 69..cut + 73], &[0; 4]);
    match change {
        "recovery" => evidence.boundary_frames[cut + 137] = 1,
        "witness" => evidence.boundary_frames[cut + 169 + 23] = 1,
        "output" => evidence.incarnations[0].streams.stdout[0] = b'X',
        "choice" => evidence.incarnations[0].streams.choice[2 * 96 + 16] = 1,
        "cut" => evidence.boundary_frames[cut + 53..cut + 61].copy_from_slice(&3u64.to_le_bytes()),
        "single" => {
            evidence.boundary_frames.truncate(16 + first_bytes);
            evidence.boundary_frames[8..12].copy_from_slice(&1u32.to_le_bytes());
            evidence
                .world
                .truncate(usize::try_from(evidence.boundaries[0].world_bytes).unwrap());
            let mut lifecycle = evidence.lifecycle[..36].to_vec();
            lifecycle.extend(&evidence.lifecycle[64..78]);
            evidence.lifecycle = lifecycle;
            evidence.incarnations.truncate(1);
            evidence.incarnations[0].terminal_order = 3;
        }
        "" => {}
        other => panic!("unknown inspection mutation {other}"),
    }
    let raw = serde_json::to_vec(&evidence).unwrap();
    Evidence::decode(&raw).unwrap();
    raw
}

#[test]
fn native_inspection_diagnostic_orders_real_multi_owner_cuts_and_terminal_unknowns() {
    use gomad_wasmhost::linked_inspection::Evidence;
    let base = multi_inspection("");
    let expected = Evidence::decode(&base).unwrap();
    assert_eq!(expected.boundaries[0].trace, 17);
    assert_eq!(expected.boundaries[0].decision, 0);
    for (change, field, shared) in [
        ("recovery", "recovery-state", 4),
        ("witness", "runtime-witness", 4),
        ("cut", "output:stdout", 4),
        ("output", "output:stdout", 2),
        ("choice", "choice", 2),
    ] {
        let raw = multi_inspection(change);
        let observed = Evidence::decode(&raw).unwrap();
        let report =
            inspection::diagnostic(&expected, &observed, &raw, b"world", b"world").unwrap();
        assert_eq!(report["first"]["field"], field, "{change}");
        assert_eq!(report["first"]["coordinate"]["Shared"], shared, "{change}");
        assert_eq!(report["first"]["instance"], "a");
        assert_eq!(report["first"]["incarnation"], 1);
        assert_eq!(report["first"]["coordinate_availability"]["event"], false);
        assert_ne!(report["first"]["expected"], report["first"]["actual"]);
        if shared == 4 {
            assert_eq!(report["preceding_verified_prefix"]["decision"], 0);
            assert_eq!(report["preceding_verified_prefix"]["trace"], 17);
        }
        if change == "choice" {
            assert_eq!(report["first"]["coordinate"]["ChoiceOrdinal"], 2);
        }
    }
    let single = Evidence::decode(&multi_inspection("single")).unwrap();
    let presence = inspection::diagnostic(&single, &expected, &base, b"world", b"world").unwrap();
    assert_eq!(presence["first"]["field"], "incarnation-presence");
    assert_eq!(presence["first"]["incarnation"], 2);
    assert_eq!(presence["first"]["coordinate"]["Shared"], 3);
    assert_eq!(presence["virtual_time_available"], false);
    assert!(presence["first"]["coordinate"]["VirtualTime"].is_null());
    assert!(presence["first"]["coordinate"]["ChoiceOrdinal"].is_null());
    let suffix = inspection::diagnostic(&expected, &expected, &base, b"world", b"world!").unwrap();
    assert_eq!(suffix["first"]["field"], "world-tail");
    assert_eq!(suffix["first"]["coordinate"]["Shared"], 6);
    assert_eq!(
        suffix["first"]["coordinate_relation"],
        "retained-terminal-upper-bound"
    );
    assert_eq!(suffix["virtual_time_available"], false);
    assert!(suffix["first"]["coordinate"]["VirtualTime"].is_null());
    assert_eq!(suffix["preceding_verified_prefix"]["order"], 4);
}

#[test]
fn retained_metadata_diagnostic_is_authenticated_and_never_replaces_strict_artifact_status() {
    use gomad_wasmhost::linked_inspection::Evidence;
    let root = std::env::temp_dir().join(format!(
        "gomad-inspection-diagnostic-{}",
        std::process::id()
    ));
    std::fs::create_dir(&root).unwrap();
    let mut expected =
        Evidence::decode(include_bytes!("testdata/native-inspection-v3.json")).unwrap();
    expected.incarnations[0].streams.choice = vec![0; 96];
    let base = serde_json::to_vec(&expected).unwrap();
    let at = expected.boundary_frames.len() - 128 - 32;
    expected.boundary_frames[at] = 1;
    let changed = serde_json::to_vec(&expected).unwrap();
    Evidence::decode(&changed).unwrap();
    let left = super::tests::publish_stress_fixture(
        &root,
        "left",
        1,
        b"target",
        b"malformed supplemental event order\n".to_vec(),
        Some(base),
        true,
    );
    let right = super::tests::publish_stress_fixture(
        &root,
        "right",
        1,
        b"target",
        b"malformed supplemental event order\n".to_vec(),
        Some(changed),
        true,
    );
    let report = report::compare(&left, &right).unwrap();
    assert_eq!(report["status"], "undiagnosable");
    assert!(report.get("first").is_none());
    assert_eq!(report["inspection_diagnostic"]["diagnostic_only"], true);
    assert_eq!(
        report["inspection_diagnostic"]["first"]["field"],
        "recovery-state"
    );
    assert_eq!(
        report["inspection_diagnostic"]["first"]["coordinate"]["Shared"],
        2
    );
    // The same decoder-valid metadata must not substitute a foreign choice stream.
    expected.incarnations[0].streams.choice.clear();
    let foreign = super::tests::publish_stress_fixture(
        &root,
        "foreign",
        1,
        b"target",
        Vec::new(),
        Some(serde_json::to_vec(&expected).unwrap()),
        true,
    );
    let rejected = report::compare(&left, &foreign).unwrap();
    assert!(rejected.get("inspection_diagnostic").is_none());
    assert!(
        rejected["inspection_diagnostic_unavailable"]
            .as_str()
            .unwrap()
            .contains("canonical retained inspection stream identity")
    );
    std::fs::remove_dir_all(root).unwrap();
}

#[test]
fn native_inspection_diagnostics_refuse_incomplete_or_excessive_comparison_work() {
    use gomad_wasmhost::linked_inspection::Evidence;
    let raw = multi_inspection("");
    let expected = Evidence::decode(&raw).unwrap();
    let mut incomplete = expected.clone();
    incomplete.complete = false;
    assert!(
        inspection::diagnostic(&expected, &incomplete, &raw, &[], &[])
            .unwrap_err()
            .contains("incomplete")
    );
    let world = vec![0; 33 << 20];
    assert!(
        inspection::diagnostic(&expected, &expected, &raw, &world, &world)
            .unwrap_err()
            .contains("capacity")
    );
}

#[test]
fn literal_reproducer_selects_the_authenticated_compiled_engine_role() {
    use gomad_wasmhost::linked_inspection::Evidence;
    let root = std::env::temp_dir().join(format!("gomad-reproducer-role-{}", std::process::id()));
    std::fs::create_dir(&root).unwrap();
    let mut metadata =
        Evidence::decode(include_bytes!("testdata/native-inspection-v3.json")).unwrap();
    metadata.incarnations[0].streams.choice = vec![0; 96];
    let metadata = serde_json::to_vec(&metadata).unwrap();
    let marker = json!({"type":"boundary","order":0,"instance":"a","incarnation":1,"trace":0,"decision":0,"time_nanos":100,"inspection_order":2});
    let left_rows = vec![marker, event(1, "clock")];
    let mut right_rows = left_rows.clone();
    right_rows[1]["value"] = json!("changed");
    let configuration = json!({"engine":linked::Conditions::compiled().configuration(),"model":{"mode":1,"diagnostics":true}});
    let publish = |name: &str, rows: &[Value], configuration: Value| {
        super::tests::publish_stress_configuration(
            &root,
            name,
            b"target",
            framed(rows),
            Some(metadata.clone()),
            &configuration,
        )
    };
    let left = publish("left", &left_rows, configuration.clone());
    let right = publish("right", &right_rows, configuration.clone());
    let report = report::compare(&left, &right).unwrap();
    #[cfg(feature = "helper-determinism-fixture")]
    assert_eq!(
        super::compare_counterfactual(&left, &right).unwrap(),
        report
    );
    assert_eq!(report["status"], "diverged");
    let command = report["reproducer"]["replay_command"].as_array().unwrap();
    assert_eq!(report["reproducer"]["stop"]["stop"]["value"], 0);
    assert!(
        command
            .windows(2)
            .any(|pair| pair == [json!("--stop"), json!("choice=0")])
    );
    match linked::Conditions::compiled() {
        linked::Conditions::Ordinary => assert!(
            !command
                .iter()
                .any(|arg| arg == "--stress-jitter" || arg == "--determinism-fixture")
        ),
        linked::Conditions::StressReplay => assert!(command.contains(&json!("--stress-jitter"))),
        linked::Conditions::DeterminismFixtureReplay => {
            assert!(command.contains(&json!("--determinism-fixture")));
        }
        linked::Conditions::DiagnosticProfile => panic!("no public diagnostic-profile replay flag"),
    }
    let mut unknown = configuration;
    unknown["engine"] = json!(format!(
        "{}-unknown",
        linked::Conditions::compiled().configuration()
    ));
    let left = publish("unknown-left", &left_rows, unknown.clone());
    let right = publish("unknown-right", &right_rows, unknown);
    let rejected = report::compare(&left, &right).unwrap();
    #[cfg(feature = "helper-determinism-fixture")]
    assert!(
        super::compare_counterfactual(&left, &right)
            .unwrap_err()
            .contains("replay role")
    );
    assert_eq!(rejected["status"], "undiagnosable");
    assert!(rejected["reproducer"].get("replay_command").is_none());
    assert!(
        rejected["reproducer"]["prefix_unavailable"]
            .as_str()
            .unwrap()
            .contains("replay role")
    );
    std::fs::remove_dir_all(root).unwrap();
}

#[test]
fn retained_replay_roles_require_exact_engines_and_cluster_agreement() {
    let engine = linked::Conditions::compiled().configuration();
    assert!(
        report::configuration_role(&json!({"engine":engine}), false).unwrap()
            == linked::Conditions::compiled()
    );
    assert!(
        report::configuration_role(
            &json!({"incarnations":[{"engine":engine},{"engine":engine}]}),
            true
        )
        .unwrap()
            == linked::Conditions::compiled()
    );
    for config in [
        json!({"incarnations":[]}),
        json!({"incarnations":[{"engine":engine},{"engine":"unknown"}]}),
        json!({"incarnations":[{}]}),
    ] {
        assert!(report::configuration_role(&config, true).is_err());
    }
    for role in [
        linked::Conditions::Ordinary,
        linked::Conditions::StressReplay,
        linked::Conditions::DeterminismFixtureReplay,
    ] {
        if role != linked::Conditions::compiled() {
            assert!(
                report::configuration_role(&json!({"engine":role.configuration()}), false)
                    .err()
                    .unwrap()
                    .contains("compiled replay role")
            );
        }
    }
    assert!(
        report::configuration_role(&json!({"engine":format!("{engine}-unknown")}), false)
            .err()
            .unwrap()
            .contains("unknown retained replay role")
    );
}
