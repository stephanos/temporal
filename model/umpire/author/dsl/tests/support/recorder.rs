//! Recorder control flow, precedence, scope restoration and thread isolation.

use std::{cell::Cell, panic::catch_unwind};

use umpire_dsl::{Hole, Verdict, check, claim, unknown};

#[test]
fn checks_record_first_failure_without_stopping_control_flow() {
    let calls = Cell::new(0);
    let result = claim::evaluate(|| {
        check!({
            calls.set(calls.get() + 1);
            false
        });
        check!(false);
        calls.set(calls.get() + 1);
    });
    assert_eq!(calls.get(), 2);
    assert_eq!(result.verdict, Verdict::Fails);
    assert!(result.failure.is_some_and(|text| text.contains("calls.set")));
}

#[test]
fn failures_dominate_holes_in_both_orders_and_first_hole_wins() {
    for hole_first in [false, true] {
        let result = claim::evaluate(|| {
            if hole_first {
                unknown!("gap");
            }
            check!(false);
            if !hole_first {
                unknown!("gap");
            }
        });
        assert_eq!(result.verdict, Verdict::Fails);
        assert_eq!(result.failure, Some("false"));
    }
    assert_eq!(
        claim::evaluate(|| {
            unknown!("first");
            unknown!("second");
        })
        .verdict,
        Verdict::Unknown(Hole { reason: "first" })
    );
}

#[test]
fn only_reached_checks_count_after_ordinary_early_returns() {
    fn conditional(enabled: bool) {
        if !enabled {
            return;
        }
        check!(false);
    }
    assert_eq!(claim::evaluate(|| conditional(false)).verdict, Verdict::Holds);
    assert_eq!(claim::evaluate(|| conditional(true)).verdict, Verdict::Fails);
}

#[test]
fn direct_nested_and_panicking_calls_restore_evaluation() {
    check!(false);
    unknown!("inactive");
    claim::fail("inactive direct failure");
    assert_eq!(claim::evaluate(|| {}).verdict, Verdict::Holds);
    let outer = claim::evaluate(|| {
        check!(1 == 2);
        assert_eq!(claim::evaluate(|| check!(false)).failure, Some("false"));
        assert!(
            catch_unwind(|| claim::evaluate(|| {
                check!(false);
                panic!("model panic");
            }))
            .is_err()
        );
        unknown!("outer gap");
    });
    assert_eq!(outer.failure, Some("1 == 2"));
    assert_eq!(claim::evaluate(|| {}).verdict, Verdict::Holds);
}

#[test]
#[allow(
    clippy::disallowed_methods,
    clippy::disallowed_types,
    reason = "native test of recorder isolation across threads"
)]
fn evaluations_are_thread_local() {
    let result = claim::evaluate(|| {
        let child = std::thread::spawn(|| claim::evaluate(|| check!(false))).join();
        let Ok(child) = child else { panic!("child evaluation panicked") };
        assert_eq!(child.failure, Some("false"));
    });
    assert_eq!(result.verdict, Verdict::Holds);
}
