//! Applicability and observer coverage linked to exact retained producer events.
use super::{CountEvidence, WorkCounts};
use serde::{Deserialize, Serialize};
use std::{io, path::Path};

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct CountAuthority {
    /// Invocation-local identity referenced by modality evidence.
    pub id: String,
    /// SHA-256 of the exact retained authority bytes.
    pub sha256: String,
    /// Canonical manifest of event names and their content digests.
    pub records: String,
}

pub struct ObservedWork {
    /// Completed numeric observations, without expected-count substitution.
    pub counts: WorkCounts,
    /// Provenance and applicability of each modality.
    pub evidence: CountEvidence,
    /// Exact event manifest backing this stage's evidence.
    pub authority: CountAuthority,
}

/// Collect current-stage evidence without treating missing observers as zero.
pub fn collect_evidenced(_directory: &Path) -> io::Result<ObservedWork> {
    Err(io::Error::other("count authority implementation pending"))
}

#[cfg(test)]
mod tests {
    use super::*;
    use super::super::{MetricEvidence, NoWorkReason};
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
        let payload = format!("{producer}\trust_tlc_observed_v1\t1\t{}\t{}\t{}", "a".repeat(64), "b".repeat(64), "c".repeat(64));
        records.push(format!("1\tcoverage-begin\t{payload}"));
        records.push(format!("1\tcoverage-finish\t{producer}\t{}", bytes_sha256(payload.as_bytes())));
    }

    #[test]
    fn missing_dispatch_and_observers_remain_unknown() -> io::Result<()> {
        let observation = collect(&[])?;
        assert_eq!(observation.counts, WorkCounts::default());
        for metric in [&observation.evidence.tests, &observation.evidence.tlc_launches, &observation.evidence.tlc_configurations] {
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
        assert!(matches!(observation.evidence.tests, MetricEvidence::NotApplicable { reason: NoWorkReason::Formatting, .. }));
        assert!(matches!(observation.evidence.tlc_launches, MetricEvidence::NotApplicable { reason: NoWorkReason::Formatting, .. }));
        assert_eq!(observation.authority.sha256, bytes_sha256(observation.authority.records.as_bytes()));
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
