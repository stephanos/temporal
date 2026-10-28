//! Pure stage-budget admission over canonical observed counts and supplied inventory.
use super::work_counts::{CountEvidence, MetricEvidence, WorkCounts};
use serde::{Deserialize, Deserializer};
use std::{
    collections::{BTreeMap, BTreeSet},
    io,
};

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum Limit {
    Bounded(u64),
    NotApplicable,
}

impl<'de> Deserialize<'de> for Limit {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        #[derive(Deserialize)]
        #[serde(untagged)]
        enum Value {
            Bounded(u64),
            Label(String),
        }
        match Value::deserialize(deserializer)? {
            Value::Bounded(value) => Ok(Self::Bounded(value)),
            Value::Label(value) if value == "not-applicable" => Ok(Self::NotApplicable),
            Value::Label(_) => Err(serde::de::Error::custom(
                "metric limit must be unsigned integer or exact not-applicable string",
            )),
        }
    }
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct StageSpec {
    reason: String,
    warm_seconds: f64,
    discovered: Limit,
    executed: Limit,
    ignored: Limit,
    skipped: Limit,
    tlc_launches: Limit,
    tlc_configurations: Limit,
}

impl StageSpec {
    fn limits(&self) -> [Limit; 6] {
        [
            self.discovered,
            self.executed,
            self.ignored,
            self.skipped,
            self.tlc_launches,
            self.tlc_configurations,
        ]
    }
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct TableSpec {
    version: u8,
    admitted: bool,
    drain_seconds: f64,
    stages: BTreeMap<String, StageSpec>,
}

#[derive(Debug)]
pub(crate) struct Table {
    admitted: bool,
    drain_seconds: f64,
    stages: BTreeMap<String, StageBudget>,
}
#[derive(Debug)]
pub(crate) struct StageBudget {
    name: String,
    spec: StageSpec,
}
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub(crate) enum TimeDecision {
    WithinBudget,
    RetryRequired,
}

fn refusal(stage: &str, reason: &str) -> io::Error {
    io::Error::other(format!("stage {stage} budget: {reason}"))
}

fn stage_identity(name: &str) -> io::Result<()> {
    let parts = name.split('/').collect::<Vec<_>>();
    if parts.len() != 2
        || parts.iter().any(|part| {
            part.is_empty()
                || !part
                    .bytes()
                    .all(|byte| byte.is_ascii_lowercase() || byte.is_ascii_digit() || byte == b'-')
                || !part.as_bytes().first().is_some_and(u8::is_ascii_alphanumeric)
                || !part.as_bytes().last().is_some_and(u8::is_ascii_alphanumeric)
        })
    {
        return Err(refusal(name, "invalid scoped stage identity"));
    }
    Ok(())
}

impl Table {
    pub(crate) fn parse(source: &str) -> io::Result<Self> {
        let table: TableSpec = toml::from_str(source).map_err(io::Error::other)?;
        if table.version != 1
            || !table.drain_seconds.is_finite()
            || table.drain_seconds <= 0.0
            || table.stages.is_empty()
        {
            return Err(io::Error::other(
                "invalid stage budget table version, drain bound or empty stages",
            ));
        }
        let mut stages = BTreeMap::new();
        for (name, spec) in table.stages {
            stage_identity(&name)?;
            if spec.reason.trim().is_empty()
                || !spec.warm_seconds.is_finite()
                || spec.warm_seconds <= 0.0
            {
                return Err(refusal(&name, "missing review reason or invalid warm-time budget"));
            }
            let limits = spec.limits();
            let quartet = &limits[..4];
            if quartet.iter().any(|limit| matches!(limit, Limit::Bounded(_)))
                && quartet.contains(&Limit::NotApplicable)
            {
                return Err(refusal(
                    &name,
                    "test limits must be all numeric or all not-applicable",
                ));
            }
            stages.insert(name.clone(), StageBudget { name, spec });
        }
        Ok(Self { admitted: table.admitted, drain_seconds: table.drain_seconds, stages })
    }

    pub(crate) fn validate_inventory(
        &self,
        registered: &[String],
        selected: &[String],
    ) -> io::Result<()> {
        for name in registered.iter().chain(selected) {
            stage_identity(name)?;
        }
        let actual: BTreeSet<_> = registered.iter().map(String::as_str).collect();
        let chosen: BTreeSet<_> = selected.iter().map(String::as_str).collect();
        if actual.len() != registered.len() || chosen.len() != selected.len() {
            return Err(io::Error::other("duplicate registered or selected budget stage"));
        }
        let declared: BTreeSet<_> = self.stages.keys().map(String::as_str).collect();
        if declared != actual {
            let missing = actual.difference(&declared).copied().collect::<Vec<_>>();
            let phantom = declared.difference(&actual).copied().collect::<Vec<_>>();
            return Err(io::Error::other(format!(
                "stage budget registry mismatch: missing {missing:?}, unregistered {phantom:?}"
            )));
        }
        if !chosen.is_subset(&actual) {
            return Err(io::Error::other("selected budget stage is not registered"));
        }
        Ok(())
    }

    pub(crate) fn budget(&self, stage: &str) -> io::Result<&StageBudget> {
        if !self.admitted {
            return Err(refusal(stage, "table is not admitted"));
        }
        self.stages.get(stage).ok_or_else(|| refusal(stage, "missing registered budget"))
    }

    pub(crate) fn drain_seconds(&self) -> f64 {
        self.drain_seconds
    }
}

impl StageBudget {
    pub(crate) fn check_counts(
        &self,
        counts: &WorkCounts,
        evidence: &CountEvidence,
    ) -> io::Result<()> {
        evidence
            .validate_counts(counts)
            .map_err(|error| refusal(&self.name, &error.to_string()))?;
        let values = [
            counts.discovered,
            counts.executed,
            counts.ignored,
            counts.skipped,
            counts.tlc_launches,
            counts.tlc_configurations,
        ];
        let evidence = [
            &evidence.tests,
            &evidence.tests,
            &evidence.tests,
            &evidence.tests,
            &evidence.tlc_launches,
            &evidence.tlc_configurations,
        ];
        for (((metric, limit), actual), evidence) in
            ["discovered", "executed", "ignored", "skipped", "tlc_launches", "tlc_configurations"]
                .into_iter()
                .zip(self.spec.limits())
                .zip(values)
                .zip(evidence)
        {
            match (limit, actual, evidence) {
                (Limit::Bounded(maximum), Some(actual), MetricEvidence::Observed { .. }) => {
                    if actual > maximum {
                        return Err(refusal(
                            &self.name,
                            &format!("{metric} {actual} exceeds {maximum}"),
                        ));
                    }
                }
                (Limit::NotApplicable, None, MetricEvidence::NotApplicable { .. }) => (),
                _ => {
                    return Err(refusal(
                        &self.name,
                        &format!("{metric} limit does not match complete count evidence"),
                    ));
                }
            }
        }
        Ok(())
    }

    pub(crate) fn check_time(&self, seconds: f64, attempt: u8) -> io::Result<TimeDecision> {
        if !seconds.is_finite() || seconds < 0.0 || ![1, 2].contains(&attempt) {
            return Err(refusal(&self.name, "invalid elapsed time or attempt"));
        }
        if seconds / self.spec.warm_seconds <= 1.25 {
            return Ok(TimeDecision::WithinBudget);
        }
        if attempt == 1 {
            return Ok(TimeDecision::RetryRequired);
        }
        Err(refusal(
            &self.name,
            &format!(
                "second attempt {seconds}s exceeds warm budget {}s plus 25%",
                self.spec.warm_seconds
            ),
        ))
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
            source().replace("warm_seconds = 10.0", "warm_seconds = 0.0"),
            source().replace("warm_seconds = 10.0", "warm_seconds = -1.0"),
            source().replace("warm_seconds = 10.0", "warm_seconds = nan"),
            source().replace("warm_seconds = 10.0", "warm_seconds = inf"),
            source().replace("reason = 'reviewed control workload'", "reason = '   '"),
            source().replace(
                "reason = 'reviewed control workload'",
                "reason = 'reviewed'\nextra = true",
            ),
            source().replace("framework/tests", "tests"),
            source().replace("framework/tests", "Framework/tests"),
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
        invalid.tests = MetricEvidence::Observed {
            authority: AuthorityRef { id: "bad/path".into(), sha256: "bad".into() },
        };
        assert!(table.budget("framework/tests")?.check_counts(&counts(), &invalid).is_err());
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
