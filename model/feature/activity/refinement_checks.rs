//! Native full-state refinement of the independent activity protocol by the feature Model,
//! through the registry's declared `refines` maps and the read's observable target.

use umpire_dsl::{Machine, Outcome, Steps};
use umpire_runtime::ReachTarget as _;
use umpire_temporal::{
    Realization, activity,
    activity_protocol::{adapter, parameters::Parameters, protocol, view::View},
    projection,
    registry::Refined,
};
use umpire_testing::refinement::{Mapping, Mutant, Relation, Report, Verdict, refine, render};

const BUDGET: usize = 4096;
type TestResult = umpire_testing::TestResult;
type Error = Box<dyn std::error::Error>;

fn declared() -> umpire_temporal::registry::Refines<
    protocol::World,
    activity::World,
    umpire_temporal::ActivityTarget,
> {
    <Realization as Refined>::refines()
}

/// The generic relation built only from the registry declaration.
fn relation() -> Relation<protocol::World, activity::World> {
    Relation {
        state: declared().state,
        action: |action| Mapping::Mapped((declared().action)(action)),
        outcome: declared().outcome,
        fact: declared().fact,
        silent: |outcome| *outcome == Outcome::Accepted,
        compatible: Some(|state, image| (declared().view)(state).matches(image)),
    }
}

fn worlds(scope: Scope) -> Result<(protocol::World, activity::World), Error> {
    let exact = Parameters::new(scope.0, scope.1)?;
    Ok((adapter::world(exact)?, projection::world(exact)?))
}

fn label(scope: Scope) -> Result<String, Error> {
    Ok(format!("{:?}; bound={BUDGET}", Parameters::new(scope.0, scope.1)?))
}

/// Every selected protocol scope, rendered for `expected/refinement.txt`.
pub(crate) fn refinement() -> Result<String, Error> {
    let mut scopes = Vec::new();
    for scope in crate::protocol_fixtures::SELECTED {
        let (concrete, feature) = worlds(scope)?;
        scopes.push((label(scope)?, refine(&concrete, &feature, &relation(), BUDGET)?));
    }
    Ok(render(&scopes))
}

type Edit = umpire_testing::refinement::Edit<protocol::World>;
/// One selected scope as exact neutral parameters.
type Scope = (u8, [Option<u8>; 4]);

/// A protocol graph whose edges are edited independently of the authored source.
fn edited(
    scope: Scope,
    step: Edit,
    relation: &Relation<protocol::World, activity::World>,
) -> Result<Report, Error> {
    let (concrete, feature) = worlds(scope)?;
    Ok(refine(&Mutant { inner: concrete, starts: None, step }, &feature, relation, BUDGET)?)
}

fn mapped(
    scope: Scope,
    relation: &Relation<protocol::World, activity::World>,
) -> Result<Report, Error> {
    edited(scope, |_, _, steps| steps, relation)
}

/// Whether some offending edge is reported for exactly this reason.
fn failed_because(report: &Report, reason: &str) -> bool {
    report.violations.iter().any(|violation| violation.ends_with(reason))
}

const ALL_TIMERS: Scope = (3, [Some(3); 4]);
const SPANNING: Scope = (3, [Some(3), Some(1), Some(3), None]);
const HEARTBEAT: Scope = (3, [None, None, None, Some(2)]);

#[test]
fn every_selected_scope_refines_with_full_state_and_observable_compatibility() -> TestResult {
    for scope in crate::protocol_fixtures::SELECTED {
        let report = mapped(scope, &relation())?;
        assert_eq!(report.verdict(), Verdict::Refined, "{scope:?}: {report}");
        assert_eq!(report.mapped, report.edges, "{scope:?}: every protocol edge maps");
        assert!(report.complete && report.stutters == 0 && report.gaps.is_empty());
    }
    Ok(())
}

#[test]
fn hidden_budget_controls_fail_even_where_public_views_match() -> TestResult {
    let mut erased = relation();
    erased.state = |state| {
        projection::feature_state(state).map(|mut image| {
            image.heartbeat_remaining = None;
            image
        })
    };
    let mut defaulted = relation();
    defaulted.state = |state| {
        projection::feature_state(state).map(|mut image| {
            image.schedule_to_close_remaining =
                image.schedule_to_close_remaining.or(Some(activity::Remaining { ticks: 3 }));
            image
        })
    };
    let mut tick_stutter = relation();
    tick_stutter.action = |action| match action {
        protocol::Action::Timer(_) => Mapping::Stutter,
        other => Mapping::Mapped((declared().action)(other)),
    };
    let mut heartbeat_stutter = relation();
    heartbeat_stutter.action = |action| match action {
        protocol::Action::Worker(protocol::Worker::Heartbeat) => Mapping::Stutter,
        other => Mapping::Mapped((declared().action)(other)),
    };
    let mut swapped_fact = relation();
    swapped_fact.fact = |fact| match fact {
        protocol::Expired::Heartbeat => activity::ExpiredDeadline::StartToClose,
        other => (declared().fact)(other),
    };
    let mut wrong_action = relation();
    wrong_action.action = |action| match action {
        protocol::Action::Worker(protocol::Worker::Fail { retryable }) => {
            Mapping::Mapped(activity::Worker::RespondFailed { retryable: !retryable }.into())
        }
        other => Mapping::Mapped((declared().action)(other)),
    };
    let mut wrong_outcome = relation();
    wrong_outcome.outcome = |_| Outcome::Accepted;
    for (name, scope, relation, reason) in [
        ("erased heartbeat budget", ALL_TIMERS, erased, "reaches a different state"),
        ("defaulted spanning budget", ALL_TIMERS, defaulted, "reaches a different state"),
        ("visible-equal Tick stutter", ALL_TIMERS, tick_stutter, "stutter changes abstract state"),
        (
            "visible-equal Heartbeat stutter",
            ALL_TIMERS,
            heartbeat_stutter,
            "stutter changes abstract state",
        ),
        ("swapped expiry fact", HEARTBEAT, swapped_fact, "emits different facts"),
        ("wrong action projection", ALL_TIMERS, wrong_action, "reaches a different state"),
        ("erased rejection outcome", ALL_TIMERS, wrong_outcome, "has a different outcome"),
    ] {
        let report = mapped(scope, &relation)?;
        assert_eq!(report.verdict(), Verdict::Violated, "{name}: {report}");
        assert!(failed_because(&report, reason), "{name}: {report}");
    }
    Ok(())
}

#[test]
#[expect(clippy::too_many_lines, reason = "one independently specified protocol edit per control")]
fn changed_timer_reset_spanning_priority_and_start_edges_fail() -> TestResult {
    let edits: [(&str, Scope, Edit, &str); 7] = [
        (
            "countdown skips heartbeat",
            ALL_TIMERS,
            |before, action, mut steps| {
                if *action == protocol::Timer::Tick.into() {
                    for step in &mut steps {
                        if step.2.lifecycle == protocol::Lifecycle::Executing {
                            step.2.heartbeat_budget = before.heartbeat_budget;
                        }
                    }
                }
                steps
            },
            "reaches a different state",
        ),
        (
            "heartbeat does not reset",
            ALL_TIMERS,
            |before, action, mut steps| {
                if *action == protocol::Worker::Heartbeat.into() {
                    for step in &mut steps {
                        step.2.heartbeat_budget = before.heartbeat_budget;
                    }
                }
                steps
            },
            "reaches a different state",
        ),
        (
            "retry replenishes spanning budget",
            SPANNING,
            |before, _, mut steps| {
                for step in &mut steps {
                    if step.2.attempt != before.attempt && step.2.execution_budget.is_some() {
                        step.2.execution_budget = Some(protocol::Remaining { ticks: 3 });
                    }
                }
                steps
            },
            "reaches a different state",
        ),
        (
            "local expiry outranks spanning expiry",
            ALL_TIMERS,
            |before, action, mut steps| {
                let one = Some(protocol::Remaining { ticks: 1 });
                if *action == protocol::Timer::Tick.into()
                    && before.execution_budget == one
                    && before.attempt_budget == one
                    && before.attempt < 3
                {
                    // Simultaneous expiry takes the attempt branch: a retry with its own cause.
                    for step in &mut steps {
                        step.1 = vec![protocol::Expired::StartToClose];
                        step.2 = protocol::Model {
                            lifecycle: protocol::Lifecycle::Queued,
                            attempt: before.attempt + 1,
                            cancellation_pending: false,
                            terminal_outcome: protocol::Terminal::None,
                            retained_failure: protocol::Failure::StartToClose,
                            queued_budget: Some(protocol::Remaining { ticks: 3 }),
                            attempt_budget: None,
                            execution_budget: Some(protocol::Remaining { ticks: 0 }),
                            heartbeat_budget: None,
                        };
                    }
                }
                steps
            },
            "emits different facts",
        ),
        (
            "retry outranks spanning expiry with its fact kept",
            ALL_TIMERS,
            |before, action, mut steps| {
                let one = Some(protocol::Remaining { ticks: 1 });
                if *action == protocol::Timer::Tick.into()
                    && before.execution_budget == one
                    && before.attempt_budget == one
                    && before.attempt < 3
                {
                    // The ScheduleToClose fact stays; only the destination reveals the retry.
                    for step in &mut steps {
                        step.2 = protocol::Model {
                            lifecycle: protocol::Lifecycle::Queued,
                            attempt: before.attempt + 1,
                            cancellation_pending: false,
                            terminal_outcome: protocol::Terminal::None,
                            retained_failure: protocol::Failure::StartToClose,
                            queued_budget: Some(protocol::Remaining { ticks: 3 }),
                            attempt_budget: None,
                            execution_budget: Some(protocol::Remaining { ticks: 0 }),
                            heartbeat_budget: None,
                        };
                    }
                }
                steps
            },
            "reaches a different state",
        ),
        (
            "cancellation acknowledgement completes",
            ALL_TIMERS,
            |_, action, mut steps| {
                if matches!(
                    action,
                    protocol::Action::Worker(protocol::Worker::AcknowledgeCancel(_))
                ) {
                    for step in &mut steps {
                        if step.2.terminal_outcome == protocol::Terminal::Canceled {
                            step.2.terminal_outcome = protocol::Terminal::Completed;
                        }
                    }
                }
                steps
            },
            "reaches a different state",
        ),
        (
            "rejection becomes acceptance",
            ALL_TIMERS,
            |_, _, mut steps| {
                for step in &mut steps {
                    step.0 = Outcome::Accepted;
                }
                steps
            },
            "has a different outcome",
        ),
    ];
    for (name, scope, edit, reason) in edits {
        let report = edited(scope, edit, &relation())?;
        assert_eq!(report.verdict(), Verdict::Violated, "{name}: {report}");
        assert!(failed_because(&report, reason), "{name}: {report}");
    }
    let (concrete, feature) = worlds(ALL_TIMERS)?;
    let started = Mutant {
        inner: concrete,
        starts: Some(|world| {
            world
                .starts()
                .into_iter()
                .map(|mut start| {
                    start.attempt = 2;
                    start
                })
                .collect()
        }),
        step: |_, _, steps| steps,
    };
    let report = refine(&started, &feature, &relation(), BUDGET)?;
    assert!(
        report.violations.iter().any(|violation| violation.starts_with("start 0:")),
        "{report}"
    );
    let cut = refine(&worlds(ALL_TIMERS)?.0, &feature, &relation(), 4)?;
    assert_eq!(cut.verdict(), Verdict::Incomplete, "{cut}");
    Ok(())
}

/// The read target of one reachable protocol state accepts the image of another exactly when
/// their public views agree outside the retained failure the feature does not represent.
#[test]
fn observable_target_commutes_with_the_full_state_map_in_both_directions() -> TestResult {
    for scope in crate::protocol_fixtures::SELECTED {
        let (concrete, _) = worlds(scope)?;
        let graph = umpire_testing::Checker::explore(&concrete, BUDGET)?;
        assert!(!graph.counts.cut);
        let states: Vec<_> = graph.states().map(|(_, state)| state.clone()).collect();
        for source in &states {
            let target = (declared().view)(source);
            let public = View::from(source);
            for candidate in &states {
                let image = (declared().state)(candidate).ok_or("reachable state has no image")?;
                let mut other = View::from(candidate);
                other.retained_failure = public.retained_failure;
                assert_eq!(
                    target.matches(&image),
                    public == other,
                    "{scope:?}: {source:?} against {candidate:?}"
                );
            }
        }
    }
    Ok(())
}

#[test]
fn rendered_report_is_deterministic_and_names_no_state_values() -> TestResult {
    let text = refinement()?;
    assert_eq!(text, refinement()?);
    assert!(!text.contains("Remaining") && !text.contains("budget:"), "{text}");
    assert_eq!(text.matches("verdict=refined").count(), crate::protocol_fixtures::SELECTED.len());
    Ok(())
}
