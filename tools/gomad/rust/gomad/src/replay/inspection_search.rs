use super::Search;
use serde_json::{Value, json};
use std::{collections::BTreeMap, time::Duration};

pub(super) struct Evaluation {
    pub outcome: &'static str,
    pub detail: String,
    pub prefix_verified: bool,
}

#[expect(
    clippy::too_many_lines,
    reason = "Scan and monotonic bisect share one bounded evaluation cache and result state"
)]
pub(super) fn run(
    search: &Search,
    mut elapsed_time: impl FnMut() -> Duration,
    mut evaluate: impl FnMut(u64, Duration) -> Evaluation,
) -> Value {
    let mut result = json!({"status":"", "evaluations":null,"remaining":{"first":search.first,"last":search.last},"conditional_on_monotonicity":search.mode == "bisect"});
    let mut evaluations = Vec::new();
    let mut cache = BTreeMap::new();
    let mut evaluate_choice = |choice: u64, result: &mut Value| -> Option<bool> {
        if let Some(value) = cache.get(&choice) {
            return Some(*value);
        }
        let elapsed = elapsed_time();
        let wall = Duration::from_nanos(search.wall_time);
        if evaluations.len() as u64 >= search.attempt_budget || elapsed >= wall {
            result["status"] = "incomplete".into();
            return None;
        }
        let Evaluation {
            outcome,
            detail,
            prefix_verified: verified,
        } = evaluate(choice, wall.saturating_sub(elapsed));
        let timeout = elapsed_time() >= wall;
        evaluations.push(json!({"choice":choice,"outcome":if timeout { "timeout" } else { outcome },"prefix_verified":verified,"detail":if timeout { "context deadline exceeded" } else { &detail }}));
        result["evaluations"] = evaluations.clone().into();
        if timeout {
            result["status"] = "incomplete".into();
            return None;
        }
        if outcome == "divergence" || matches!(outcome, "pass" | "fail") && !verified {
            result["status"] = "replay_divergence".into();
            return None;
        }
        if !matches!(outcome, "pass" | "fail") {
            result["status"] = "invariant_evaluation_failure".into();
            return None;
        }
        let passed = outcome == "pass";
        cache.insert(choice, passed);
        if search.mode == "bisect"
            && cache.iter().any(|(position, value)| {
                !*value && passed && *position < choice || *value && !passed && *position > choice
            })
        {
            result["status"] = "monotonicity_violation".into();
            return None;
        }
        Some(passed)
    };
    let finish = |result: &mut Value, failure: Option<u64>| {
        result.as_object_mut().unwrap().remove("remaining");
        if let Some(failure) = failure {
            result["first_failure"] = failure.into();
            result["status"] = if failure == search.first {
                "already_failing_at_first"
            } else {
                "first_failure"
            }
            .into();
        } else {
            result["status"] = "no_failure".into();
        }
    };
    if search.mode == "scan" {
        let mut choice = search.first;
        loop {
            result["remaining"] = json!({"first":choice,"last":search.last});
            let Some(passed) = evaluate_choice(choice, &mut result) else {
                return result;
            };
            if !passed {
                finish(&mut result, Some(choice));
                return result;
            }
            if choice == search.last {
                finish(&mut result, None);
                return result;
            }
            choice += 1;
        }
    }
    let (mut lower, mut upper) = (search.first, search.last);
    let Some(low_pass) = evaluate_choice(lower, &mut result) else {
        return result;
    };
    if lower == upper {
        finish(&mut result, if low_pass { None } else { Some(lower) });
        return result;
    }
    let Some(high_pass) = evaluate_choice(upper, &mut result) else {
        return result;
    };
    if !low_pass {
        finish(&mut result, Some(lower));
        return result;
    }
    if high_pass {
        finish(&mut result, None);
        return result;
    }
    while upper - lower > 1 {
        result["remaining"] = json!({"first":lower + 1,"last":upper});
        let middle = lower + (upper - lower) / 2;
        let Some(passed) = evaluate_choice(middle, &mut result) else {
            return result;
        };
        if passed {
            lower = middle;
        } else {
            upper = middle;
        }
    }
    finish(&mut result, Some(upper));
    result
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::Cell;

    fn request(mode: &str) -> Search {
        Search {
            check: "healthy".into(),
            first: 0,
            last: 4,
            mode: mode.into(),
            attempt_budget: 10,
            wall_time: 1_000_000_000,
        }
    }

    fn evaluation(outcome: &'static str, prefix_verified: bool) -> Evaluation {
        Evaluation {
            outcome,
            detail: outcome.into(),
            prefix_verified,
        }
    }

    #[test]
    fn scan_does_not_skip_recovered_failures() {
        let mut called = Vec::new();
        let result = run(
            &request("scan"),
            || Duration::ZERO,
            |choice, _| {
                called.push(choice);
                evaluation(
                    if choice == 1 || choice == 4 {
                        "fail"
                    } else {
                        "pass"
                    },
                    true,
                )
            },
        );
        assert_eq!(result["status"], "first_failure");
        assert_eq!(called, [0, 1]);
        assert_eq!(result["first_failure"], 1);
        assert!(result.get("remaining").is_none());
    }

    #[test]
    fn budgets_and_predicate_errors_never_claim_first_failure() {
        for (outcome, budget, verified, status) in [
            ("pass", 2, true, "incomplete"),
            ("panic", 10, true, "invariant_evaluation_failure"),
            ("timeout", 10, true, "invariant_evaluation_failure"),
            ("divergence", 10, true, "replay_divergence"),
            ("pass", 10, false, "replay_divergence"),
        ] {
            let mut search = request("scan");
            search.attempt_budget = budget;
            let result = run(
                &search,
                || Duration::ZERO,
                |_, _| evaluation(outcome, verified),
            );
            assert_eq!(result["status"], status, "{outcome}/{budget}/{verified}");
            assert!(result.get("first_failure").is_none());
            assert!(result.get("remaining").is_some());
            assert_eq!(
                result["evaluations"].as_array().unwrap().len(),
                if budget == 2 { 2 } else { 1 }
            );
        }
    }

    #[test]
    fn bisect_declares_conditionality_and_refuses_observed_reversal() {
        for (values, status, first) in [
            ([true, true, false, false, false], "first_failure", Some(2)),
            (
                [false, false, false, false, false],
                "already_failing_at_first",
                Some(0),
            ),
            ([true, true, true, true, true], "no_failure", None),
            (
                [false, true, false, false, true],
                "monotonicity_violation",
                None,
            ),
        ] {
            let mut called = Vec::new();
            let result = run(
                &request("bisect"),
                || Duration::ZERO,
                |choice, _| {
                    assert!(
                        !called.contains(&choice),
                        "a cached decision executed twice"
                    );
                    called.push(choice);
                    evaluation(
                        if values[usize::try_from(choice).unwrap()] {
                            "pass"
                        } else {
                            "fail"
                        },
                        true,
                    )
                },
            );
            assert_eq!(result["conditional_on_monotonicity"], true);
            assert_eq!(result["status"], status);
            assert_eq!(result.get("first_failure").and_then(Value::as_u64), first);
            assert_eq!(
                result.get("remaining").is_some(),
                status == "monotonicity_violation"
            );
        }
    }

    #[test]
    fn wall_bound_includes_evaluator_cleanup() {
        let elapsed = Cell::new(Duration::from_millis(1));
        let result = run(
            &request("scan"),
            || elapsed.get(),
            |_, remaining| {
                assert_eq!(remaining, Duration::from_millis(999));
                elapsed.set(Duration::from_secs(1));
                evaluation("pass", true)
            },
        );
        assert_eq!(result["status"], "incomplete");
        assert_eq!(result["evaluations"][0]["outcome"], "timeout");
        assert_eq!(result["evaluations"].as_array().unwrap().len(), 1);
        assert!(result.get("first_failure").is_none());
    }

    #[test]
    fn expired_wall_bound_never_calls_evaluator() {
        let result = run(
            &request("scan"),
            || Duration::from_secs(1),
            |_, _| panic!("deadline expired before replay"),
        );
        assert_eq!(result["status"], "incomplete");
        assert!(result["evaluations"].is_null());
        assert!(result.get("first_failure").is_none());
    }
}
