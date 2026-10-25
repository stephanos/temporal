//! Independent finite-universe closure oracle for small generated worlds.

#![forbid(unsafe_code)]

use std::collections::{BTreeSet, VecDeque};

use proptest::prelude::*;
use umpire_checker::{Checker, ProgressVerdict};
use umpire_dsl::{
    Assumption, AssumptionExt, CheckOutcome, Claim, Hole, Machine, Progress, StepClaim, Steps,
    Verdict,
};

#[derive(Clone, Debug)]
struct Rule {
    guard_var: usize,
    guard_value: u8,
    effect_var: usize,
    increments: Vec<u8>,
    internal: bool,
}

#[derive(Clone, Debug)]
struct World {
    domains: Vec<u8>,
    starts: Vec<[u8; 3]>,
    actions: Vec<Rule>,
    forbidden: [u8; 3],
    hole: [u8; 3],
    excluded: u8,
    fair: u8,
    trigger: u32,
    target: u32,
    bound: Option<u32>,
}

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
struct Action(usize);

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
struct State([u8; 3]);

impl std::fmt::Display for State {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        std::fmt::Debug::fmt(&self.0, out)
    }
}

impl std::fmt::Display for Action {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str("Action { 0: ")?;
        write!(out, "{:?}", self.0)?;
        out.write_str(" }")?;
        Ok(())
    }
}

impl Machine for World {
    type State = State;
    type Action = Action;
    type Outcome = bool;
    type Fact = usize;

    fn starts(&self) -> Vec<Self::State> {
        self.starts.iter().copied().map(State).collect()
    }
    fn actions(&self) -> Vec<Action> {
        (0..self.actions.len()).map(Action).collect()
    }
    fn internal(&self, action: &Action) -> bool {
        self.actions[action.0].internal
    }
    fn step(&self, state: &Self::State, action: Action) -> Steps<Self::State, bool, usize> {
        let rule = &self.actions[action.0];
        let mut choices = Steps::new();
        if state.0[rule.guard_var] == rule.guard_value {
            for (choice, increment) in rule.increments.iter().enumerate() {
                let mut after = *state;
                after.0[rule.effect_var] =
                    (after.0[rule.effect_var] + increment) % self.domains[rule.effect_var];
                choices.push((choice % 2 == 0, vec![choice], after));
            }
        }
        choices
    }
    fn invariants(&self) -> Vec<Claim<Self>> {
        vec![Claim::<Self>::new("not_forbidden", |world, state| {
            if state.0 == world.forbidden {
                Verdict::Fails
            } else if state.0 == world.hole {
                Verdict::Unknown(Hole { reason: "random hole" })
            } else {
                Verdict::Holds
            }
        })]
    }
    fn postconditions(&self) -> Vec<StepClaim<Self>> {
        vec![
            StepClaim::<Self>::new("nondecreasing", |_, before, after, _, _, _| {
                (after.0[0] >= before.0[0]).into()
            }),
            StepClaim::<Self>::after("zero_changes", Action(0), |_, before, after, _, _, _| {
                (before != after).into()
            }),
            StepClaim::<Self>::new("choice_facts", |_, _, _, _, outcome, facts| {
                (facts.len() == 1 && *outcome == (facts[0] % 2 == 0)).into()
            }),
        ]
    }
    fn progress(&self) -> Vec<Progress<Self>> {
        let claim = Progress::<Self>::new("eventually_target")
            .whenever(|world, state| member(world.trigger, state.0).into())
            .eventually(|world, state| member(world.target, state.0).into())
            .assuming(
                Assumption::fair(
                    &(0..self.actions.len())
                        .filter(|action| self.fair & (1 << action) != 0)
                        .map(Action)
                        .collect::<Vec<_>>(),
                )
                .without(
                    (0..self.actions.len())
                        .filter(|action| self.excluded & (1 << action) != 0)
                        .map(Action)
                        .collect::<Vec<_>>(),
                ),
            );
        vec![match self.bound {
            Some(bound) => claim.within_steps(bound),
            None => claim,
        }]
    }
}

fn member(set: u32, state: [u8; 3]) -> bool {
    set & (1 << (state[0] + 3 * state[1] + 9 * state[2])) != 0
}

fn worlds() -> impl Strategy<Value = World> {
    (
        prop::collection::vec(2u8..=3, 2..=3),
        prop::collection::vec(any::<[u8; 3]>(), 1..=3),
        prop::collection::vec(
            (
                0usize..3,
                any::<u8>(),
                0usize..3,
                prop::collection::vec(0u8..=2, 1..=2),
                any::<bool>(),
            ),
            2..=4,
        ),
        any::<[u8; 3]>(),
        any::<[u8; 3]>(),
        any::<u8>(),
        any::<u8>(),
        0u32..(1 << 27),
        0u32..(1 << 27),
        prop::option::of(1u32..=5),
    )
        .prop_map(
            |(domains, starts, rules, forbidden, hole, excluded, fair, trigger, target, bound)| {
                let normalize = |mut values: [u8; 3]| {
                    for (index, value) in values.iter_mut().enumerate() {
                        *value = domains.get(index).map_or(0, |d| *value % d);
                    }
                    values
                };
                let catalog = (1u8 << rules.len()) - 1;
                let excluded = if excluded & catalog == catalog { excluded & !1 } else { excluded };
                World {
                    starts: starts.into_iter().map(normalize).collect(),
                    actions: rules
                        .into_iter()
                        .map(|(guard_var, guard_value, effect_var, increments, internal)| {
                            let guard_var = guard_var % domains.len();
                            Rule {
                                guard_var,
                                guard_value: guard_value % domains[guard_var],
                                effect_var: effect_var % domains.len(),
                                increments,
                                internal,
                            }
                        })
                        .collect(),
                    forbidden: normalize(forbidden),
                    hole: normalize(hole),
                    excluded,
                    fair,
                    trigger,
                    target,
                    bound,
                    domains,
                }
            },
        )
}

type Edge = ([u8; 3], usize, usize, [u8; 3]);

#[derive(Debug)]
struct Reference {
    reached: BTreeSet<[u8; 3]>,
    outcomes: Vec<(&'static str, CheckOutcome)>,
    edges: Vec<Edge>,
}

fn progress_fails(world: &World, reached: &BTreeSet<[u8; 3]>, edges: &[Edge]) -> bool {
    let scoped: Vec<_> =
        edges.iter().filter(|(_, action, _, _)| world.excluded & (1 << action) == 0).collect();
    let mut pending: BTreeSet<_> = reached
        .iter()
        .filter(|state| member(world.trigger, **state) && !member(world.target, **state))
        .copied()
        .collect();
    loop {
        let additions: Vec<_> = scoped
            .iter()
            .filter(|(before, _, _, after)| {
                pending.contains(before) && !member(world.target, *after)
            })
            .map(|(_, _, _, after)| *after)
            .collect();
        let old_len = pending.len();
        pending.extend(additions);
        if pending.len() == old_len {
            break;
        }
    }
    let required = world.fair & !world.excluded & ((1u8 << world.actions.len()) - 1);
    if let Some(bound) = world.bound {
        if pending.iter().any(|state| !scoped.iter().any(|(before, _, _, _)| before == state)) {
            return true;
        }
        let mut frontier: BTreeSet<_> = reached
            .iter()
            .filter(|state| member(world.trigger, **state) && !member(world.target, **state))
            .copied()
            .collect();
        for _ in 0..bound {
            frontier = scoped
                .iter()
                .filter(|(before, _, _, after)| {
                    frontier.contains(before) && !member(world.target, *after)
                })
                .map(|(_, _, _, after)| *after)
                .collect();
        }
        return !frontier.is_empty();
    }
    let disabled = |state: &[u8; 3]| {
        scoped
            .iter()
            .filter(|(before, _, _, _)| before == state)
            .fold(required, |mask, (_, action, _, _)| mask & !(1 << action))
    };
    for entry in pending {
        if !scoped.iter().any(|(before, _, _, _)| *before == entry) {
            return true;
        }
        let initial = (entry, disabled(&entry));
        let mut seen = BTreeSet::from([initial]);
        let mut queue = VecDeque::from([initial]);
        while let Some((state, mask)) = queue.pop_front() {
            for (_, action, _, after) in scoped
                .iter()
                .filter(|(before, _, _, after)| *before == state && !member(world.target, *after))
            {
                let next_mask = mask | disabled(after) | (required & (1 << action));
                if *after == entry && next_mask == required {
                    return true;
                }
                if seen.insert((*after, next_mask)) {
                    queue.push_back((*after, next_mask));
                }
            }
        }
    }
    false
}

fn reference(world: &World) -> Reference {
    let mut edges = Vec::new();
    for x in 0..world.domains[0] {
        for y in 0..world.domains[1] {
            for z in 0..world.domains.get(2).copied().unwrap_or(1) {
                let before = [x, y, z];
                for (action, rule) in world.actions.iter().enumerate() {
                    if before[rule.guard_var] != rule.guard_value {
                        continue;
                    }
                    for (choice, increment) in rule.increments.iter().enumerate() {
                        let mut after = before;
                        for _ in 0..*increment {
                            after[rule.effect_var] =
                                if after[rule.effect_var] + 1 == world.domains[rule.effect_var] {
                                    0
                                } else {
                                    after[rule.effect_var] + 1
                                };
                        }
                        edges.push((before, action, choice, after));
                    }
                }
            }
        }
    }
    let mut reached: BTreeSet<_> = world.starts.iter().copied().collect();
    loop {
        let additions: Vec<_> = edges
            .iter()
            .filter(|(before, _, _, _)| reached.contains(before))
            .map(|(_, _, _, after)| *after)
            .collect();
        let old_len = reached.len();
        reached.extend(additions);
        if old_len == reached.len() {
            break;
        }
    }
    edges.retain(|(before, _, _, _)| reached.contains(before));
    let outcome = |fails| if fails { CheckOutcome::Refuted } else { CheckOutcome::Verified };
    Reference {
        outcomes: vec![
            (
                "not_forbidden",
                if reached.contains(&world.forbidden) {
                    CheckOutcome::Refuted
                } else if reached.contains(&world.hole) {
                    CheckOutcome::Unknown
                } else {
                    CheckOutcome::Verified
                },
            ),
            (
                "nondecreasing",
                outcome(edges.iter().any(|(before, _, _, after)| after[0] < before[0])),
            ),
            (
                "zero_changes",
                outcome(
                    edges.iter().any(|(before, action, _, after)| *action == 0 && before == after),
                ),
            ),
            ("choice_facts", CheckOutcome::Verified),
            ("eventually_target", outcome(progress_fails(world, &reached, &edges))),
        ],
        reached,
        edges,
    }
}

proptest! {
    #![proptest_config(ProptestConfig { cases: 256, .. ProptestConfig::default() })]
    #[test]
    fn brute_force_oracle(world in worlds()) {
        let expected = reference(&world);
        let actual = Checker::explore(&world, 27)?;
        prop_assert_eq!(actual.states().map(|(_, s)| s.0).collect::<BTreeSet<_>>(), expected.reached);
        let report = actual.verify_all()?;
        prop_assert_eq!(report.claims.iter().map(|c| (c.name.as_str(), c.outcome)).collect::<Vec<_>>(), expected.outcomes);
        prop_assert_eq!(actual.counts.emitted, expected.edges.len());
        prop_assert_eq!(actual.counts.evaluated, actual.counts.reached * world.actions.len());
        prop_assert!(!actual.counts.cut);
        for (id, state) in actual.states() {
            let rows: Vec<_> = actual.rows(id)?.map(|edge| (edge.action.0, edge.choice, edge.emitted.2.0)).collect();
            let wanted: Vec<_> = expected.edges.iter().filter(|(before, _, _, _)| *before == state.0)
                .map(|(_, action, choice, after)| (*action, *choice, *after)).collect();
            prop_assert_eq!(rows, wanted);
        }
        for claim in report.claims {
            if claim.name == "eventually_target" {
                prop_assert_eq!(claim.path.is_some(), claim.outcome == CheckOutcome::Refuted);
            }
            if let Some(path) = claim.path {
                prop_assert!(world.starts.contains(&path.start.0));
                let mut before = path.start.0;
                for step in &path.steps {
                    prop_assert!(expected.edges.contains(&(before, step.action.0, step.choice, step.after.0)));
                    before = step.after.0;
                }
                if claim.name == "not_forbidden" { prop_assert_eq!(before, world.forbidden); }
                if claim.name == "eventually_target" {
                    prop_assert!(path.failure.is_none());
                    let states: Vec<_> = std::iter::once(path.start.0)
                        .chain(path.steps.iter().map(|step| step.after.0)).collect();
                    let suffix_end = path.cycle_start.unwrap_or(path.steps.len());
                    prop_assert!(suffix_end <= path.steps.len());
                    prop_assert!((0..=suffix_end).any(|trigger| {
                        member(world.trigger, states[trigger])
                            && states[trigger..].iter().all(|state| !member(world.target, *state))
                            && path.steps[trigger..].iter()
                                .all(|step| world.excluded & (1 << step.action.0) == 0)
                    }), "progress witness contains a trigger and scoped target-avoiding suffix");
                    if let Some(cycle) = path.cycle_start {
                        prop_assert!(cycle < path.steps.len());
                        prop_assert_eq!(before, states[cycle]);
                        for action in 0..world.actions.len() {
                            if matches!(claim.verdict, Some(ProgressVerdict::Cycle))
                                && world.fair & (1 << action) != 0
                                && world.excluded & (1 << action) == 0
                            {
                                prop_assert!(path.steps[cycle..].iter()
                                    .any(|step| step.action.0 == action)
                                    || states[cycle..path.steps.len()].iter().any(|state| {
                                        !expected.edges.iter().any(|(source, candidate, _, _)| {
                                            source == state && *candidate == action
                                        })
                                }), "cycle satisfies each explicitly fair action's weak fairness");
                            }
                        }
                    } else {
                        let dead_end = !expected.edges.iter().any(|(source, action, _, _)| {
                            *source == before && world.excluded & (1 << action) == 0
                        });
                        let missed_bound = world.bound.is_some_and(|bound| {
                            (0..states.len()).any(|trigger| {
                                member(world.trigger, states[trigger])
                                    && states[trigger..].iter().all(|state| !member(world.target, *state))
                                    && path.steps.len() - trigger >= bound as usize
                            })
                        });
                        prop_assert!(dead_end || missed_bound, "finite progress witness ends at a scoped dead end or misses its bound");
                    }
                }
            }
        }
    }
}
