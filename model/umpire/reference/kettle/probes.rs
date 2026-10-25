//! Descriptor-driven request mutations with catalog predictions and actual-wire classification.

use crate::{
    KettleServiceMethod as Method, api,
    config::{self, Limits},
    kettle_request_fields, mutate_kettle_request,
    wire::umpire::reference::kettle::v1 as k,
};
use prost::Message;
use std::{collections::BTreeSet, fmt::Write};
use umpire_catalog::{
    Expectation, GapReport, MutationApplicability, ProbeResult, Violation, WireCase,
    mutation_entries, wire_cases,
};
use umpire_runtime::annotations::{ProbeCase, ProbeExpectation, ProbeObservation, ProbeResponse};

type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;

struct Entry {
    method: Method,
    prerequisite: &'static str,
    case: WireCase,
}

impl Entry {
    fn id(&self) -> String {
        format!("generic::{}::{}", self.method.name(), self.case.id())
    }

    fn prepare(&self, baseline: bool) -> Result<(ProbeCase, Vec<u8>, Vec<Violation>)> {
        let (state, refusal) = match &self.case.applicability {
            MutationApplicability::Applicable => (0, String::new()),
            MutationApplicability::NotApplicable(reason) => (1, reason.clone()),
            MutationApplicability::Refused(reason) => (2, reason.clone()),
            MutationApplicability::Unchanged => {
                (3, "mutation leaves the baseline unchanged".into())
            }
        };
        let mut metadata = ProbeCase {
            id: self.id(),
            method: method(self.method),
            prerequisite: self.prerequisite.into(),
            generic: true,
            applicability: state,
            refusal,
            ..Default::default()
        };
        if state != 0 {
            return Ok((metadata, Vec::new(), Vec::new()));
        }
        let original = baseline_bytes(self.method);
        if !api::request_findings(self.method, &original, "room")?.is_empty() {
            return Err("invalid Kettle probe baseline".into());
        }
        if baseline {
            return Ok((metadata, original, Vec::new()));
        }
        let Ok(bytes) = mutate_kettle_request(
            self.method,
            &original,
            &self.case.field.path,
            &self.case.mutation,
        ) else {
            metadata.applicability = 2;
            metadata.refusal = "typed mutation construction refused".into();
            return Ok((metadata, Vec::new(), Vec::new()));
        };
        if bytes == original {
            metadata.applicability = 3;
            metadata.refusal = "mutation leaves the baseline unchanged".into();
            return Ok((metadata, Vec::new(), Vec::new()));
        }
        let mut findings = api::request_findings(self.method, &bytes, "room")?;
        findings.sort_by_key(|v| (v.path.clone(), v.constraint.clone(), v.fields.clone()));
        findings.dedup();
        findings.retain(|v| {
            v.expected.as_ref().is_some_and(|e| !matches!(e.expectation, Expectation::Unobservable))
        });
        if findings.is_empty()
            && self.prerequisite != "empty"
            && (self.case.field.path.ends_with(".kettle_id")
                || self.case.field.path.ends_with(".enabled")
                || self.case.field.path.ends_with(".action"))
        {
            metadata.applicability = 2;
            metadata.refusal = "mutation changes a stateful prerequisite".into();
            return Ok((metadata, Vec::new(), Vec::new()));
        }
        metadata.predictions = findings
            .iter()
            .enumerate()
            .map(|(index, finding)| {
                let mut expected = ProbeExpectation {
                    id: format!(
                        "{}::prediction:{index}:{}:{}",
                        self.id(),
                        finding.path,
                        finding.constraint
                    ),
                    ..Default::default()
                };
                let Some(declared) = &finding.expected else {
                    unreachable!("filtered expectation")
                };
                match &declared.expectation {
                    Expectation::Accepts => expected.accepts = true,
                    Expectation::RejectsCode(code) => expected.code = *code as i32,
                    Expectation::RejectsAnyStatus => {}
                    Expectation::Rejects { code, message_pattern, details_type } => {
                        expected.code = *code;
                        expected.message_pattern.clone_from(message_pattern);
                        expected.details_type = details_type.clone().unwrap_or_default();
                    }
                    Expectation::Unobservable => unreachable!(),
                }
                expected
            })
            .collect();
        Ok((metadata, bytes, findings))
    }

    fn classify(&self, observation: &ProbeObservation) -> Result<GapReport> {
        let (canonical, _, findings) = self.prepare(false)?;
        if observation.applicability != canonical.applicability
            || observation.refusal != canonical.refusal
            || observation.prediction_count as usize != canonical.predictions.len()
            || observation.predictions.len() != canonical.predictions.len()
            || observation.message_matches
            || !observation.details_type.is_empty()
        {
            return Err("observation diverges from prepared Kettle probe".into());
        }
        let mut case = self.case.clone();
        if canonical.applicability != 0 {
            if observation.accepted || observation.code != 0 {
                return Err("local mutation has a fabricated server response".into());
            }
            case.applicability = match canonical.applicability {
                1 => MutationApplicability::NotApplicable(canonical.refusal),
                2 => MutationApplicability::Refused(canonical.refusal),
                3 => MutationApplicability::Unchanged,
                _ => return Err("unknown applicability".into()),
            };
            return Ok(GapReport {
                entries: mutation_entries(&case, &[], ProbeResult::Unverifiable)?,
            });
        }
        if observation.accepted && observation.code != 0
            || !observation.accepted && !matches!(observation.code, 3 | 5..=11 | 16)
        {
            return Err("invalid or inconclusive Kettle server response".into());
        }
        let result = |matched, details| {
            if observation.accepted {
                ProbeResult::Accepted
            } else {
                ProbeResult::Rejected {
                    code: observation.code,
                    message_matches: matched,
                    details_type: details,
                }
            }
        };
        let mut indices = BTreeSet::new();
        let mut predictions = Vec::new();
        for observed in &observation.predictions {
            let index = observed.index as usize;
            if !indices.insert(index)
                || canonical.predictions.get(index).is_none_or(|v| v.id != observed.prediction_id)
                || observed.details_type.len() > 512
            {
                return Err("invalid Kettle prediction correlation".into());
            }
            predictions.push((
                findings[index].clone(),
                result(
                    observed.message_matches,
                    (!observed.details_type.is_empty()).then(|| observed.details_type.clone()),
                ),
            ));
        }
        Ok(GapReport { entries: mutation_entries(&case, &predictions, result(false, None))? })
    }
}

fn entries() -> Vec<Entry> {
    let mut entries = Vec::new();
    let limits = Limits::default();
    let bounds = [
        Limits::for_kettle("compact").get(&config::MAX_TAGS),
        limits.get(&config::MAX_TAGS),
        limits.get(&config::MAX_KETTLE_ID_BYTES),
        60,
    ];
    for (method, prerequisite) in [
        (Method::Fill, "empty"),
        (Method::Heat, "filled"),
        (Method::Pour, "filled"),
        (Method::Describe, "filled"),
        (Method::PollHeating, "heating"),
        (Method::RespondHeating, "heating"),
    ] {
        for case in wire_cases(&kettle_request_fields(method), &bounds) {
            entries.push(Entry { method, prerequisite, case });
        }
    }
    entries.sort_by_key(Entry::id);
    entries
}

fn method(method: Method) -> String {
    format!("/umpire.reference.kettle.v1.KettleService/{}", method.name())
}

fn baseline_bytes(method: Method) -> Vec<u8> {
    let id = "room".to_string();
    match method {
        Method::Fill => k::FillRequest {
            kettle_id: id,
            idempotency_key: "12345678-1234-4234-8234-123456789abc".into(),
            options: Some(k::fill_request::FillOptions {
                water_kind: 1,
                source: Some(k::fill_request::fill_options::Source::Tap("tap".into())),
                labels: [("origin".into(), "local".into())].into_iter().collect(),
                tags: vec!["clean".into()],
                timeout: Some(prost_types::Duration { seconds: 1, nanos: 0 }),
                ..Default::default()
            }),
        }
        .encode_to_vec(),
        Method::Heat => k::HeatRequest { kettle_id: id, enabled: true }.encode_to_vec(),
        Method::Pour => k::PourRequest { kettle_id: id }.encode_to_vec(),
        Method::Describe => k::DescribeRequest { kettle_id: id }.encode_to_vec(),
        Method::PollHeating => k::PollHeatingRequest { kettle_id: id }.encode_to_vec(),
        Method::RespondHeating => {
            k::RespondHeatingRequest { kettle_id: id, action: 1 }.encode_to_vec()
        }
        Method::ListKettles | Method::ResetKettle => unreachable!("out-of-scope probe method"),
    }
}

/// Prepare descriptor mutations or classify the complete set of actual server observations.
pub fn run(request: &k::ProbeRequest) -> Result<ProbeResponse> {
    if !request.kettle_id.is_empty() && request.kettle_id != "room" {
        return Err("Kettle probe context must use the canonical room kettle".into());
    }
    let entries = entries();
    match request.operation {
        0 => Ok(ProbeResponse {
            cases: entries.iter().map(|v| v.prepare(false).map(|v| v.0)).collect::<Result<_>>()?,
            ..Default::default()
        }),
        1 => {
            let entry =
                entries.iter().find(|e| e.id() == request.case_id).ok_or("unknown Kettle probe")?;
            let (case, message, _) = entry.prepare(request.baseline)?;
            Ok(ProbeResponse {
                method: case.method.clone(),
                cases: vec![case],
                message,
                ..Default::default()
            })
        }
        2 => {
            let actual: BTreeSet<_> =
                request.observations.iter().map(|v| v.case_id.as_str()).collect();
            let expected: BTreeSet<_> = entries.iter().map(Entry::id).collect();
            if actual.len() != request.observations.len()
                || actual.len() != expected.len()
                || !expected.iter().all(|id| actual.contains(id.as_str()))
            {
                return Err("incomplete, duplicate or unknown Kettle probe observations".into());
            }
            let mut response = ProbeResponse::default();
            for entry in entries {
                let observed = request
                    .observations
                    .iter()
                    .find(|v| v.case_id == entry.id())
                    .ok_or("missing Kettle observation")?;
                let report = entry.classify(observed)?;
                response.fails |= report.fails();
                writeln!(response.report, "Generic case: {}", entry.id())?;
                write!(response.report, "{report}")?;
            }
            Ok(response)
        }
        3 => Ok(ProbeResponse {
            report: api::catalog::coverage(&Limits::default())?,
            ..Default::default()
        }),
        _ => Err("unknown Kettle probe operation".into()),
    }
}
