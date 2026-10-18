//! Independent finite-universe closure oracle for small generated worlds.

use std::collections::BTreeSet;

use framework_checker::Checker;
use framework_dsl::{Claim, Hole, Machine, Outcome, StepClaim, Steps, Verdict};
use proptest::prelude::*;

#[derive(Clone, Debug)]
struct Rule {
    guard_var: usize,
    guard_value: u8,
    effect_var: usize,
    increments: Vec<u8>,
}

#[derive(Clone, Debug)]
struct World {
    domains: Vec<u8>,
    starts: Vec<[u8; 3]>,
    actions: Vec<Rule>,
    forbidden: [u8; 3],
    hole: [u8; 3],
}

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
struct Action(usize);

impl Machine for World {
    type State = [u8; 3];
    type Action = Action;
    type Outcome = bool;
    type Fact = usize;

    fn starts(&self) -> Vec<Self::State> {
        self.starts.clone()
    }
    fn actions(&self) -> Vec<Action> {
        (0..self.actions.len()).map(Action).collect()
    }
    fn internal(&self, _: &Action) -> bool {
        false
    }
    fn step(&self, state: &Self::State, action: Action) -> Steps<Self::State, bool, usize> {
        let rule = &self.actions[action.0];
        let mut choices = Steps::new();
        if state[rule.guard_var] == rule.guard_value {
            for (choice, increment) in rule.increments.iter().enumerate() {
                let mut after = *state;
                after[rule.effect_var] =
                    (after[rule.effect_var] + increment) % self.domains[rule.effect_var];
                choices.push((choice % 2 == 0, vec![choice], after));
            }
        }
        choices
    }
    fn invariants(&self) -> Vec<Claim<Self>> {
        vec![Claim::<Self>::new("not_forbidden", |world, state| {
            if state == &world.forbidden {
                Verdict::Fails
            } else if state == &world.hole {
                Verdict::Unknown(Hole { reason: "random hole" })
            } else {
                Verdict::Holds
            }
        })]
    }
    fn postconditions(&self) -> Vec<StepClaim<Self>> {
        vec![
            StepClaim::<Self>::new("nondecreasing", |_, before, after, _, _, _| {
                (after[0] >= before[0]).into()
            }),
            StepClaim::<Self>::after("zero_changes", Action(0), |_, before, after, _, _, _| {
                (before != after).into()
            }),
            StepClaim::<Self>::new("choice_facts", |_, _, _, _, outcome, facts| {
                (facts.len() == 1 && *outcome == (facts[0] % 2 == 0)).into()
            }),
        ]
    }
}

fn worlds() -> impl Strategy<Value = World> {
    (
        prop::collection::vec(2u8..=3, 2..=3),
        prop::collection::vec(any::<[u8; 3]>(), 1..=3),
        prop::collection::vec(
            (0usize..3, any::<u8>(), 0usize..3, prop::collection::vec(0u8..=2, 1..=2)),
            2..=4,
        ),
        any::<[u8; 3]>(),
        any::<[u8; 3]>(),
    )
        .prop_map(|(domains, starts, rules, forbidden, hole)| {
            let normalize = |mut values: [u8; 3]| {
                for (index, value) in values.iter_mut().enumerate() {
                    *value = domains.get(index).map_or(0, |d| *value % d);
                }
                values
            };
            World {
                starts: starts.into_iter().map(normalize).collect(),
                actions: rules
                    .into_iter()
                    .map(|(guard_var, guard_value, effect_var, increments)| {
                        let guard_var = guard_var % domains.len();
                        Rule {
                            guard_var,
                            guard_value: guard_value % domains[guard_var],
                            effect_var: effect_var % domains.len(),
                            increments,
                        }
                    })
                    .collect(),
                forbidden: normalize(forbidden),
                hole: normalize(hole),
                domains,
            }
        })
}

#[derive(Debug)]
struct Reference {
    reached: BTreeSet<[u8; 3]>,
    outcomes: Vec<(&'static str, Outcome)>,
    edges: Vec<([u8; 3], usize, usize, [u8; 3])>,
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
    let outcome = |fails| if fails { Outcome::Refuted } else { Outcome::Verified };
    Reference {
        outcomes: vec![
            (
                "not_forbidden",
                if reached.contains(&world.forbidden) {
                    Outcome::Refuted
                } else if reached.contains(&world.hole) {
                    Outcome::Unknown
                } else {
                    Outcome::Verified
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
            ("choice_facts", Outcome::Verified),
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
        let actual = Checker::explore(&world, 27);
        prop_assert_eq!(actual.states().map(|(_, s)| *s).collect::<BTreeSet<_>>(), expected.reached);
        let report = actual.verify_all();
        prop_assert_eq!(report.claims.iter().map(|c| (c.name.as_str(), c.outcome)).collect::<Vec<_>>(), expected.outcomes);
        prop_assert_eq!(actual.counts.emitted, expected.edges.len());
        prop_assert_eq!(actual.counts.evaluated, actual.counts.reached * world.actions.len());
        prop_assert!(!actual.counts.cut);
        for (id, state) in actual.states() {
            let rows: Vec<_> = actual.rows(id).map(|edge| (edge.action.0, edge.choice, edge.emitted.2)).collect();
            let wanted: Vec<_> = expected.edges.iter().filter(|(before, _, _, _)| before == state)
                .map(|(_, action, choice, after)| (*action, *choice, *after)).collect();
            prop_assert_eq!(rows, wanted);
        }
        for claim in report.claims {
            if let Some(path) = claim.path {
                prop_assert!(world.starts.contains(&path.start));
                let mut before = path.start;
                for step in &path.steps {
                    prop_assert!(expected.edges.contains(&(before, step.action.0, step.choice, step.after)));
                    before = step.after;
                }
                if claim.name == "not_forbidden" { prop_assert_eq!(before, world.forbidden); }
            }
        }
    }
}
