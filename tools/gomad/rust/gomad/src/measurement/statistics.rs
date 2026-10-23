use std::collections::BTreeMap;

#[derive(Clone, Copy, Debug, Default, Eq, PartialEq)]
pub struct TimingRange {
    pub median: u64,
    pub min: u64,
    pub max: u64,
}

#[must_use]
pub fn timing(values: &[u64]) -> TimingRange {
    let mut ordered = values.to_vec();
    ordered.sort_unstable();
    ordered
        .first()
        .map_or(TimingRange::default(), |minimum| TimingRange {
            median: ordered[ordered.len() / 2],
            min: *minimum,
            max: ordered[ordered.len() - 1],
        })
}

#[must_use]
pub fn recommendation(
    baseline: &[u64],
    candidate: &[u64],
    failed: bool,
    identical: bool,
) -> &'static str {
    if failed || !identical {
        return "reject";
    }
    if baseline.is_empty() || candidate.is_empty() {
        return "investigate";
    }
    let before = timing(baseline);
    let after = timing(candidate);
    if after.median >= before.median {
        "reject"
    } else if before.median - after.median > (before.max - before.min).max(after.max - after.min) {
        "adopt"
    } else {
        "investigate"
    }
}

#[must_use]
pub fn reap_consistent(observed_wall_nanos: u64, recorded_monotonic_nanos: u64) -> bool {
    observed_wall_nanos > 0
        && recorded_monotonic_nanos > 0
        && recorded_monotonic_nanos <= i64::MAX as u64
        && observed_wall_nanos.abs_diff(recorded_monotonic_nanos)
            <= 10_000.max(recorded_monotonic_nanos / 1000)
}

#[derive(Clone, Debug)]
pub struct NamespaceSample {
    pub role: String,
    pub seed: u64,
    pub repetition: usize,
    pub authorized_nanos: u64,
    pub evidence_sha256: String,
    pub environment_identity: String,
    pub calls: u64,
    pub time_advances: u64,
    pub counter_present: bool,
    pub external_filesystem_calls: BTreeMap<String, u64>,
}

fn digest(value: &str) -> bool {
    value.len() == 64 && value.bytes().all(|byte| byte.is_ascii_hexdigit())
}

/// Compares four matched seed cohorts; an actual empty external-owner counter is required.
/// # Errors
/// Refuses missing observations, mismatched evidence or any per-seed median above 75 percent.
pub fn compare_namespace(
    before: &[NamespaceSample],
    after: &[NamespaceSample],
) -> Result<(), String> {
    if before.len() != 20 || after.len() != before.len() {
        return Err("four matched seeds with five fresh repetitions required".into());
    }
    for (seed_index, seed) in [0, 1_000_003, 2_000_006, 3_000_009].into_iter().enumerate() {
        let mut baseline = Vec::new();
        let mut candidate = Vec::new();
        for repetition in 0..5 {
            let index = seed_index * 5 + repetition;
            let b = &before[index];
            let a = &after[index];
            if b.role != "baseline"
                || a.role != "candidate"
                || b.seed != seed
                || a.seed != seed
                || b.repetition != repetition
                || a.repetition != repetition
                || b.authorized_nanos == 0
                || a.authorized_nanos == 0
                || !a.counter_present
                || !a.external_filesystem_calls.is_empty()
            {
                return Err(format!(
                    "seed {seed} repetition {repetition} lacks matched timings or empty actual counter"
                ));
            }
            if !digest(&b.evidence_sha256)
                || a.evidence_sha256 != b.evidence_sha256
                || b.environment_identity.is_empty()
                || a.environment_identity != b.environment_identity
                || a.calls != b.calls
                || a.time_advances != b.time_advances
                || b.evidence_sha256 != before[seed_index * 5].evidence_sha256
            {
                return Err(format!(
                    "seed {seed} repetition {repetition} changed canonical evidence or model counts"
                ));
            }
            baseline.push(b.authorized_nanos);
            candidate.push(a.authorized_nanos);
        }
        let b = timing(&baseline).median;
        let a = timing(&candidate).median;
        if a > b / 4 * 3 + b % 4 * 3 / 4 {
            return Err(format!(
                "seed {seed} authorized median {a} exceeds 0.75 baseline {b}"
            ));
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn reliable_gain_preserves_strict_spread_and_failure_precedence() {
        for (before, after, failed, identical, expected) in [
            (
                vec![98, 99, 100, 101, 102],
                vec![78, 79, 80, 81, 82],
                false,
                true,
                "adopt",
            ),
            (
                vec![90, 99, 100, 101, 110],
                vec![80, 89, 90, 91, 100],
                false,
                true,
                "investigate",
            ),
            (
                vec![98, 99, 100, 101, 102],
                vec![108, 109, 110, 111, 112],
                false,
                true,
                "reject",
            ),
            (
                vec![98, 99, 100, 101, 102],
                vec![78, 79, 80, 81, 82],
                false,
                false,
                "reject",
            ),
            (vec![98, 99, 100, 101, 102], vec![], true, false, "reject"),
            (vec![100], vec![], false, true, "investigate"),
            (vec![u64::MAX], vec![u64::MAX - 1], false, true, "adopt"),
        ] {
            assert_eq!(recommendation(&before, &after, failed, identical), expected);
        }
        assert_eq!(timing(&[10, 1, 5, 3]).median, 5);
        assert_eq!(timing(&[]), TimingRange::default());
    }

    fn namespace() -> (Vec<NamespaceSample>, Vec<NamespaceSample>) {
        let mut before = Vec::new();
        let mut after = Vec::new();
        for seed in [0, 1_000_003, 2_000_006, 3_000_009] {
            for repetition in 0..5 {
                let sample = NamespaceSample {
                    role: "baseline".into(),
                    seed,
                    repetition,
                    authorized_nanos: 100,
                    evidence_sha256: "e".repeat(64),
                    environment_identity: "environment".into(),
                    calls: 10,
                    time_advances: 3,
                    counter_present: false,
                    external_filesystem_calls: BTreeMap::new(),
                };
                before.push(sample.clone());
                after.push(NamespaceSample {
                    role: "candidate".into(),
                    authorized_nanos: 75,
                    counter_present: true,
                    ..sample
                });
            }
        }
        (before, after)
    }

    #[test]
    fn namespace_original_twenty_sample_mutations_and_overflow_boundary() {
        let (before, after) = namespace();
        compare_namespace(&before, &after).unwrap();
        let mut outlier = after.clone();
        outlier[0].authorized_nanos = 1000;
        compare_namespace(&before, &outlier).unwrap();
        for mutation in 0..12 {
            let mut changed = after.clone();
            match mutation {
                0 => {
                    for row in &mut changed[5..10] {
                        row.authorized_nanos += 1;
                    }
                }
                1 => changed[0].seed += 1,
                2 => changed[0].repetition += 1,
                3 => changed[0].authorized_nanos = 0,
                4 => {
                    changed[0]
                        .external_filesystem_calls
                        .insert("fd_write".into(), 0);
                }
                5 => {
                    changed[0]
                        .external_filesystem_calls
                        .insert("fd_pwrite".into(), 1);
                }
                6 => changed[0].counter_present = false,
                7 => changed[0].evidence_sha256 = "f".repeat(64),
                8 => changed[0].calls += 1,
                9 => changed[0].time_advances += 1,
                10 => changed[0].environment_identity = "other".into(),
                _ => {
                    changed.pop();
                }
            }
            assert!(
                compare_namespace(&before, &changed).is_err(),
                "mutation {mutation}"
            );
        }
        let mut maximum = before;
        let mut candidate = after;
        for row in &mut maximum {
            row.authorized_nanos = u64::MAX;
        }
        for row in &mut candidate {
            row.authorized_nanos = u64::MAX / 4 * 3 + u64::MAX % 4 * 3 / 4;
        }
        compare_namespace(&maximum, &candidate).unwrap();
        for row in &mut candidate {
            row.authorized_nanos += 1;
        }
        assert!(compare_namespace(&maximum, &candidate).is_err());
    }

    #[test]
    fn reap_tolerates_only_original_clock_slew_bounds() {
        assert!(reap_consistent(100_001_000, 100_000_000));
        assert!(reap_consistent(110_000, 100_000));
        assert!(!reap_consistent(110_001, 100_000));
        assert!(!reap_consistent(0, 100));
        assert!(!reap_consistent(100, 0));
        assert!(!reap_consistent(u64::MAX, u64::MAX));
    }
}
