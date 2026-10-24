use std::cell::Cell;

use umpire_checker::Checker;
use umpire_dsl::{Claim, Machine, Progress, StepClaim, Steps};

struct World {
    ordered: bool,
    sealed: Cell<bool>,
}

impl World {
    fn observed(&self) {
        assert!(!self.sealed.get(), "verify_all re-ran model code");
    }
}

impl Machine for World {
    type State = bool;
    type Action = bool;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<bool> {
        self.observed();
        vec![false]
    }

    fn actions(&self) -> Vec<bool> {
        self.observed();
        vec![false, true]
    }

    fn internal(&self, _: &bool) -> bool {
        self.observed();
        false
    }

    fn step(&self, _: &bool, action: bool) -> Steps<bool, (), ()> {
        self.observed();
        assert!(!action, "second failure");
        panic!("first failure");
    }

    fn invariants(&self) -> Vec<Claim<Self>> {
        self.observed();
        vec![
            Claim::new("step_0_0", |world: &Self, _| {
                world.observed();
                false.into()
            }),
            Claim::new("duplicate", |world: &Self, _| {
                world.observed();
                true.into()
            }),
        ]
    }

    fn postconditions(&self) -> Vec<StepClaim<Self>> {
        self.observed();
        vec![StepClaim::new("duplicate", |world: &Self, _, _, _, (), _| {
            world.observed();
            true.into()
        })]
    }

    fn progress(&self) -> Vec<Progress<Self>> {
        self.observed();
        vec![Progress::new("duplicate")
            .whenever(|world: &Self, _| {
                world.observed();
                true.into()
            })
            .eventually(|world: &Self, _| {
                world.observed();
                true.into()
            })]
    }

    fn claim_order(&self) -> Vec<&'static str> {
        self.observed();
        if self.ordered { vec!["duplicate", "step_0_0"] } else { vec![] }
    }
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let mut text = String::new();
    for (ordered, supplemental) in [(false, false), (true, false), (true, true)] {
        text.push_str(if supplemental {
            "supplemental name collision:\n"
        } else if ordered {
            "ordered world:\n"
        } else {
            "manual default:\n"
        });
        let world = World { ordered, sealed: Cell::new(false) };
        let result = if supplemental {
            Checker::refuted(&world, Claim::new("duplicate", |world: &World, _| {
                world.observed();
                false.into()
            }))?
        } else {
            Checker::explore(&world, 1)?
        };
        world.sealed.set(true);
        let repeated = (result.verify_all()?).to_string();
        text.push_str(&result.verify_all()?.to_string());
        assert!(text.ends_with(&repeated));
    }
    Ok(umpire_testing::Files::from([("report.txt".into(), text)]))
}
