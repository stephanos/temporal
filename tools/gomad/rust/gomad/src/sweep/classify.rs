#[cfg(test)]
mod tests {
    use super::*;
    use std::fmt::Write as _;

    #[test]
    fn native_typed_failures_preserve_category_with_opaque_reasons() {
        for (category, class) in [
            ("unsupported", Class::RefusedImport),
            ("capacity", Class::CapacityExceeded),
            ("infrastructure", Class::Infrastructure),
            ("future_category", Class::Infrastructure),
        ] {
            let event = serde_json::json!({"schema":"gomad.explore-event/v3","type":"error","classification":"runner_failure","message":format!("backend {category}: opaque diagnostic")});
            let input = Observation {
                stdout: serde_json::to_vec(&event).unwrap(),
                ..Observation::default()
            };
            let observed = classify(&input);
            assert_eq!(observed.class, class);
            assert!(observed.reason.contains("opaque diagnostic"));
        }
        let input=Observation{stdout:serde_json::to_vec(&serde_json::json!({"schema":"gomad.explore-event/v3","type":"error","classification":"runner_failure","message":"compiler command failed: # fixture\n/abs/fixture.go:9: undefined: missing"})).unwrap(),..Observation::default()};
        assert_eq!(classify(&input).class, Class::BuildFailure);
    }
    #[test]
    fn class_order_is_the_nine_observation_dispositions() {
        assert_eq!(
            serde_json::to_value(CLASSES).unwrap(),
            serde_json::json!([
                "pass",
                "guest-failure",
                "refused-import",
                "capacity-exceeded",
                "timeout",
                "build-failure",
                "infrastructure",
                "replay-divergence",
                "not-run"
            ])
        );
    }
    #[test]
    fn backend_categories_are_only_read_from_the_native_error_prefix() {
        for (message, class) in [
            (
                "opaque diagnostic containing backend unsupported: socket",
                Class::Infrastructure,
            ),
            ("backend unsupported: socket", Class::RefusedImport),
            ("backend future-category: socket", Class::Infrastructure),
        ] {
            let result = classify_error(&Event {
                classification: "runner_failure".into(),
                message: message.into(),
                ..Event::default()
            });
            assert_eq!(result.class, class);
            if class == Class::Infrastructure {
                assert!(result.reason.contains("backend"));
            }
        }
    }

    fn classify_trusted(observation: &Observation) -> Outcome {
        let events: Vec<Event> = observation
            .stdout
            .split(|byte| *byte == b'\n')
            .filter(|line| !line.is_empty())
            .map(|line| serde_json::from_slice(line).unwrap())
            .collect();
        classify_native(observation, &events)
    }

    fn observation(classification: &str, facts: ArtifactFacts) -> Observation {
        let event = |kind: &str| Event {
            schema: "gomad.explore-event/v3".into(),
            r#type: kind.into(),
            classification: classification.into(),
            path: "artifact".into(),
            ..Event::default()
        };
        Observation {
            test: "TestSuite/TestOne".into(),
            stdout: [
                serde_json::to_vec(&event("result")).unwrap(),
                serde_json::to_vec(&event("artifact")).unwrap(),
            ]
            .join(b"\n".as_slice()),
            artifacts: [("artifact".into(), facts)].into(),
            ..Observation::default()
        }
    }

    #[test]
    fn selected_method_verdict_is_required() {
        for (selection, output, class, skipped) in [
            (
                "TestSuite/TestOne",
                "    --- PASS: TestSuite/TestOne (0.01s)\n--- PASS: TestSuite (0.01s)",
                Class::Pass,
                false,
            ),
            (
                "TestSuite/.*/TestOne",
                "        --- PASS: TestSuite/variant/TestOne (0.01s)",
                Class::Pass,
                false,
            ),
            (
                "TestSuite/TestOne",
                "    --- PASS: TestSuite/TestOther (0.01s)\n--- PASS: TestSuite (0.01s)",
                Class::Infrastructure,
                false,
            ),
            (
                "TestSuite/TestOne",
                "        --- PASS: TestSuite/TestOne/case (0.01s)",
                Class::Infrastructure,
                false,
            ),
            (
                "TestSuite/TestOne",
                "    --- SKIP: TestSuite/TestOne (0.01s)",
                Class::Pass,
                true,
            ),
        ] {
            let mut input = observation(
                "success",
                ArtifactFacts {
                    stdout: output.as_bytes().into(),
                    ..ArtifactFacts::default()
                },
            );
            input.test = selection.into();
            let outcome = classify_trusted(&input);
            assert_eq!(
                (outcome.class, outcome.skipped),
                (class, skipped),
                "{output}"
            );
        }
    }

    #[test]
    fn failure_classes_and_reason_precedence() {
        for (classification, message, class, reason) in [
            (
                "runner_failure",
                "backend unsupported: sock_accept not modeled",
                Class::RefusedImport,
                "execute: sock_accept not modeled",
            ),
            (
                "runner_failure",
                "backend capacity: fuel exhausted after 1125899906842624 units",
                Class::CapacityExceeded,
                "fuel exhausted after N units",
            ),
            (
                "runner_failure",
                "backend trap: out of bounds",
                Class::GuestFailure,
                "trap: out of bounds",
            ),
            (
                "runner_failure",
                "backend divergence: choice ordinal 1234",
                Class::ReplayDivergence,
                "choice ordinal N",
            ),
            (
                "runner_failure",
                "backend infrastructure: helper exited",
                Class::Infrastructure,
                "backend infrastructure: helper exited",
            ),
            (
                "runner_failure",
                "backend invalid: invalid module",
                Class::Infrastructure,
                "backend invalid: invalid module",
            ),
            (
                "unsupported_target",
                "requires os/exec",
                Class::RefusedImport,
                "prepare: requires os/exec",
            ),
            (
                "capacity",
                "artifact exceeds 4294967296",
                Class::CapacityExceeded,
                "artifact or journal capacity: artifact exceeds N",
            ),
            (
                "runner_failure",
                "choice_trace_overflow",
                Class::CapacityExceeded,
                "choice trace overflow",
            ),
            (
                "runner_failure",
                "WASM compiler command failed: # package\n/abs/checkout/build_util.go:17:2: undefined: exec.LookPath",
                Class::BuildFailure,
                "build_util.go: undefined: exec.LookPath",
            ),
            (
                "runner_failure",
                "WASM compiler command failed: ",
                Class::Infrastructure,
                "compiler command failed without a diagnostic (provider compiler limit or host failure)",
            ),
        ] {
            let event = Event {
                schema: "gomad.explore-event/v3".into(),
                r#type: "error".into(),
                classification: classification.into(),
                message: message.into(),
                ..Event::default()
            };
            let input = Observation {
                stdout: serde_json::to_vec(&event).unwrap(),
                ..Observation::default()
            };
            let outcome = classify_trusted(&input);
            assert_eq!((outcome.class, outcome.reason.as_str()), (class, reason));
        }
    }

    #[test]
    fn retained_artifact_facts_control_verdict() {
        for (classification, facts, class, reason) in [
            ("success", ArtifactFacts { stdout: b"testing: warning: no tests to run\nPASS".to_vec(), ..ArtifactFacts::default() }, Class::BuildFailure, "no tests to run"),
            ("success", ArtifactFacts { stdout_truncated: true, ..ArtifactFacts::default() }, Class::Infrastructure, "stdout truncated"),
            ("success", ArtifactFacts { read_error: "unreadable".into(), ..ArtifactFacts::default() }, Class::Infrastructure, "open success artifact"),
            ("target_failure", ArtifactFacts { reason: "world_capacity".into(), ..ArtifactFacts::default() }, Class::CapacityExceeded, "world capacity"),
            ("mixed_failure", ArtifactFacts { reason: "world_replay_divergence".into(), ..ArtifactFacts::default() }, Class::ReplayDivergence, "world replay divergence"),
            ("target_failure", ArtifactFacts { kind: "gomad.watchdog-timeout/v1".into(), ..ArtifactFacts::default() }, Class::Timeout, "execution-timeout"),
            ("watchdog_observation", ArtifactFacts::default(), Class::Timeout, "execution-timeout"),
            ("replay_divergence", ArtifactFacts { reason: "replay_mismatch".into(), ..ArtifactFacts::default() }, Class::ReplayDivergence, "diverged: replay_mismatch"),
            ("target_failure", ArtifactFacts { exit_code: Some(2), stderr: b"panic: address 0xc000123456".to_vec(), stdout: b"  Error: other error".to_vec(), ..ArtifactFacts::default() }, Class::GuestFailure, "exit N: panic: address 0x?"),
            ("target_failure", ArtifactFacts { exit_code: Some(1), stdout: b"  Error: Received unexpected error:\n    --- FAIL: TestSuite/TestOne/case (0.10s)".to_vec(), ..ArtifactFacts::default() }, Class::GuestFailure, "exit N: Received unexpected error:"),
        ] {
            let outcome = classify_trusted(&observation(classification, facts));
            assert_eq!(outcome.class, class);
            assert!(outcome.reason.contains(reason), "{} must contain {reason}", outcome.reason);
        }
        let mut input = observation("success", ArtifactFacts::default());
        input.artifacts.clear();
        assert!(
            classify_trusted(&input)
                .reason
                .contains("without a retained success artifact")
        );
        input.exec_error = "signal: killed".into();
        assert_eq!(
            classify_trusted(&input).reason,
            "gomad subprocess: signal: killed"
        );
    }

    #[test]
    fn subtests_require_indentation_and_selected_parent_and_cap_failed_names() {
        let mut output = String::from(
            "--- PASS: TestSuite/TestOne (0s)\n    --- PASS: TestSuite/TestOne/pass (0s)\n    --- SKIP: TestSuite/TestOne/skip (0s)\n    --- FAIL: TestSuite/TestOther/unrelated (0s)\n",
        );
        for i in 0..25 {
            writeln!(output, "    --- FAIL: TestSuite/TestOne/case{i} (0s)").unwrap();
        }
        let outcome = classify_trusted(&observation(
            "success",
            ArtifactFacts {
                stdout: output.into_bytes(),
                ..ArtifactFacts::default()
            },
        ));
        let subtests = outcome.subtests.unwrap();
        assert_eq!(
            (
                subtests.pass,
                subtests.skip,
                subtests.fail,
                subtests.failed.len()
            ),
            (1, 1, 25, 20)
        );
    }

    #[test]
    fn first_error_last_result_and_first_failure_artifact_win() {
        let event = |kind: &str, classification: &str, path: &str| Event {
            schema: "gomad.explore-event/v3".into(),
            r#type: kind.into(),
            classification: classification.into(),
            path: path.into(),
            ..Event::default()
        };
        let mut input = observation("success", ArtifactFacts::default());
        input.stdout = [
            event("result", "unknown", ""),
            event("result", "target_failure", ""),
            event("artifact", "success", "artifact"),
            event("artifact", "target_failure", "failed"),
            event("artifact", "target_failure", "later"),
        ]
        .iter()
        .map(|event| serde_json::to_vec(event).unwrap())
        .collect::<Vec<_>>()
        .join(b"\n".as_slice());
        input.artifacts.insert(
            "failed".into(),
            ArtifactFacts {
                reason: "world_capacity".into(),
                module_sha256: "module".into(),
                ..ArtifactFacts::default()
            },
        );
        input.artifacts.insert(
            "later".into(),
            ArtifactFacts {
                reason: "world_replay_divergence".into(),
                ..ArtifactFacts::default()
            },
        );
        let result = classify_trusted(&input);
        assert_eq!(result.class, Class::CapacityExceeded);
        assert_eq!(result.artifact_path, "failed");
        assert_eq!(result.module_sha256, "module");
        input.stdout.extend_from_slice(b"\n{\"schema\":\"gomad.explore-event/v3\",\"type\":\"error\",\"classification\":\"overall_timeout\"}\n{\"schema\":\"gomad.explore-event/v3\",\"type\":\"error\",\"classification\":\"unsupported_target\"}");
        assert_eq!(classify_trusted(&input).class, Class::Timeout);
        let result = classify_trusted(&Observation {
            exit: 3,
            stderr: b"\nhandshake failed\n".to_vec(),
            ..Observation::default()
        });
        assert_eq!(
            result.reason,
            "gomad exit 3 without a result event: handshake failed"
        );
    }

    #[test]
    fn parse_events_accepts_only_current_native_reports() {
        let native = serde_json::json!({"type":"result", "classification":"success", "campaign_path":"campaign", "selection_count":1,"attempted":1,"succeeded":1,"failures":0,"watchdogs":0,"cancelled":0,"distinct_failures":0,"stop_reason":"exhausted","success_artifacts":[{"seed":1,"path":"artifact","record_hash":"hash"}],"retained_evidence":1,"retained_bytes":123});
        let bytes = serde_json::to_vec(&native).unwrap();
        assert_eq!(parse_events(&bytes).unwrap()[0].attempted, 1);
        assert_eq!(
            super::classify(&Observation {
                stdout: bytes,
                ..Observation::default()
            })
            .class,
            Class::Infrastructure
        );
        for key in [
            "schema",
            "partial",
            "missing",
            "shards",
            "plan_sha256",
            "extra",
        ] {
            let mut invalid = native.clone();
            invalid[key] = serde_json::json!("unexpected");
            assert!(
                parse_events(&serde_json::to_vec(&invalid).unwrap()).is_err(),
                "{key}"
            );
        }
        for bytes in [b"{\"schema\":\"gomad.explore-event/v3\",\"type\":\"result\",\"classification\":\"success\"}".as_slice(), b"{\"schema\":\"gomad.explore-event/v3\",\"type\":\"artifact\",\"path\":\"artifact\"}", b"not json"] {
            assert!(parse_events(bytes).is_err());
        }
        assert!(parse_events(b"{\"schema\":\"gomad.explore-event/v3\",\"type\":\"error\",\"classification\":\"capacity\",\"message\":\"full\"}").is_ok());
        assert!(
            parse_events(&vec![b'x'; 16 << 20])
                .unwrap_err()
                .contains("maximum size")
        );
    }

    #[test]
    fn normalize_preserves_go_grouping_and_boundaries() {
        for (input, expected) in [
            (
                "panic at 0xc000123456 goroutine 17",
                "panic at 0x? goroutine N",
            ),
            (
                "open /tmp/a/b/sqlite.db: no such file",
                "open <path>/sqlite.db: no such file",
            ),
            (
                "workflow 4f1c2d3e-0000-4000-8000-1234567890ab timed out after 30s",
                "workflow <uuid> timed out after N",
            ),
            (
                "module sha256:abcdefabcdefabcdef mismatch",
                "module <hex> mismatch",
            ),
            ("module 1234567890123456 mismatch", "module N mismatch"),
            (
                "--- FAIL: TestSuite/TestMethod (1.00s)",
                "--- FAIL: TestSuite/TestMethod (N)",
            ),
            ("a123 123a 123_ _123 1µs 1KiB", "a123 123a 123_ _123 N N"),
            ("open '/one/two/'", "open '<path>/two'"),
        ] {
            assert_eq!(normalize(input), expected, "{input}");
        }
        assert_eq!(
            normalize(&"界".repeat(300)),
            format!("{}…", "界".repeat(240))
        );
    }
}

use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

#[derive(Clone, Copy, Debug, Default, Deserialize, Eq, PartialEq, Ord, PartialOrd, Serialize)]
#[serde(rename_all = "kebab-case")]
pub enum Class {
    Pass,
    GuestFailure,
    RefusedImport,
    CapacityExceeded,
    Timeout,
    BuildFailure,
    Infrastructure,
    ReplayDivergence,
    #[default]
    NotRun,
}

#[cfg(test)]
const CLASSES: [Class; 9] = [
    Class::Pass,
    Class::GuestFailure,
    Class::RefusedImport,
    Class::CapacityExceeded,
    Class::Timeout,
    Class::BuildFailure,
    Class::Infrastructure,
    Class::ReplayDivergence,
    Class::NotRun,
];

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
#[serde(default)]
pub struct Event {
    pub schema: String,
    pub r#type: String,
    pub classification: String,
    pub message: String,
    pub campaign_path: String,
    pub path: String,
    pub attempted: u64,
    pub succeeded: u64,
    pub failures: u64,
    pub watchdogs: u64,
    pub replay_divergences: u64,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
#[serde(default)]
pub struct ArtifactFacts {
    pub kind: String,
    pub module_sha256: String,
    pub domain: String,
    pub reason: String,
    pub termination: String,
    pub exit_code: Option<u64>,
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
    pub stdout_truncated: bool,
    pub stored_bytes: u64,
    pub read_error: String,
}

#[derive(Clone, Debug, Default)]
pub struct Observation {
    pub test: String,
    pub exit: i32,
    pub exec_error: String,
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
    pub artifacts: BTreeMap<String, ArtifactFacts>,
}

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Subtests {
    pub pass: usize,
    pub fail: usize,
    pub skip: usize,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub failed: Vec<String>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct Outcome {
    pub class: Class,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub reason: String,
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub skipped: bool,
    #[serde(skip)]
    pub artifact_path: String,
    #[serde(skip)]
    pub module_sha256: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub subtests: Option<Subtests>,
}

fn outcome(class: Class, reason: impl Into<String>) -> Outcome {
    Outcome {
        class,
        reason: reason.into(),
        ..Outcome::default()
    }
}

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct NativeSuccessArtifact {
    pub seed: u64,
    pub path: String,
    pub record_hash: String,
}

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct NativeArtifactPresentation {
    pub path: String,
    pub record_hash: String,
    pub replay_argv: Vec<String>,
    pub replay_command: String,
}
#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct NativeResult {
    pub r#type: String,
    pub classification: String,
    pub campaign_path: String,
    pub selection_count: u64,
    pub attempted: u64,
    pub succeeded: u64,
    pub failures: u64,
    pub watchdogs: u64,
    pub cancelled: u64,
    pub distinct_failures: u64,
    pub stop_reason: String,
    pub success_artifacts: Vec<NativeSuccessArtifact>,
    pub retained_evidence: u64,
    pub retained_bytes: u64,
    pub artifacts: Option<Vec<NativeArtifactPresentation>>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct NativeError {
    schema: String,
    r#type: String,
    classification: String,
    message: String,
}

pub fn parse_events(stdout: &[u8]) -> Result<Vec<Event>, String> {
    let mut events = Vec::new();
    let mut have_result = false;
    for line in stdout.split(|byte| *byte == b'\n') {
        if line.len() >= 16 << 20 {
            return Err("explore stdout line exceeds maximum size".into());
        }
        let line = std::str::from_utf8(line)
            .map_err(|error| format!("explore stdout is not UTF-8: {error}"))?
            .trim();
        if line.is_empty() {
            continue;
        }
        let error = || {
            format!(
                "explore stdout line is not a native explore report: {}",
                line.chars().take(200).collect::<String>()
            )
        };
        let value: serde_json::Value = serde_json::from_str(line).map_err(|_| error())?;
        match value.get("type").and_then(serde_json::Value::as_str) {
            Some("result") => {
                let result: NativeResult = serde_json::from_str(line).map_err(|_| error())?;
                if have_result
                    || result.r#type != "result"
                    || !matches!(
                        result.classification.as_str(),
                        "success" | "target_failure" | "mixed_failure" | "watchdog_observation"
                    )
                {
                    return Err(error());
                }
                have_result = true;
                events.push(Event {
                    r#type: result.r#type,
                    classification: result.classification,
                    campaign_path: result.campaign_path,
                    attempted: result.attempted,
                    succeeded: result.succeeded,
                    failures: result.failures,
                    watchdogs: result.watchdogs,
                    ..Event::default()
                });
            }
            Some("error") => {
                let failure: NativeError = serde_json::from_str(line).map_err(|_| error())?;
                if failure.schema != "gomad.explore-event/v3"
                    || failure.r#type != "error"
                    || !matches!(
                        failure.classification.as_str(),
                        "runner_failure"
                            | "capacity"
                            | "unsupported_target"
                            | "cancelled"
                            | "overall_timeout"
                            | "semantic_coverage_failure"
                            | "invalid_input"
                    )
                {
                    return Err(error());
                }
                events.push(Event {
                    schema: failure.schema,
                    r#type: failure.r#type,
                    classification: failure.classification,
                    message: failure.message,
                    ..Event::default()
                });
            }
            _ => return Err(error()),
        }
    }
    Ok(events)
}

#[cfg(test)]
fn classify(observation: &Observation) -> Outcome {
    if !observation.exec_error.is_empty() {
        return classify_native(observation, &[]);
    }
    match parse_events(&observation.stdout) {
        Ok(events) => classify_native(observation, &events),
        Err(error) => outcome(
            Class::Infrastructure,
            format!("explore output: {}", normalize(&error)),
        ),
    }
}

// Artifact events passed here must come from authenticated native journal references.
pub fn classify_native(observation: &Observation, events: &[Event]) -> Outcome {
    let mut result = classify_observation(observation, events);
    if result.class != Class::Pass && result.reason.is_empty() {
        result.reason = "unclassified: no diagnostic".into();
    }
    result
}

fn classify_observation(observation: &Observation, events: &[Event]) -> Outcome {
    if !observation.exec_error.is_empty() {
        return outcome(
            Class::Infrastructure,
            format!("gomad subprocess: {}", normalize(&observation.exec_error)),
        );
    }
    if let Some(error) = events.iter().find(|event| event.r#type == "error") {
        return classify_error(error);
    }
    let Some(result) = events.iter().rev().find(|event| event.r#type == "result") else {
        return outcome(
            Class::Infrastructure,
            format!(
                "gomad exit {} without a result event: {}",
                observation.exit,
                first_line(&observation.stderr)
            ),
        );
    };
    let mut retained = None;
    for event in events.iter().filter(|event| event.r#type == "artifact") {
        if let Some(facts) = observation.artifacts.get(&event.path) {
            retained = Some((event.path.as_str(), facts));
            if event.classification != "success" {
                break;
            }
        }
    }
    let empty = ArtifactFacts::default();
    let (path, facts) = retained.unwrap_or(("", &empty));
    let mut output = classify_result(&observation.test, result, facts, !path.is_empty());
    output.artifact_path = path.into();
    output.module_sha256.clone_from(&facts.module_sha256);
    if !path.is_empty() && facts.read_error.is_empty() {
        output.subtests = parse_subtests(&observation.test, &facts.stdout);
    }
    output
}

fn classify_error(event: &Event) -> Outcome {
    let message = &event.message;
    if let Some(rest) = message.strip_prefix("backend ")
        && let Some((kind, detail)) = rest.split_once(": ")
    {
        let detail = normalize(detail.split('\n').next().unwrap_or_default());
        return match kind {
            "unsupported" => outcome(Class::RefusedImport, format!("execute: {detail}")),
            "capacity" => outcome(Class::CapacityExceeded, detail),
            "trap" => outcome(Class::GuestFailure, format!("trap: {detail}")),
            "divergence" => outcome(Class::ReplayDivergence, detail),
            _ => outcome(Class::Infrastructure, format!("backend {kind}: {detail}")),
        };
    }
    match event.classification.as_str() {
        "overall_timeout" => {
            return outcome(
                Class::Timeout,
                "overall-timeout: preparation plus execution exceeded explore --overall-timeout",
            );
        }
        "capacity" => {
            return outcome(
                Class::CapacityExceeded,
                format!("artifact or journal capacity: {}", normalize(message)),
            );
        }
        "unsupported_target" => {
            return outcome(
                Class::RefusedImport,
                format!("prepare: {}", normalize(message)),
            );
        }
        _ => {}
    }
    if message.contains("choice_trace_overflow") {
        return outcome(Class::CapacityExceeded, "choice trace overflow");
    }
    if let Some((_, output)) = message
        .split_once("WASM compiler command failed:")
        .or_else(|| message.split_once("compiler command failed:"))
    {
        let reason = compile_reason(output);
        return if reason.is_empty() {
            outcome(
                Class::Infrastructure,
                "compiler command failed without a diagnostic (provider compiler limit or host failure)",
            )
        } else {
            outcome(Class::BuildFailure, reason)
        };
    }
    outcome(
        Class::Infrastructure,
        format!(
            "{}: {}",
            event.classification,
            normalize(&first_line(message.as_bytes()))
        ),
    )
}

fn compile_reason(output: &str) -> String {
    for line in output.split('\n').map(str::trim) {
        // File locations use either file:line or file:line:column.
        if let Some((location, diagnostic)) = line.split_once(": ") {
            let mut parts = location.rsplit(':');
            if !parts
                .next()
                .is_some_and(|part| !part.is_empty() && part.bytes().all(|b| b.is_ascii_digit()))
            {
                continue;
            }
            let mut file = parts.next().unwrap_or_default();
            if !file.is_empty() && file.bytes().all(|b| b.is_ascii_digit()) {
                file = parts.next().unwrap_or_default();
            }
            let base = file.rsplit('/').next().unwrap_or_default();
            if !base.chars().any(char::is_whitespace)
                && std::path::Path::new(base)
                    .extension()
                    .is_some_and(|extension| extension == "go" || extension == "s")
            {
                return normalize(&format!("{base}: {diagnostic}"));
            }
        }
    }
    output
        .split('\n')
        .map(str::trim)
        .find(|line| !line.is_empty() && !line.starts_with('#'))
        .map_or_else(String::new, normalize)
}

fn classify_result(
    test: &str,
    result: &Event,
    facts: &ArtifactFacts,
    have_artifact: bool,
) -> Outcome {
    match result.classification.as_str() {
        "success" => {
            if !have_artifact {
                return outcome(
                    Class::Infrastructure,
                    "success without a retained success artifact; verdict unverifiable",
                );
            }
            if !facts.read_error.is_empty() {
                return outcome(
                    Class::Infrastructure,
                    format!("open success artifact: {}", normalize(&facts.read_error)),
                );
            }
            if verdict_line(&facts.stdout, "PASS", test) {
                return outcome(Class::Pass, "");
            }
            if verdict_line(&facts.stdout, "SKIP", test) {
                return Outcome {
                    skipped: true,
                    ..outcome(Class::Pass, "")
                };
            }
            if String::from_utf8_lossy(&facts.stdout).contains("testing: warning: no tests to run")
            {
                return outcome(
                    Class::BuildFailure,
                    "guest-file-set: test not linked in the wasip1 guest (no tests to run)",
                );
            }
            outcome(
                Class::Infrastructure,
                if facts.stdout_truncated {
                    "verdict-unverifiable: stdout truncated before the test verdict"
                } else {
                    "verdict-unverifiable: exit 0 without a PASS or SKIP line for the test"
                },
            )
        }
        "watchdog_observation" => outcome(
            Class::Timeout,
            "execution-timeout: watchdog stopped the guest",
        ),
        "replay_divergence" => outcome(
            Class::ReplayDivergence,
            format!(
                "replay of the observed execution diverged{}",
                if facts.reason.is_empty() {
                    String::new()
                } else {
                    format!(": {}", facts.reason)
                }
            ),
        ),
        "target_failure" | "mixed_failure" => {
            if !have_artifact || !facts.read_error.is_empty() {
                return outcome(
                    Class::GuestFailure,
                    format!(
                        "target failure; artifact unreadable: {}",
                        normalize(&facts.read_error)
                    ),
                );
            }
            match facts.reason.as_str() {
                "world_capacity" => return outcome(Class::CapacityExceeded, "world capacity"),
                "world_replay_divergence" => {
                    return outcome(Class::ReplayDivergence, "world replay divergence");
                }
                _ => {}
            }
            if facts.kind == "gomad.watchdog-timeout/v1" || facts.termination == "timeout" {
                return outcome(
                    Class::Timeout,
                    "execution-timeout: watchdog stopped the guest",
                );
            }
            outcome(Class::GuestFailure, guest_failure_reason(test, facts))
        }
        _ => outcome(
            Class::Infrastructure,
            format!(
                "unknown explore classification {}",
                normalize(&result.classification)
            ),
        ),
    }
}

fn selected_test(selection: &str, observed: &str) -> bool {
    let mut selected = selection.split('/');
    let mut actual = observed.split('/');
    loop {
        match (selected.next(), actual.next()) {
            (None, None) => return true,
            (Some(part), Some(value)) if part == ".*" || part == value => {}
            _ => return false,
        }
    }
}

fn verdict_line(stdout: &[u8], verdict: &str, test: &str) -> bool {
    let prefix = format!("--- {verdict}: ");
    String::from_utf8_lossy(stdout).split('\n').any(|line| {
        line.trim()
            .strip_prefix(&prefix)
            .and_then(|name| name.split_once(" ("))
            .is_some_and(|(name, _)| selected_test(test, name))
    })
}

fn guest_failure_reason(test: &str, facts: &ArtifactFacts) -> String {
    let reason = if facts.reason.is_empty() || facts.reason == "nonzero_exit" {
        facts
            .exit_code
            .map_or_else(|| "nonzero_exit".into(), |code| format!("exit {code}"))
    } else {
        facts.reason.clone()
    };
    for stream in [&facts.stderr, &facts.stdout] {
        for line in String::from_utf8_lossy(stream).split('\n').map(str::trim) {
            if line.starts_with("panic: ") || line.starts_with("fatal error: ") {
                return normalize(&format!("{reason}: {line}"));
            }
        }
    }
    let stdout = String::from_utf8_lossy(&facts.stdout);
    for line in stdout.split('\n') {
        if let Some(detail) = line.trim_start_matches(ascii_space).strip_prefix("Error:")
            && detail.starts_with(ascii_space)
            && !detail.trim().is_empty()
        {
            return normalize(&format!("{reason}: {}", detail.trim()));
        }
    }
    let parent = format!("--- FAIL: {test} (");
    for line in stdout.split('\n').map(str::trim) {
        if line.starts_with("--- FAIL: ") && !line.starts_with(&parent) {
            return normalize(&format!("{reason}: {line}"));
        }
    }
    let line = first_line(&facts.stderr);
    normalize(&if line.is_empty() {
        reason
    } else {
        format!("{reason}: {line}")
    })
}

fn parse_subtests(test: &str, stdout: &[u8]) -> Option<Subtests> {
    let mut summary = Subtests::default();
    let depth = test.split('/').count();
    for line in String::from_utf8_lossy(stdout).split('\n') {
        if !line.starts_with(ascii_space) {
            continue;
        }
        let Some(verdict) = line.trim_start_matches(ascii_space).strip_prefix("--- ") else {
            continue;
        };
        let Some((verdict, rest)) = verdict.split_once(": ") else {
            continue;
        };
        let name = rest.split(ascii_space).next().unwrap_or_default();
        let parts: Vec<_> = name.split('/').collect();
        if parts.len() <= depth || !selected_test(test, &parts[..depth].join("/")) {
            continue;
        }
        match verdict {
            "PASS" => summary.pass += 1,
            "SKIP" => summary.skip += 1,
            "FAIL" => {
                summary.fail += 1;
                if summary.failed.len() < 20 {
                    summary.failed.push(name.into());
                }
            }
            _ => {}
        }
    }
    (summary.pass + summary.fail + summary.skip != 0).then_some(summary)
}

fn first_line(data: &[u8]) -> String {
    String::from_utf8_lossy(data)
        .split('\n')
        .map(str::trim)
        .find(|line| !line.is_empty())
        .unwrap_or_default()
        .into()
}

fn ascii_space(c: char) -> bool {
    matches!(c, ' ' | '\t' | '\n' | '\r' | '\x0c')
}
fn word(byte: u8) -> bool {
    byte.is_ascii_alphanumeric() || byte == b'_'
}
fn boundary(bytes: &[u8], index: usize) -> bool {
    index.checked_sub(1).is_some_and(|i| word(bytes[i]))
        != bytes.get(index).is_some_and(|b| word(*b))
}

fn replace_matches(
    message: &str,
    matcher: impl Fn(&str, usize) -> Option<(usize, String)>,
) -> String {
    let mut result = String::new();
    let mut index = 0;
    while index < message.len() {
        if let Some((end, replacement)) = matcher(message, index) {
            result.push_str(&replacement);
            index = end;
        } else {
            let c = message[index..].chars().next().unwrap();
            result.push(c);
            index += c.len_utf8();
        }
    }
    result
}

fn normalize_identities(message: &str) -> String {
    let message = replace_matches(message, |text, start| {
        let bytes = text.as_bytes();
        let end = start + 36;
        if end > bytes.len() || !boundary(bytes, start) || !boundary(bytes, end) {
            return None;
        }
        let uuid = &bytes[start..end];
        uuid.iter()
            .enumerate()
            .all(|(i, b)| {
                if [8, 13, 18, 23].contains(&i) {
                    *b == b'-'
                } else {
                    b.is_ascii_hexdigit()
                }
            })
            .then(|| (end, "<uuid>".into()))
    });
    let message = replace_matches(&message, |text, start| {
        let rest = text[start..].strip_prefix("0x")?;
        let length = rest.bytes().take_while(u8::is_ascii_hexdigit).count();
        (length > 0).then(|| (start + 2 + length, "0x?".into()))
    });
    replace_matches(&message, |text, start| {
        let prefixed = text[start..].starts_with("sha256:");
        let begin = start + if prefixed { 7 } else { 0 };
        if !boundary(text.as_bytes(), begin) {
            return None;
        }
        let length = text[begin..]
            .bytes()
            .take_while(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(b))
            .count();
        let end = begin + length;
        if length < 16 || !boundary(text.as_bytes(), end) {
            return None;
        }
        let value = &text[begin..end];
        Some((
            end,
            if prefixed || value.bytes().any(|b| (b'a'..=b'f').contains(&b)) {
                "<hex>".into()
            } else {
                text[start..end].into()
            },
        ))
    })
}

pub fn normalize(message: &str) -> String {
    let message = normalize_identities(message);
    let message = replace_matches(&message, |text, start| {
        let first = text[start..].chars().next()?;
        let (lead, mut index) = if start == 0 && first == '/' {
            ("", start)
        } else if ascii_space(first) || matches!(first, '"' | '\'' | '=' | ':' | '(' | '[') {
            (
                &text[start..start + first.len_utf8()],
                start + first.len_utf8(),
            )
        } else {
            return None;
        };
        let bytes = text.as_bytes();
        let mut base = None;
        while bytes.get(index) == Some(&b'/') {
            let begin = index + 1;
            let length = bytes[begin..]
                .iter()
                .take_while(|b| b.is_ascii_alphanumeric() || b"._@+~-".contains(b))
                .count();
            if length == 0 {
                break;
            }
            index = begin + length;
            base = Some(&text[begin..index]);
        }
        let base = base?;
        if bytes.get(index) == Some(&b'/') {
            index += 1;
        }
        Some((index, format!("{lead}<path>/{base}")))
    });
    let message = replace_matches(&message, |text, start| {
        if !boundary(text.as_bytes(), start) || !text.as_bytes()[start].is_ascii_digit() {
            return None;
        }
        let integer_end = start + text[start..].bytes().take_while(u8::is_ascii_digit).count();
        let mut end = integer_end;
        if text.as_bytes().get(end) == Some(&b'.')
            && text.as_bytes().get(end + 1).is_some_and(u8::is_ascii_digit)
        {
            end += 1;
            end += text[end..].bytes().take_while(u8::is_ascii_digit).count();
        }
        for suffix in [
            "ns", "µs", "us", "ms", "s", "m", "h", "B", "KiB", "MiB", "GiB", "",
        ] {
            if text[end..].starts_with(suffix) && boundary(text.as_bytes(), end + suffix.len()) {
                return Some((end + suffix.len(), "N".into()));
            }
        }
        if end != integer_end && boundary(text.as_bytes(), integer_end) {
            return Some((integer_end, "N".into()));
        }
        None
    });
    let mut result = String::new();
    let mut whitespace = false;
    for c in message.chars() {
        if ascii_space(c) {
            whitespace = true;
        } else {
            if whitespace && !result.is_empty() {
                result.push(' ');
            }
            result.push(c);
            whitespace = false;
        }
    }
    let result = result.trim();
    if result.chars().count() > 240 {
        format!("{}…", result.chars().take(240).collect::<String>())
    } else {
        result.into()
    }
}
