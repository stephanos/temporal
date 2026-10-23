use super::Configuration;
use crate::{
    campaign::{exploration::base64, scenario, soft_assertions},
    evidence::canonical,
};
use gomad_wasmhost::{linked, linked_cluster as helper};
use serde::{Deserialize, Serialize};
#[cfg(feature = "determinism-fixture")]
use serde_json::Value;
use serde_json::json;
use std::collections::{BTreeMap, BTreeSet};

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Plan {
    schema: String,
    #[serde(default)]
    prefix: String,
    maximum_bytes: u64,
    #[serde(default)]
    histories: Vec<HistoryComparison>,
    #[serde(default)]
    convergence: Vec<Convergence>,
    #[serde(default)]
    soft_assertions: Vec<Policy>,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Policy {
    handle: helper::Handle,
    report_only: bool,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct HistoryComparison {
    name: String,
    expected: helper::Handle,
    actual: helper::Handle,
    expected_events: Vec<String>,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Convergence {
    name: String,
    handles: Vec<helper::Handle>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct History {
    cluster_id: String,
    #[serde(rename = "history")]
    encoded: String,
    event_ids: Vec<i64>,
    types: Vec<String>,
}

#[derive(Serialize)]
pub struct Evidence {
    label: String,
    value: String,
    full_sha256: String,
}

#[derive(Serialize)]
struct Assessment {
    passed: bool,
    inconclusive: bool,
}

#[derive(Serialize)]
pub struct Oracle {
    name: String,
    #[serde(flatten)]
    assessment: Assessment,
    soft_assertion: bool,
    report_only: bool,
    evidence: Vec<Evidence>,
    failure_identity: String,
    identity: String,
}

#[derive(Serialize)]
pub struct Report {
    schema: &'static str,
    pub outcome: &'static str,
    reason: &'static str,
    failure_identity: String,
    pub oracles: Vec<Oracle>,
}

fn bound(configuration: &Configuration) -> u64 {
    configuration
        .oracle_plan
        .as_ref()
        .map_or(64 << 20, |plan| plan.maximum_bytes)
}

fn policy(configuration: &Configuration, handle: &helper::Handle) -> bool {
    configuration
        .oracle_plan
        .as_ref()
        .and_then(|plan| {
            plan.soft_assertions
                .iter()
                .find(|policy| policy.handle == *handle)
        })
        .is_some_and(|policy| policy.report_only)
}

pub(crate) fn validate(configuration: &Configuration) -> Result<(), String> {
    let known: BTreeMap<_, _> = configuration
        .incarnations
        .iter()
        .map(|item| (&item.handle, item))
        .collect();
    if known
        .values()
        .any(|item| !matches!(item.workload_kind.as_str(), "" | "go-run" | "go-test"))
    {
        return Err("invalid recorded cluster workload kind".into());
    }
    let Some(plan) = &configuration.oracle_plan else {
        return Ok(());
    };
    if plan.maximum_bytes == 0
        || plan.maximum_bytes > 64 << 20
        || plan
            .histories
            .len()
            .saturating_mul(3)
            .saturating_add(plan.convergence.len())
            .saturating_add(plan.soft_assertions.len())
            > 4096
    {
        return Err("invalid cluster oracle evidence bound".into());
    }
    let mut policies = BTreeSet::new();
    for policy in &plan.soft_assertions {
        if !policies.insert(&policy.handle)
            || known
                .get(&policy.handle)
                .is_none_or(|item| item.workload_kind != "go-test")
        {
            return Err("soft assertion policy requires a recorded Go-test workload".into());
        }
    }
    match plan.schema.as_str() {
        "gomad.wasm.soft-assertion-observation/v1"
            if !policies.is_empty()
                && plan.prefix.is_empty()
                && plan.histories.is_empty()
                && plan.convergence.is_empty() => {}
        "gomad.wasm.history-observation/v1"
            if !plan.prefix.is_empty()
                && plan.prefix.len() <= 256
                && !plan.prefix.contains(['\r', '\n'])
                && plan.maximum_bytes <= 1 << 20
                && (!plan.histories.is_empty() || !plan.convergence.is_empty()) =>
        {
            let mut names = BTreeSet::new();
            for history in &plan.histories {
                if !known.contains_key(&history.expected)
                    || !known.contains_key(&history.actual)
                    || history.expected == history.actual
                    || history.expected_events.is_empty()
                {
                    return Err("invalid bound cluster history oracle intent".into());
                }
                for suffix in [".history", ".events", ".identity"] {
                    if !names.insert(format!("{}{suffix}", history.name)) {
                        return Err("duplicate cluster oracle name".into());
                    }
                }
            }
            for convergence in &plan.convergence {
                let handles: BTreeSet<_> = convergence.handles.iter().collect();
                if handles.len() < 2
                    || handles.len() != convergence.handles.len()
                    || handles.iter().any(|handle| !known.contains_key(handle))
                    || !names.insert(convergence.name.clone())
                {
                    return Err("invalid independent convergence oracle intent".into());
                }
            }
        }
        _ => return Err("invalid typed cluster oracle plan".into()),
    }
    Ok(())
}

fn oracle(
    name: String,
    passed: bool,
    evidence: Vec<(&str, Vec<u8>)>,
    maximum: u64,
) -> Result<Oracle, String> {
    if evidence
        .iter()
        .map(|(_, value)| value.len() as u64)
        .sum::<u64>()
        > maximum
    {
        return Err("capacity: resource=oracle_evidence_bytes".into());
    }
    let evidence: Vec<_> = evidence
        .into_iter()
        .map(|(label, value)| Evidence {
            label: label.into(),
            full_sha256: canonical::digest(&value),
            value: base64(&value),
        })
        .collect();
    let mut result = Oracle {
        name,
        assessment: Assessment {
            passed,
            inconclusive: false,
        },
        soft_assertion: false,
        report_only: false,
        evidence,
        failure_identity: String::new(),
        identity: String::new(),
    };
    seal(&mut result)?;
    Ok(result)
}

fn seal(result: &mut Oracle) -> Result<(), String> {
    result.identity.clear();
    result.failure_identity.clear();
    if !result.assessment.passed {
        result.failure_identity = canonical::digest(&canonical::encode(&json!({
            "name":result.name,"evidence":result.evidence,
        }))?);
    }
    result.identity = canonical::digest(&canonical::encode(result)?);
    Ok(())
}

fn history(
    execution: &helper::Execution,
    handle: &helper::Handle,
    plan: &Plan,
) -> Result<History, String> {
    let item = execution
        .incarnations
        .iter()
        .find(|item| item.handle == *handle)
        .ok_or("cluster oracle guest did not complete")?;
    let stdout = item
        .execution
        .evidence
        .get("stdout")
        .map_or(&[][..], Vec::as_slice);
    let matches: Vec<_> = stdout
        .split(|byte| *byte == b'\n')
        .filter_map(|line| line.strip_prefix(plan.prefix.as_bytes()))
        .collect();
    if matches.len() != 1 || matches[0].len() as u64 > plan.maximum_bytes {
        return Err("missing, ambiguous or oversized WASM history observation".into());
    }
    let value: History = serde_json::from_slice(matches[0]).map_err(|error| error.to_string())?;
    if value.cluster_id.is_empty()
        || scenario::canonical_base64(&value.encoded)?.is_empty()
        || value.event_ids.is_empty()
        || value.event_ids.len() != value.types.len()
        || value.event_ids.iter().any(|id| *id <= 0)
        || value.types.iter().any(String::is_empty)
    {
        return Err("incomplete WASM history observation".into());
    }
    Ok(value)
}

fn histories(
    execution: &helper::Execution,
    plan: &Plan,
    output: &mut Vec<Oracle>,
) -> Result<(), String> {
    let operation = |value: &History| {
        json!([{"id":"committed","actor":"workflow","kind":"workflow-history",
        "invocation":1,"completion":2,"output":value.encoded}])
    };
    for comparison in &plan.histories {
        let expected = history(execution, &comparison.expected, plan)?;
        let actual = history(execution, &comparison.actual, plan)?;
        output.push(oracle(
            format!("{}.history", comparison.name),
            expected.encoded == actual.encoded,
            vec![
                ("actual", canonical::encode(&operation(&actual))?),
                ("expected", canonical::encode(&operation(&expected))?),
            ],
            plan.maximum_bytes,
        )?);
        let mut ids: Vec<_> = actual.event_ids.iter().map(ToString::to_string).collect();
        let mut expected_ids = comparison.expected_events.clone();
        ids.sort();
        expected_ids.sort();
        output.push(oracle(
            format!("{}.events", comparison.name),
            ids == expected_ids,
            vec![
                ("actual", canonical::encode(&ids)?),
                ("expected", canonical::encode(&expected_ids)?),
            ],
            plan.maximum_bytes,
        )?);
        output.push(oracle(
            format!("{}.identity", comparison.name),
            actual.cluster_id == expected.cluster_id,
            vec![
                ("actual", actual.cluster_id.into_bytes()),
                ("expected", expected.cluster_id.into_bytes()),
            ],
            plan.maximum_bytes,
        )?);
    }
    for convergence in &plan.convergence {
        let mut values = BTreeMap::new();
        for handle in &convergence.handles {
            values.insert(
                format!("{}-{}", handle.node, handle.incarnation),
                scenario::canonical_base64(&history(execution, handle, plan)?.encoded)?,
            );
        }
        let passed = values
            .values()
            .all(|value| Some(value) == values.values().next());
        output.push(oracle(
            convergence.name.clone(),
            passed,
            values
                .iter()
                .map(|(name, value)| (name.as_str(), value.clone()))
                .collect(),
            plan.maximum_bytes,
        )?);
    }
    Ok(())
}

pub(crate) fn assess(
    execution: &helper::Execution,
    configuration: &Configuration,
) -> Result<Report, String> {
    validate(configuration)?;
    let mut oracles = Vec::new();
    if let Some(plan) = &configuration.oracle_plan
        && plan.schema == "gomad.wasm.history-observation/v1"
    {
        histories(execution, plan, &mut oracles)?;
    }
    for incarnation in &configuration.incarnations {
        if incarnation.workload_kind != "go-test" {
            continue;
        }
        let Some(item) = execution
            .incarnations
            .iter()
            .find(|item| item.handle == incarnation.handle)
        else {
            continue;
        };
        if let Some(report) = soft_assertions::observe(
            item.execution
                .evidence
                .get("stdout")
                .map_or(&[], Vec::as_slice),
            item.execution
                .evidence
                .get("stderr")
                .map_or(&[], Vec::as_slice),
            scenario::complete_output(&item.execution),
            policy(configuration, &item.handle),
        ) {
            let instance = format!("{}/{}", item.handle.node, item.handle.incarnation);
            let mut result = oracle(
                format!(
                    "soft-assertion.instance-{}",
                    canonical::digest(instance.as_bytes())
                ),
                !report.inconclusive && (report.messages.is_empty() || report.report_only),
                vec![
                    ("instance", instance.into_bytes()),
                    ("messages", canonical::encode(&report.messages)?),
                    (
                        "status",
                        if report.inconclusive {
                            b"inconclusive".to_vec()
                        } else {
                            b"observed".to_vec()
                        },
                    ),
                ],
                bound(configuration),
            )?;
            result.soft_assertion = true;
            result.report_only = report.report_only;
            result.assessment.inconclusive = report.inconclusive;
            seal(&mut result)?;
            oracles.push(result);
        }
    }
    if oracles.len() > 4096 {
        return Err("capacity: scenario oracle results".into());
    }
    let failed: Vec<_> = oracles
        .iter()
        .filter(|oracle| !oracle.assessment.passed)
        .collect();
    let failure_identity = if failed.is_empty() {
        String::new()
    } else {
        canonical::digest(&canonical::encode(&failed)?)
    };
    Ok(Report {
        schema: "gomad.cluster-oracles/v1",
        outcome: if failed.is_empty() {
            "completed"
        } else {
            "oracle_failed"
        },
        reason: if failed.iter().any(|oracle| oracle.soft_assertion) {
            "soft_assertion"
        } else if failed.is_empty() {
            ""
        } else {
            "oracle_failed"
        },
        failure_identity,
        oracles,
    })
}

pub(crate) fn apply(observed: &mut linked::Execution, report: &Report) -> Result<(), String> {
    if report.oracles.is_empty() {
        return Ok(());
    }
    if report.outcome == "oracle_failed"
        && observed.outcome.termination == "exit"
        && observed.outcome.exit_code == Some(0)
    {
        observed.outcome.termination = "oracle_failed";
        observed.outcome.exit_code = None;
        observed.outcome.message = report.reason.into();
    }
    observed
        .evidence
        .insert("cluster_oracles".into(), canonical::encode(report)?);
    Ok(())
}

#[cfg(feature = "determinism-fixture")]
pub(crate) fn capacity_observation(
    execution: &helper::Execution,
    configuration: &Configuration,
) -> Value {
    let assessment = assess(execution, configuration);
    let mut value = match &assessment {
        Ok(report) => json!({"oracles":report.oracles,"oracle_error":""}),
        Err(error) => json!({"oracles":[],"oracle_error":error}),
    };
    if let Some(item) = execution.incarnations.iter().find(|item| {
        item.execution.outcome.termination == "capacity"
            && item.execution.outcome.message == "output capacity"
    }) {
        let test_workload = configuration.incarnations.iter().any(|incarnation| {
            incarnation.handle == item.handle && incarnation.workload_kind == "go-test"
        });
        let report = test_workload
            .then(|| {
                soft_assertions::observe(
                    item.execution
                        .evidence
                        .get("stdout")
                        .map_or(&[], Vec::as_slice),
                    item.execution
                        .evidence
                        .get("stderr")
                        .map_or(&[], Vec::as_slice),
                    false,
                    policy(configuration, &item.handle),
                )
            })
            .flatten();
        value["capacity_failure"] = json!({"termination":"capacity","message":"output capacity", "soft_assertions":report,
            "detail":assessment.err().unwrap_or_else(|| "choice trace is not complete".into())});
    }
    value
}

#[cfg(test)]
mod replay_tests {
    use super::*;
    use crate::evidence::{
        identity::Identity,
        record::{Limits, Outcome, Record, Stream},
    };
    use serde_json::Value;

    fn history_report() -> Report {
        Report {
            schema: "gomad.cluster-oracles/v1",
            outcome: "completed",
            reason: "",
            failure_identity: String::new(),
            oracles: vec![Oracle {
                name: "history".into(),
                assessment: Assessment {
                    passed: true,
                    inconclusive: false,
                },
                soft_assertion: false,
                report_only: false,
                evidence: vec![Evidence {
                    label: "history".into(),
                    value: "committed".into(),
                    full_sha256: canonical::digest(b"committed"),
                }],
                failure_identity: String::new(),
                identity: canonical::digest(b"history"),
            }],
        }
    }

    #[test]
    fn replay_compares_actual_canonical_cluster_oracle_report() {
        let mut outcome = linked::Outcome::new("exit", "");
        outcome.exit_code = Some(0);
        let mut observed = linked::Execution {
            terminal_monotonic_nanos: None,
            admission_terminal_to_return_nanos: None,
            timing: None,
            #[cfg(feature = "helper-guest-profile")]
            guest_profile_error: None,
            #[cfg(feature = "helper-diagnostic-profile")]
            external_owner_attempts: 0,
            authorization_to_terminal_nanos: None,
            outcome,
            evidence: BTreeMap::new(),
            event_order: None,
        };
        apply(&mut observed, &history_report()).unwrap();
        let identity = Identity {
            target: canonical::digest(b"target"),
            toolchain: canonical::digest(b"toolchain"),
            platform: canonical::digest(b"platform"),
            implementation: canonical::digest(b"implementation"),
            configuration: canonical::digest(b"configuration"),
        };
        let raw = &observed.evidence["cluster_oracles"];
        assert_eq!(
            canonical::digest(raw),
            "04c76f5cc0aa430c09351ecb29f0c3ca1042ec23d6ec259e4ae09246d46b578f"
        );
        let retained_report = raw.clone();
        let mut record = Record {
            schema: "gomad.execution-record/v2".into(),
            execution_identity: identity.digest().unwrap(),
            identity,
            profile: "cooperative".into(),
            seed: 7,
            arguments: vec![],
            environment: BTreeMap::new(),
            limits: Limits {
                fuel: 1,
                memory_bytes: 1,
                wall_millis: 1,
                output_bytes: 1,
                evidence_bytes: 4096,
            },
            virtual_time_nanos: 0,
            outcome: Outcome {
                class: "success".into(),
                termination: "exit".into(),
                exit_code: Some(0),
                reason: String::new(),
            },
            streams: BTreeMap::from([(
                "evidence/cluster_oracles".into(),
                Stream {
                    bytes: raw.len() as u64,
                    sha256: canonical::digest(raw),
                    records: 0,
                },
            )]),
        };
        record.bytes().unwrap();
        assert_eq!(crate::replay::divergence::first(&record, &observed), None);
        let mut changed: Value = serde_json::from_slice(raw).unwrap();
        changed["oracles"][0]["evidence"][0]["value"] = json!("different history");
        let changed = canonical::encode(&changed).unwrap();
        observed.evidence.insert("cluster_oracles".into(), changed);
        assert_eq!(
            crate::replay::divergence::first(&record, &observed).as_deref(),
            Some("backend.evidence.sha256")
        );
        observed.evidence.remove("cluster_oracles");
        assert_eq!(
            crate::replay::divergence::first(&record, &observed).as_deref(),
            Some("backend.evidence.sha256")
        );
        record.streams.clear();
        assert_eq!(crate::replay::divergence::first(&record, &observed), None);
        observed
            .evidence
            .insert("cluster_oracles".into(), retained_report);
        assert_eq!(
            crate::replay::divergence::first(&record, &observed).as_deref(),
            Some("backend.evidence.sha256")
        );
    }
}
