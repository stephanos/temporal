//! Pure stage-budget admission over canonical observed counts and supplied inventory.
use super::work_counts::{CountEvidence, WorkCounts};
use std::io;

#[derive(Debug)]
pub(crate) struct Table;
#[derive(Debug)]
pub(crate) struct StageBudget;
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub(crate) enum TimeDecision {
    WithinBudget,
    RetryRequired,
}

impl Table {
    pub(crate) fn parse(_source: &str) -> io::Result<Self> {
        Err(io::Error::other("stage budget parser not implemented"))
    }
    pub(crate) fn validate_inventory(
        &self,
        _registered: &[String],
        _selected: &[String],
    ) -> io::Result<()> {
        Err(io::Error::other("stage budget inventory admission not implemented"))
    }
    pub(crate) fn budget(&self, _stage: &str) -> io::Result<&StageBudget> {
        Err(io::Error::other("stage budget lookup not implemented"))
    }
    pub(crate) fn drain_seconds(&self) -> f64 {
        0.0
    }
}

impl StageBudget {
    pub(crate) fn check_counts(
        &self,
        _counts: &WorkCounts,
        _evidence: &CountEvidence,
    ) -> io::Result<()> {
        Err(io::Error::other("stage count admission not implemented"))
    }
    pub(crate) fn check_time(&self, _seconds: f64, _attempt: u8) -> io::Result<TimeDecision> {
        Err(io::Error::other("stage time admission not implemented"))
    }
}

#[cfg(test)]
mod tests {
    use super::super::work_counts::{AuthorityRef, MetricEvidence, NoWorkReason};
    use super::*;

    fn header() -> &'static str {
        "version = 1\nadmitted = true\ndrain_seconds = 2.5\n"
    }
    fn section(name: &str, limit: &str) -> String {
        format!(
            "[stages.\"{name}\"]\nreason = 'reviewed control workload'\nwarm_seconds = 10.0\ndiscovered = {limit}\nexecuted = {limit}\nignored = {limit}\nskipped = {limit}\ntlc_launches = {limit}\ntlc_configurations = {limit}\n"
        )
    }
    fn source() -> String {
        [
            header().to_owned(),
            section("framework/tests", "100"),
            section("extension/scan", "'not-applicable'"),
        ]
        .concat()
    }
    fn authority() -> AuthorityRef {
        AuthorityRef { id: "control-producer".into(), sha256: "a".repeat(64) }
    }
    fn observed() -> MetricEvidence {
        MetricEvidence::Observed { authority: authority() }
    }
    fn evidence() -> CountEvidence {
        CountEvidence {
            version: 1,
            tests: observed(),
            tlc_launches: observed(),
            tlc_configurations: observed(),
        }
    }
    fn no_work() -> MetricEvidence {
        MetricEvidence::NotApplicable { authority: authority(), reason: NoWorkReason::NativeScan }
    }
    fn no_work_evidence() -> CountEvidence {
        CountEvidence {
            version: 1,
            tests: no_work(),
            tlc_launches: no_work(),
            tlc_configurations: no_work(),
        }
    }
    fn counts() -> WorkCounts {
        WorkCounts {
            discovered: Some(4),
            executed: Some(2),
            ignored: Some(1),
            skipped: Some(1),
            tlc_launches: Some(2),
            tlc_configurations: Some(1),
            ..WorkCounts::default()
        }
    }

    #[test]
    fn registered_inventory_and_selected_child_subset_are_admitted() -> io::Result<()> {
        let table = Table::parse(&source())?;
        table.validate_inventory(
            &["framework/tests".into(), "extension/scan".into()],
            &["framework/tests".into()],
        )?;
        assert_eq!(table.drain_seconds(), 2.5);
        table.budget("framework/tests")?.check_counts(&counts(), &evidence())?;
        table
            .budget("extension/scan")?
            .check_counts(&WorkCounts::default(), &no_work_evidence())?;
        let registered: Vec<_> =
            (0..18).map(|index| format!("framework/control-{index}")).collect();
        let selected = registered[..17].to_vec();
        let fixture = format!(
            "{}{}",
            header(),
            (0..18)
                .map(|index| section(&format!("framework/control-{index}"), "100"))
                .collect::<String>()
        );
        Table::parse(&fixture)?.validate_inventory(&registered, &selected)?;
        Ok(())
    }

    #[test]
    fn actual_registry_omissions_unknowns_duplicates_and_crossed_selection_refuse() -> io::Result<()>
    {
        let table = Table::parse(&source())?;
        for (registered, selected) in [
            (vec!["framework/tests"], vec!["framework/tests"]),
            (
                vec!["framework/tests", "extension/scan", "framework/unknown"],
                vec!["framework/tests"],
            ),
            (vec!["framework/tests", "extension/scan", "framework/tests"], vec!["framework/tests"]),
            (vec!["framework/tests", "extension/scan"], vec!["framework/unknown"]),
            (vec!["framework/tests", "extension/scan"], vec!["framework/tests", "framework/tests"]),
        ] {
            assert!(
                table
                    .validate_inventory(
                        &registered.into_iter().map(String::from).collect::<Vec<_>>(),
                        &selected.into_iter().map(String::from).collect::<Vec<_>>()
                    )
                    .is_err()
            );
        }
        assert!(table.budget("framework/unknown").is_err());
        Ok(())
    }

    #[test]
    fn unadmitted_tables_remain_planning_data_and_cannot_supply_execution_budgets() -> io::Result<()>
    {
        let table = Table::parse(&source().replace("admitted = true", "admitted = false"))?;
        assert!(table.budget("framework/tests").is_err());
        Ok(())
    }

    #[test]
    fn parser_rejects_bad_schema_times_identity_and_partial_applicability() {
        for broken in [
            source().replace("version = 1", "version = 2"),
            source().replace("version = 1", "version = 1\nextra = true"),
            source().replace("version = 1", "version = 1\nversion = 1"),
            source().replace("drain_seconds = 2.5", "drain_seconds = 0.0"),
            source().replace("drain_seconds = 2.5", "drain_seconds = nan"),
            source().replace("drain_seconds = 2.5", "drain_seconds = inf"),
            source().replace("warm_seconds = 10.0", "warm_seconds = -1.0"),
            source().replace("warm_seconds = 10.0", "warm_seconds = nan"),
            source().replace("warm_seconds = 10.0", "warm_seconds = inf"),
            source().replace("reason = 'reviewed control workload'", "reason = '   '"),
            source().replace(
                "reason = 'reviewed control workload'",
                "reason = 'reviewed'\nextra = true",
            ),
            source().replace("framework/tests", "tests"),
            source().replace("framework/tests", "framework/nested/tests"),
            source().replace("framework/tests", "framework/../tests"),
            source().replace("framework/tests", " framework/tests"),
            source().replace("discovered = 100", "discovered = 'not-applicable'"),
            source().replace("discovered = 100", "discovered = -1"),
            source().replace("discovered = 100", "discovered = 1.0"),
            source().replace("discovered = 100", "discovered = 18446744073709551616"),
            source().replace("'not-applicable'", "'unknown'"),
            source().replace("'not-applicable'", "'not_applicable'"),
            source().replace("discovered = 100\n", ""),
            format!("{}{}", source(), section("framework/tests", "100")),
        ] {
            assert!(Table::parse(&broken).is_err(), "accepted {broken}");
        }
    }

    #[test]
    fn every_counter_is_a_hard_bound_and_zero_is_an_observed_limit() -> io::Result<()> {
        for metric in
            ["discovered", "executed", "ignored", "skipped", "tlc_launches", "tlc_configurations"]
        {
            let source = source().replacen(&format!("{metric} = 100"), &format!("{metric} = 0"), 1);
            let table = Table::parse(&source)?;
            let error = table
                .budget("framework/tests")?
                .check_counts(&counts(), &evidence())
                .expect_err("over-budget actual count");
            let error = error.to_string();
            assert!(error.contains("framework/tests") && error.contains(metric), "{error}");
        }
        let source = format!("{}{}", header(), section("framework/zero", "0"));
        let zero = WorkCounts {
            discovered: Some(0),
            executed: Some(0),
            ignored: Some(0),
            skipped: Some(0),
            tlc_launches: Some(0),
            tlc_configurations: Some(0),
            ..WorkCounts::default()
        };
        Table::parse(&source)?.budget("framework/zero")?.check_counts(&zero, &evidence())?;
        Ok(())
    }

    #[test]
    fn unknown_is_neither_zero_nor_not_applicable_and_evidence_must_match_counts() -> io::Result<()>
    {
        let table = Table::parse(&source())?;
        assert!(
            table.budget("framework/tests")?.check_counts(&counts(), &no_work_evidence()).is_err()
        );
        assert!(
            table
                .budget("extension/scan")?
                .check_counts(&WorkCounts::default(), &evidence())
                .is_err()
        );
        assert!(
            table.budget("extension/scan")?.check_counts(&counts(), &no_work_evidence()).is_err()
        );
        for target in ["tests", "launches", "configurations"] {
            let mut evidence = evidence();
            let unknown = MetricEvidence::Unknown { reason: "incomplete observer".into() };
            let mut counts = counts();
            match target {
                "tests" => {
                    evidence.tests = unknown;
                    counts.discovered = None;
                    counts.executed = None;
                    counts.ignored = None;
                    counts.skipped = None;
                }
                "launches" => {
                    evidence.tlc_launches = unknown;
                    counts.tlc_launches = None;
                    counts.tlc_configurations = None;
                    evidence.tlc_configurations =
                        MetricEvidence::Unknown { reason: "unconfirmed launch".into() };
                }
                _ => {
                    evidence.tlc_configurations = unknown;
                    counts.tlc_configurations = None;
                }
            }
            assert!(table.budget("framework/tests")?.check_counts(&counts, &evidence).is_err());
            assert!(table.budget("extension/scan")?.check_counts(&counts, &evidence).is_err());
        }
        let mut invalid = counts();
        invalid.executed = Some(100);
        assert!(table.budget("framework/tests")?.check_counts(&invalid, &evidence()).is_err());
        let mut invalid = counts();
        invalid.discovered = None;
        assert!(table.budget("framework/tests")?.check_counts(&invalid, &evidence()).is_err());
        let mut invalid = evidence();
        invalid.version = 2;
        assert!(table.budget("framework/tests")?.check_counts(&counts(), &invalid).is_err());
        Ok(())
    }

    #[test]
    fn fixed_tolerance_requests_only_one_retry_and_second_fast_is_admitted() -> io::Result<()> {
        let table = Table::parse(&source())?;
        let budget = table.budget("framework/tests")?;
        for attempt in [1, 2] {
            assert_eq!(budget.check_time(0.0, attempt)?, TimeDecision::WithinBudget);
            assert_eq!(budget.check_time(12.5, attempt)?, TimeDecision::WithinBudget);
        }
        assert_eq!(budget.check_time(12.500001, 1)?, TimeDecision::RetryRequired);
        assert_eq!(budget.check_time(10.0, 2)?, TimeDecision::WithinBudget);
        assert!(budget.check_time(12.500001, 2).is_err());
        for (seconds, attempt) in [
            (-1.0, 1),
            (f64::NAN, 1),
            (f64::INFINITY, 1),
            (f64::NEG_INFINITY, 1),
            (1.0, 0),
            (1.0, 3),
        ] {
            assert!(budget.check_time(seconds, attempt).is_err());
        }
        Ok(())
    }
}
