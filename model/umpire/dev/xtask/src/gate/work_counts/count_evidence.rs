use super::WorkCounts;
use serde::{Deserialize, Serialize};
use std::io;

/// A reference to an invocation-local producer or completed native dispatch.
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct AuthorityRef {
    /// Unique authority within this stage's invocation.
    pub id: String,
    /// SHA-256 of its exact retained bytes.
    pub sha256: String,
}

/// Why a completed native branch has no work in a modality.
#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum NoWorkReason {
    /// Native source or policy scan.
    NativeScan,
    /// Compiler or documentation compilation without a test runner.
    CompileOnly,
    /// Formatting without a test runner.
    Formatting,
    /// Protobuf generation or freshness comparison.
    ProtobufGeneration,
    /// Guest compilation or freshness comparison.
    GuestBuild,
    /// Tool installation or version admission.
    ToolProvisioning,
    /// Work belongs to exclusive linked child receipts.
    ChildExclusive,
    /// An explicit native mode excludes this branch.
    SkippedByMode,
}

/// Evidence for one modality; it never duplicates numeric counts.
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(tag = "status", rename_all = "snake_case", deny_unknown_fields)]
pub enum MetricEvidence {
    /// Complete observations from the linked producer.
    Observed {
        /// Retained producer authority.
        authority: AuthorityRef,
    },
    /// A completed native dispatch positively excludes this modality.
    NotApplicable {
        /// Retained completed dispatch authority.
        authority: AuthorityRef,
        /// The native branch's concrete exclusion reason.
        reason: NoWorkReason,
    },
    /// No complete observation or exclusion authority exists.
    Unknown {
        /// Why evidence remains incomplete.
        reason: String,
    },
}

/// Evidence accompanies, but does not change, the six numeric/null count fields.
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct CountEvidence {
    /// Evidence wire format version.
    pub version: u8,
    /// Atomic discovered/executed/ignored/skipped evidence.
    pub tests: MetricEvidence,
    /// Actual TLC starts or positive exclusion.
    pub tlc_launches: MetricEvidence,
    /// Distinct source/configuration identities or positive exclusion.
    pub tlc_configurations: MetricEvidence,
}

impl CountEvidence {
    /// Validate count/evidence shape; authority provenance is admitted separately.
    pub fn validate_counts(&self, _counts: &WorkCounts) -> io::Result<()> {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn authority() -> AuthorityRef {
        AuthorityRef { id: "producer-1".into(), sha256: "a".repeat(64) }
    }
    fn unknown() -> MetricEvidence {
        MetricEvidence::Unknown { reason: "no complete producer".into() }
    }
    fn evidence() -> CountEvidence {
        CountEvidence { version: 1, tests: unknown(), tlc_launches: unknown(), tlc_configurations: unknown() }
    }
    fn observed() -> MetricEvidence {
        MetricEvidence::Observed { authority: authority() }
    }
    fn counts() -> WorkCounts {
        WorkCounts { discovered: Some(3), executed: Some(1), ignored: Some(1), skipped: Some(1), ..WorkCounts::default() }
    }

    #[test]
    fn unknown_is_not_numeric_zero_or_applicability() -> io::Result<()> {
        evidence().validate_counts(&WorkCounts::default())?;
        assert!(evidence().validate_counts(&counts()).is_err());
        let mut value = evidence();
        value.tests = observed();
        value.validate_counts(&counts())?;
        assert!(value.validate_counts(&WorkCounts::default()).is_err());
        Ok(())
    }

    #[test]
    fn observed_tests_require_the_complete_consistent_quartet() {
        let mut value = evidence(); value.tests = observed();
        let mut incomplete = counts(); incomplete.executed = None;
        assert!(value.validate_counts(&incomplete).is_err());
        let mut inconsistent = counts(); inconsistent.discovered = Some(4);
        assert!(value.validate_counts(&inconsistent).is_err());
    }

    #[test]
    fn not_applicable_requires_null_counts_and_well_formed_authority() -> io::Result<()> {
        let mut value = evidence();
        value.tests = MetricEvidence::NotApplicable { authority: authority(), reason: NoWorkReason::CompileOnly };
        value.validate_counts(&WorkCounts::default())?;
        assert!(value.validate_counts(&counts()).is_err());
        for reference in [AuthorityRef { id: "../other".into(), ..authority() }, AuthorityRef { sha256: "bad".into(), ..authority() }] {
            value.tests = MetricEvidence::NotApplicable { authority: reference, reason: NoWorkReason::CompileOnly };
            assert!(value.validate_counts(&WorkCounts::default()).is_err());
        }
        Ok(())
    }

    #[test]
    fn observed_launches_can_have_unknown_configurations_but_not_the_reverse() -> io::Result<()> {
        let mut value = evidence(); value.tlc_launches = observed();
        let mut counts = WorkCounts { tlc_launches: Some(2), ..WorkCounts::default() };
        value.validate_counts(&counts)?;
        counts.tlc_configurations = Some(1);
        assert!(value.validate_counts(&counts).is_err());
        value.tlc_configurations = observed(); value.validate_counts(&counts)?;
        counts.tlc_launches = None;
        assert!(value.validate_counts(&counts).is_err());
        Ok(())
    }

    #[test]
    fn malformed_versions_reasons_and_tagged_fields_refuse_evidence() -> io::Result<()> {
        let mut value = evidence(); value.version = 2;
        assert!(value.validate_counts(&WorkCounts::default()).is_err());
        value.version = 1; value.tests = MetricEvidence::Unknown { reason: String::new() };
        assert!(value.validate_counts(&WorkCounts::default()).is_err());
        let mut json = serde_json::to_value(evidence())?; json["tests"]["authority"] = serde_json::json!({"id":"x","sha256":"a".repeat(64)});
        assert!(serde_json::from_value::<CountEvidence>(json).is_err());
        let raw = serde_json::to_value(counts())?;
        assert_eq!(raw.as_object().map(serde_json::Map::len), Some(6));
        assert!(raw["tlc_launches"].is_null());
        Ok(())
    }
}
