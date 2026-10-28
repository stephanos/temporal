//! Applicability and observer coverage linked to exact retained producer events.
use super::{AuthorityRef, CountEvidence, MetricEvidence, NoWorkReason, WorkCounts};
use serde::{Deserialize, Serialize};
use std::{
    collections::{BTreeMap, BTreeSet},
    fs, io,
    path::Path,
};
use umpire_backend_process::bytes_sha256;

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
/// Exact retained event manifest backing the evidence in one stage receipt.
pub struct CountAuthority {
    /// Invocation-local identity referenced by modality evidence.
    pub id: String,
    /// SHA-256 of the exact retained authority bytes.
    pub sha256: String,
    /// Canonical manifest of event names and their content digests.
    pub records: String,
}

/// Counts and their applicability derived from the same producer event set.
pub struct ObservedWork {
    /// Completed numeric observations, without expected-count substitution.
    pub counts: WorkCounts,
    /// Provenance and applicability of each modality.
    pub evidence: CountEvidence,
    /// Exact event manifest backing this stage's evidence.
    pub authority: CountAuthority,
}

/// Collect current-stage evidence without treating missing observers as zero.
pub fn collect_evidenced(directory: &Path) -> io::Result<ObservedWork> {
    let paths = super::event_paths(directory)?;
    let (mut counts, tlc, trace) = super::collect_paths(&paths)?;
    (counts.tlc_launches, counts.tlc_configurations) = tlc.finish(directory)?;
    #[derive(Serialize)]
    struct Document {
        version: u8,
        scope: String,
        events: BTreeMap<String, String>,
    }
    let mut events = BTreeMap::new();
    for path in paths {
        let name = path
            .file_name()
            .and_then(|name| name.to_str())
            .ok_or_else(|| invalid("invalid count authority event name"))?
            .to_owned();
        events.insert(name, bytes_sha256(&fs::read(path)?));
    }
    let records = serde_json::to_string(&Document {
        version: 1,
        scope: super::encode_path(directory),
        events,
    })?;
    let authority = CountAuthority {
        id: "event-set-v1".into(),
        sha256: bytes_sha256(records.as_bytes()),
        records,
    };
    let reference = AuthorityRef { id: authority.id.clone(), sha256: authority.sha256.clone() };
    let evidence = trace.evidence(&mut counts, &reference)?;
    evidence.validate_counts(&counts)?;
    Ok(ObservedWork { counts, evidence, authority })
}

#[derive(Clone, Copy, Eq, PartialEq)]
enum Requirement {
    Observed,
    NotApplicable(NoWorkReason),
}

fn requirement(value: &str, tests: bool) -> io::Result<Requirement> {
    if value == if tests { "observed-tests" } else { "observed-tlc" } {
        return Ok(Requirement::Observed);
    }
    let reason = serde_json::from_value(serde_json::Value::String(value.into()))
        .map_err(|_| invalid("unknown native applicability contract"))?;
    Ok(Requirement::NotApplicable(reason))
}

fn invalid(message: &str) -> io::Error {
    io::Error::other(message)
}

fn digest(value: &str) -> bool {
    value.len() == 64 && value.bytes().all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
}

fn dispatch_id(value: &str) -> bool {
    value.strip_prefix("dispatch-").and_then(|value| value.split_once('-')).is_some_and(
        |(pid, counter)| {
            !pid.is_empty()
                && !counter.is_empty()
                && pid.bytes().all(|b| b.is_ascii_digit())
                && counter.bytes().all(|b| b.is_ascii_digit())
                && pid.parse::<u32>().is_ok()
                && counter.parse::<u64>().is_ok()
        },
    )
}

#[derive(Default)]
pub(super) struct Trace {
    dispatches: BTreeMap<String, (Requirement, Requirement)>,
    completions: BTreeMap<String, i32>,
    coverage: BTreeMap<String, String>,
    coverage_finished: BTreeMap<String, String>,
    tests: BTreeSet<String>,
    tlc_attempted: bool,
}

impl Trace {
    pub(super) fn ingest(&mut self, fields: &[&str]) -> io::Result<bool> {
        let Some(kind) = fields.get(1) else { return Ok(false) };
        if *kind == "tlc-attempt" {
            self.tlc_attempted = true;
        }
        if !kind.starts_with("dispatch-") && !kind.starts_with("coverage-") {
            return Ok(false);
        }
        match fields {
            ["1", "dispatch-begin", id, tests, tlc] if dispatch_id(id) => {
                let contract = (requirement(tests, true)?, requirement(tlc, false)?);
                if self.dispatches.insert((*id).into(), contract).is_some() {
                    return Err(invalid("duplicate native dispatch"));
                }
            }
            ["1", "dispatch-finish", id, code] if dispatch_id(id) => {
                let code = code.parse::<i32>().map_err(|_| invalid("invalid dispatch status"))?;
                if self.completions.insert((*id).into(), code).is_some() {
                    return Err(invalid("duplicate native completion"));
                }
            }
            ["1", "coverage-begin", id, capability, "1", image, inputs, command]
                if super::valid_id(id)
                    && matches!(*capability, "rust_tlc_observed_v1" | "go_checker_free_v1")
                    && [image, inputs, command].into_iter().all(|value| digest(value)) =>
            {
                let payload = fields[2..].join("\t");
                if self.coverage.insert((*id).into(), bytes_sha256(payload.as_bytes())).is_some() {
                    return Err(invalid("duplicate observer coverage"));
                }
            }
            ["1", "coverage-finish", id, sha] if super::valid_id(id) && digest(sha) => {
                if self.coverage_finished.insert((*id).into(), (*sha).into()).is_some() {
                    return Err(invalid("duplicate observer completion"));
                }
            }
            _ => return Err(invalid("invalid native contract or observer coverage")),
        }
        Ok(true)
    }

    pub(super) fn validate(&mut self, tests: &BTreeSet<String>) -> io::Result<()> {
        if !self.dispatches.keys().eq(self.completions.keys()) {
            return Err(invalid("incomplete native dispatch authority"));
        }
        if self.coverage != self.coverage_finished
            || self.coverage.keys().any(|id| !tests.contains(id))
        {
            return Err(invalid("crossed or incomplete observer authority"));
        }
        self.tests = tests.clone();
        Ok(())
    }

    fn exclusion(&self, tests: bool) -> io::Result<Option<NoWorkReason>> {
        if self.dispatches.is_empty() || self.completions.values().any(|code| *code != 0) {
            return Ok(None);
        }
        let mut reason = None;
        for contract in self.dispatches.values() {
            match if tests { contract.0 } else { contract.1 } {
                Requirement::Observed => return Ok(None),
                Requirement::NotApplicable(current) => {
                    if reason.is_some_and(|prior| prior != current) {
                        return Err(invalid("mixed native applicability reasons"));
                    }
                    reason = Some(current);
                }
            }
        }
        Ok(reason)
    }

    fn evidence(
        &self,
        counts: &mut WorkCounts,
        authority: &AuthorityRef,
    ) -> io::Result<CountEvidence> {
        let unknown = || MetricEvidence::Unknown {
            reason: "no complete producer or native exclusion authority".into(),
        };
        let observed = || MetricEvidence::Observed { authority: authority.clone() };
        let excluded =
            |reason| MetricEvidence::NotApplicable { authority: authority.clone(), reason };
        let tests = if let Some(reason) = self.exclusion(true)? {
            if counts.discovered.is_some() {
                return Err(invalid("native test exclusion contradicts observations"));
            }
            excluded(reason)
        } else if counts.discovered.is_some() {
            observed()
        } else {
            unknown()
        };
        let (tlc_launches, tlc_configurations) = if let Some(reason) = self.exclusion(false)? {
            if self.tlc_attempted {
                return Err(invalid("native checker exclusion contradicts observations"));
            }
            (excluded(reason), excluded(reason))
        } else {
            let covered = !self.tests.is_empty()
                && self.tests.iter().all(|id| self.coverage.contains_key(id));
            if covered && !self.tlc_attempted {
                counts.tlc_launches = Some(0);
                counts.tlc_configurations = Some(0);
            } else if !covered {
                counts.tlc_launches = None;
                counts.tlc_configurations = None;
            }
            (
                if counts.tlc_launches.is_some() { observed() } else { unknown() },
                if counts.tlc_configurations.is_some() { observed() } else { unknown() },
            )
        };
        Ok(CountEvidence { version: 1, tests, tlc_launches, tlc_configurations })
    }
}

#[cfg(test)]
mod tests {
    use super::super::{MetricEvidence, NoWorkReason};
    use super::*;
    use std::fs;
    use umpire_backend_process::bytes_sha256;

    fn collect(records: &[String]) -> io::Result<ObservedWork> {
        let root = tempfile::tempdir()?;
        for (index, record) in records.iter().enumerate() {
            fs::write(root.path().join(format!("{index}.claim")), b"")?;
            fs::write(root.path().join(format!("{index}.event")), format!("{record}\n"))?;
        }
        collect_evidenced(root.path())
    }

    fn observed() -> Vec<String> {
        vec![
            "1\tdispatch-begin\tdispatch-1-0\tobserved-tests\tobserved-tlc".into(),
            "1\ttests-begin\t1-0".into(),
            "1\ttests-finish\t1-0\t3\t2\t0\t1".into(),
            "1\tdispatch-finish\tdispatch-1-0\t0".into(),
        ]
    }

    fn covered(records: &mut Vec<String>, producer: &str) {
        let payload = format!(
            "{producer}\trust_tlc_observed_v1\t1\t{}\t{}\t{}",
            "a".repeat(64),
            "b".repeat(64),
            "c".repeat(64)
        );
        records.push(format!("1\tcoverage-begin\t{payload}"));
        records
            .push(format!("1\tcoverage-finish\t{producer}\t{}", bytes_sha256(payload.as_bytes())));
    }

    #[test]
    fn missing_dispatch_and_observers_remain_unknown() -> io::Result<()> {
        let observation = collect(&[])?;
        assert_eq!(observation.counts, WorkCounts::default());
        for metric in [
            &observation.evidence.tests,
            &observation.evidence.tlc_launches,
            &observation.evidence.tlc_configurations,
        ] {
            assert!(matches!(metric, MetricEvidence::Unknown { .. }));
        }
        observation.evidence.validate_counts(&observation.counts)
    }

    #[test]
    fn native_completion_proves_applicability_without_numeric_substitution() -> io::Result<()> {
        let observation = collect(&[
            "1\tdispatch-begin\tdispatch-1-0\tformatting\tformatting".into(),
            "1\tdispatch-finish\tdispatch-1-0\t0".into(),
        ])?;
        assert_eq!(observation.counts, WorkCounts::default());
        assert!(matches!(
            observation.evidence.tests,
            MetricEvidence::NotApplicable { reason: NoWorkReason::Formatting, .. }
        ));
        assert!(matches!(
            observation.evidence.tlc_launches,
            MetricEvidence::NotApplicable { reason: NoWorkReason::Formatting, .. }
        ));
        assert_eq!(
            observation.authority.sha256,
            bytes_sha256(observation.authority.records.as_bytes())
        );
        observation.evidence.validate_counts(&observation.counts)
    }

    #[test]
    fn complete_observer_coverage_proves_zero_but_test_finish_alone_does_not() -> io::Result<()> {
        let mut records = observed();
        let unknown = collect(&records)?;
        assert_eq!(unknown.counts.executed, Some(2));
        assert_eq!(unknown.counts.tlc_launches, None);
        assert!(matches!(unknown.evidence.tlc_launches, MetricEvidence::Unknown { .. }));
        covered(&mut records, "1-0");
        let complete = collect(&records)?;
        assert_eq!(complete.counts.tlc_launches, Some(0));
        assert_eq!(complete.counts.tlc_configurations, Some(0));
        assert!(matches!(complete.evidence.tlc_launches, MetricEvidence::Observed { .. }));
        complete.evidence.validate_counts(&complete.counts)
    }

    #[test]
    fn coverage_never_hides_unconfirmed_checker_start_or_native_contradiction() -> io::Result<()> {
        let mut records = observed();
        covered(&mut records, "1-0");
        records.extend(["1\ttlc-attempt\t1-9\t-".into(), "1\ttlc-finish\t1-9\t0".into()]);
        let unknown = collect(&records)?;
        assert_eq!(unknown.counts.tlc_launches, None);
        assert!(matches!(unknown.evidence.tlc_launches, MetricEvidence::Unknown { .. }));
        records[0] = "1\tdispatch-begin\tdispatch-1-0\tformatting\tformatting".into();
        assert!(collect(&records).is_err());
        Ok(())
    }

    #[test]
    fn duplicate_crossed_or_incomplete_scope_authorities_refuse() {
        let mut records = observed();
        covered(&mut records, "1-0");
        records.pop();
        assert!(collect(&records).is_err());
        let mut records = observed();
        covered(&mut records, "2-0");
        assert!(collect(&records).is_err());
        let mut records = observed();
        covered(&mut records, "1-0");
        records.push(records.last().unwrap().clone());
        assert!(collect(&records).is_err());
        let records = ["1\tdispatch-begin\tdispatch-1-0\tformatting\tformatting".into()];
        assert!(collect(&records).is_err());
    }
}
