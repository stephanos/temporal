//! Independent inventory, observable keys and finite filler capability controls.
#![forbid(unsafe_code)]

use umpire_dsl::{Machine, Outcome};
use umpire_runtime::{AnyMessage as _, Evidence};
use umpire_temporal::{
    activity::{self, Client, Deadline, Phase, Ticks, Worker},
    policy::{
        AssemblyLimits, Choice, Domain, History, Inventory, Manifest, OperationalRole, TaskKey,
        operational_role,
    },
};
use umpire_temporal_api::temporal::api::workflowservice::v1::{
    RecordActivityTaskHeartbeatRequest, RespondActivityTaskCanceledRequest,
};

#[allow(
    dead_code,
    reason = "shared fixture module also supplies Describe fixtures to other targets"
)]
mod common;

fn edge(
    world: &activity::World,
    source: &activity::Model,
    action: impl Into<activity::Action>,
) -> Evidence<activity::World> {
    let action = action.into();
    let (outcome, facts, target) = world
        .checked_step(source, action.clone())
        .unwrap_or_else(|error| panic!("{error}"))
        .into_iter()
        .next()
        .unwrap_or_else(|| panic!("disabled action"));
    Evidence { source: source.clone(), action, choice: 0, outcome, facts, target }
}

#[test]
fn inventory_keeps_heartbeat_stays_stale_transport_and_no_empty_arm_edges() {
    let world = activity::World::new(3, None, None, None, None);
    let inventory = Inventory::derive(&world, 32, 20_000).unwrap_or_else(|e| panic!("{e:?}"));
    assert!(inventory.edges.iter().any(|edge| {
        operational_role(edge) == OperationalRole::Response(Choice::Heartbeat)
            && edge.source == edge.target
            && edge.outcome == Outcome::Accepted
    }));
    assert!(inventory.edges.iter().any(|edge| {
        operational_role(edge) == OperationalRole::StalePoll
            && edge.outcome == Outcome::Rejected(umpire_dsl::Rejection::FailedPrecondition)
    }));
    assert!(inventory.edges.iter().any(|edge| {
        operational_role(edge) == OperationalRole::StaleAcknowledgment
            && edge.outcome == Outcome::Rejected(umpire_dsl::Rejection::NotFound)
    }));
    assert!(!inventory.edges.iter().any(|edge| {
        matches!(&edge.action, activity::Action::Worker(Worker::Poll(expected))
            if *expected == edge.source.attempt && edge.source.phase == Phase::Started)
    }));
    assert!(Inventory::derive(&world, 0, 20_000).is_err());
    assert!(Inventory::derive(&world, 32, 1).is_err());
}

#[test]
fn full_hidden_heartbeat_budget_changes_remain_in_inventory() {
    let world = activity::World::new(1, None, None, None, Some(Ticks(3)));
    let inventory = Inventory::derive(&world, 32, 20_000).unwrap_or_else(|e| panic!("{e:?}"));
    assert!(inventory.edges.iter().any(|edge| {
        operational_role(edge) == OperationalRole::Response(Choice::Heartbeat)
            && edge.source.phase == edge.target.phase
            && edge.source.heartbeat_remaining != edge.target.heartbeat_remaining
    }));
}

#[test]
fn task_history_uses_actual_bits_and_resets_at_attempt_delivery() {
    let world = activity::World::new(3, None, None, None, None);
    let start = &world.starts()[0];
    let delivered = edge(&world, start, Worker::Poll(1));
    let history = History::default().advance(&delivered).unwrap_or_else(|e| panic!("{e:?}"));
    let heartbeat = edge(&world, &delivered.target, Worker::Heartbeat);
    let history = history.advance(&heartbeat).unwrap_or_else(|e| panic!("{e:?}"));
    assert_eq!(
        history.task.as_ref().map(|key| key.heartbeat_history.as_slice()),
        Some(&[false][..])
    );
    let cancel = edge(&world, &heartbeat.target, Client::Cancel);
    let history = history.advance(&cancel).unwrap_or_else(|e| panic!("{e:?}"));
    let notified = edge(&world, &cancel.target, Worker::Heartbeat);
    let notified_history = history.advance(&notified).unwrap_or_else(|e| panic!("{e:?}"));
    assert_eq!(
        notified_history.task.as_ref().map(|key| key.heartbeat_history.as_slice()),
        Some(&[false, true][..])
    );

    let retry = edge(&world, &heartbeat.target, Worker::RespondFailed { retryable: true });
    let history = history.advance(&retry).unwrap_or_else(|e| panic!("{e:?}"));
    assert_eq!(history.task, None);
    let delivered = edge(&world, &retry.target, Worker::Poll(2));
    let history = history.advance(&delivered).unwrap_or_else(|e| panic!("{e:?}"));
    assert_eq!(history.task, Some(TaskKey { attempt: 2, heartbeat_history: vec![] }));
}

#[test]
fn policy_horizon_and_sdk_ack_admission_use_observed_history() {
    let domain = Domain { heartbeats_per_task: 2 };
    for history in [vec![], vec![false], vec![false, false]] {
        let key = TaskKey { attempt: 1, heartbeat_history: history };
        assert!(!domain.choices(&key).contains(&Choice::Canceled));
        assert_eq!(
            domain.choices(&key).contains(&Choice::Heartbeat),
            key.heartbeat_history.len() < 2
        );
    }
    let key = TaskKey { attempt: 1, heartbeat_history: vec![false, true] };
    assert!(domain.choices(&key).contains(&Choice::Canceled));
    assert!(!domain.choices(&key).contains(&Choice::Heartbeat));
}

#[test]
fn finite_heartbeat_and_canceled_rows_use_registry_fill() {
    let world = activity::World::new(3, None, None, None, None);
    let start = common::start(3)
        .request
        .unwrap_or_else(|| panic!("request"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    let mut key = TaskKey { attempt: 1, heartbeat_history: vec![] };
    let heartbeat: RecordActivityTaskHeartbeatRequest = Choice::Heartbeat
        .fill(&world, &start, &key)
        .unwrap_or_else(|e| panic!("{e:?}"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    assert!(heartbeat.details.is_none());
    assert!(Choice::Canceled.fill(&world, &start, &key).is_err());
    key.heartbeat_history.push(true);
    let canceled: RespondActivityTaskCanceledRequest = Choice::Canceled
        .fill(&world, &start, &key)
        .unwrap_or_else(|e| panic!("{e:?}"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    assert!(canceled.details.is_none());
}

#[test]
fn deadline_does_not_fabricate_worker_observations() {
    let world = activity::World::new(1, None, Some(Ticks(3)), None, Some(Ticks(3)));
    let delivered = edge(&world, &world.starts()[0], Worker::Poll(1));
    let history = History::default().advance(&delivered).unwrap_or_else(|e| panic!("{e:?}"));
    let tick = edge(&world, &delivered.target, Deadline::Tick);
    assert_eq!(history.advance(&tick).unwrap_or_else(|e| panic!("{e:?}")), history);
}

fn limits() -> umpire_runtime::policy::Limits {
    umpire_runtime::policy::Limits {
        depth: 32,
        work: 1_000_000,
        assignments: 1000,
        catalog: umpire_dsl::CATALOG_LIMIT,
    }
}

#[test]
fn derived_policy_families_have_independent_complete_edge_coverage() {
    let world = activity::World::new(1, None, None, None, None);
    let enumeration =
        umpire_runtime::policy::enumerate(&world, &Domain { heartbeats_per_task: 1 }, limits())
            .unwrap_or_else(|e| panic!("{e}"));
    // Three terminal roots and every false/true continuation of a heartbeat root.
    assert_eq!(enumeration.policies().len(), 15);
    let coverage =
        enumeration.coverage(enumeration.policies(), 1_000_000).unwrap_or_else(|e| panic!("{e}"));
    assert!(coverage.complete(), "{:?}", coverage.missing);
    assert!(enumeration.unrepresentable().iter().any(|finding| {
        finding.key.as_ref().is_some_and(|key| !key.heartbeat_history.contains(&true))
            && finding.reason.contains("true heartbeat")
    }));
    let terminal_only: Vec<_> = enumeration
        .policies()
        .iter()
        .filter(|policy| policy.rows.iter().all(|(_, choice)| *choice != Choice::Heartbeat))
        .cloned()
        .collect();
    assert!(
        !enumeration
            .coverage(&terminal_only, 1_000_000)
            .unwrap_or_else(|e| panic!("{e}"))
            .complete()
    );
    let refused = Manifest::derive(&world, Domain { heartbeats_per_task: 0 }, limits())
        .err()
        .unwrap_or_else(|| panic!("missing heartbeat coverage must refuse"));
    for diagnostic in ["source=", "action=", "choice=", "outcome=", "facts=", "target="] {
        assert!(refused.message.contains(diagnostic), "{}", refused.message);
    }
}

#[test]
fn a_declared_terminal_horizon_does_not_erase_heartbeat_obligations() {
    let world = activity::World::new(1, None, None, None, None);
    let enumeration =
        umpire_runtime::policy::enumerate(&world, &Domain { heartbeats_per_task: 0 }, limits())
            .unwrap_or_else(|e| panic!("{e}"));
    assert!(
        !enumeration
            .coverage(enumeration.policies(), 1_000_000)
            .unwrap_or_else(|e| panic!("{e}"))
            .complete()
    );
}

#[test]
fn finite_assignment_exhaustion_returns_no_partial_enumeration() {
    let world = activity::World::new(1, None, None, None, None);
    let mut limits = limits();
    limits.assignments = 1;
    assert!(matches!(
        umpire_runtime::policy::enumerate(&world, &Domain { heartbeats_per_task: 1 }, limits,),
        Err(umpire_runtime::policy::Error::Assignments { .. })
    ));
}

#[test]
fn manifest_names_resolve_exact_keys_and_keep_both_actual_heartbeat_bits() {
    let world = activity::World::new(1, None, None, None, None);
    let manifest = Manifest::derive(&world, Domain { heartbeats_per_task: 1 }, limits())
        .unwrap_or_else(|e| panic!("{e:?}"));
    let mut names = std::collections::BTreeSet::new();
    for policy in manifest.policies() {
        assert!(names.insert(policy.name()));
        assert!(policy.name().starts_with("activity-v1_"));
        for (key, choice) in policy.rows() {
            assert_eq!(manifest.resolve(policy.name(), key), Some(*choice));
            if *choice == Choice::Heartbeat {
                assert_eq!(policy.successors().get(key), Some(&vec![false, true]));
            }
        }
    }
    let enumeration = manifest.enumeration();
    let mut broken = enumeration
        .policies()
        .iter()
        .find(|policy| policy.rows.iter().any(|(_, choice)| *choice == Choice::Heartbeat))
        .unwrap_or_else(|| panic!("heartbeat policy"))
        .clone();
    broken.rows.retain(|(key, _)| key.heartbeat_history != [false]);
    assert!(enumeration.execution(&broken, 1_000_000).is_err());
}

#[test]
fn blocked_roles_name_the_actual_request_and_only_stale_ack_has_no_request() {
    use umpire_runtime::policy::{Adapter, Role};
    let world = activity::World::new(2, None, None, None, None);
    let delivered = edge(&world, &world.starts()[0], Worker::Poll(1));
    let history = History::default().advance(&delivered).unwrap_or_else(|e| panic!("{e:?}"));
    let domain = Domain { heartbeats_per_task: 0 };
    let heartbeat = edge(&world, &delivered.target, Worker::Heartbeat);
    assert!(matches!(domain.role(&history, &heartbeat),
        Ok(Role::OutsideHorizon(choices)) if choices == [Choice::Heartbeat]));
    let canceled = edge(&world, &delivered.target, Client::Cancel);
    let acknowledgment = edge(&world, &canceled.target, Worker::RespondCanceled(1));
    assert!(matches!(domain.role(&history, &acknowledgment),
        Ok(Role::Unrepresentable { choices, .. }) if choices == [Choice::Canceled]));
    assert!(domain.required(&acknowledgment));
    let stale = edge(&world, &delivered.target, Worker::RespondCanceled(2));
    assert!(matches!(domain.role(&history, &stale),
        Ok(Role::Unrepresentable { choices, .. }) if choices.is_empty()));
    assert!(!domain.required(&stale));
}

#[test]
fn isolated_wire_assembly_keeps_all_canonical_rows_and_independent_domains() {
    use umpire_temporal_api::umpire::temporal::v1::{ActivityTaskKey, ActivityTaskSuccessors};
    let world = activity::World::new(1, None, None, None, None);
    let manifest = Manifest::derive(&world, Domain { heartbeats_per_task: 1 }, limits())
        .unwrap_or_else(|e| panic!("{e:?}"));
    let start = common::start(1)
        .request
        .unwrap_or_else(|| panic!("request"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    let program = manifest
        .wire_program(&start, AssemblyLimits { rows: 39, continuations: 12 })
        .unwrap_or_else(|e| panic!("{e:?}"));
    // Three terminal roots, plus twelve heartbeat roots with two terminal children each.
    assert_eq!(program.rules.len(), 39);
    assert_eq!(program.continuation_domains.len(), 12);
    assert_eq!(program.namespace, "model-slice");
    assert_eq!(program.task_queue, "queue");
    let mut keys = std::collections::BTreeSet::new();
    for rule in &program.rules {
        assert_eq!(rule.respond.len(), 1);
        let key: ActivityTaskKey = rule
            .key
            .as_ref()
            .unwrap_or_else(|| panic!("key"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        assert!(keys.insert((key.scenario.clone(), key.attempt, key.heartbeat_history.clone())));
        let choice = manifest
            .resolve(
                &key.scenario,
                &TaskKey {
                    attempt: u8::try_from(key.attempt).unwrap_or_else(|e| panic!("{e}")),
                    heartbeat_history: key.heartbeat_history.clone(),
                },
            )
            .unwrap_or_else(|| panic!("resolved row"));
        if choice == Choice::Canceled {
            assert!(key.heartbeat_history.contains(&true));
            let canceled: RespondActivityTaskCanceledRequest =
                rule.respond[0].unpack().unwrap_or_else(|e| panic!("{e:?}"));
            assert!(canceled.details.is_none());
        }
        if choice == Choice::Heartbeat {
            let heartbeat: RecordActivityTaskHeartbeatRequest =
                rule.respond[0].unpack().unwrap_or_else(|e| panic!("{e:?}"));
            assert!(heartbeat.details.is_none());
        }
    }
    for packed in &program.continuation_domains {
        let domain: ActivityTaskSuccessors = packed.unpack().unwrap_or_else(|e| panic!("{e:?}"));
        assert_eq!(domain.cancel_requested, [false, true]);
        let key = domain.key.unwrap_or_else(|| panic!("domain key"));
        for bit in domain.cancel_requested {
            let mut child = key.heartbeat_history.clone();
            child.push(bit);
            assert!(keys.contains(&(key.scenario.clone(), key.attempt, child)));
        }
    }
    assert!(manifest.wire_program(&start, AssemblyLimits { rows: 38, continuations: 12 }).is_err());
    assert!(manifest.wire_program(&start, AssemblyLimits { rows: 39, continuations: 11 }).is_err());
    let crossed = common::start(2)
        .request
        .unwrap_or_else(|| panic!("request"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    assert!(
        manifest.wire_program(&crossed, AssemblyLimits { rows: 100, continuations: 100 }).is_err()
    );
}

#[test]
#[expect(clippy::print_stderr, reason = "retain exact finite matrix counts in validation evidence")]
fn retry_and_timer_world_matrix_keeps_independent_required_coverage() {
    for maximum in 1..=3 {
        for timer in 0..=4 {
            // Multi-attempt no-timer Worlds expose repeated retries. Each timer is also
            // checked independently at one attempt so hidden budget states remain explicit.
            if maximum > 1 && timer != 0 {
                continue;
            }
            let clock = |selected| (timer == selected).then_some(Ticks(2));
            let world = activity::World::new(maximum, clock(1), clock(2), clock(3), clock(4));
            let limits = umpire_runtime::policy::Limits {
                depth: 32,
                work: 300_000_000,
                assignments: 150_000,
                catalog: umpire_dsl::CATALOG_LIMIT,
            };
            let manifest = Manifest::derive(&world, Domain { heartbeats_per_task: 1 }, limits)
                .unwrap_or_else(|e| panic!("maximum={maximum} timer={timer}: {e:?}"));
            let enumeration = manifest.enumeration();
            let coverage = enumeration
                .coverage(enumeration.policies(), limits.work)
                .unwrap_or_else(|e| panic!("{e}"));
            assert!(coverage.complete());
            let aliases = manifest.task_reset_aliases().unwrap_or_else(|e| panic!("{e:?}"));
            assert_eq!(aliases.len(), usize::from(maximum - 1));
            for alias in aliases {
                assert_ne!(alias.prior_histories[0], alias.prior_histories[1]);
                assert_eq!(alias.next_key.heartbeat_history, []);
                assert_ne!(alias.incompatible_choices[0], alias.incompatible_choices[1]);
                let left = alias.paths[0].last().unwrap_or_else(|| panic!("left alias path"));
                let right = alias.paths[1].last().unwrap_or_else(|| panic!("right alias path"));
                assert_eq!(left.target, right.target);
            }
            if maximum == 3 {
                assert!(manifest.policies().iter().any(|policy| {
                    [1, 2].into_iter().all(|attempt| {
                        policy.rows().contains(&(
                            TaskKey { attempt, heartbeat_history: vec![] },
                            Choice::Retry,
                        ))
                    }) && policy.rows().contains(&(
                        TaskKey { attempt: 3, heartbeat_history: vec![] },
                        Choice::Complete,
                    ))
                }));
            }
            eprintln!(
                "policy-matrix maximum={maximum} timer={timer} horizon=1 inventory={} required={} assignments={} canonical={} conflicts={} unrepresentable={} coverage_missing={}",
                enumeration.inventory().len(),
                enumeration.inventory().iter().filter(|edge| edge.required).count(),
                enumeration.assignments(),
                enumeration.policies().len(),
                enumeration.conflicts().len(),
                enumeration.unrepresentable().len(),
                coverage.missing.len()
            );
        }
    }
}
