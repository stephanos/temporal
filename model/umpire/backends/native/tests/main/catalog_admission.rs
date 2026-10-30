//! Checked catalog admission happens before any Model step, under the selected limit.

#![forbid(unsafe_code)]

use std::sync::atomic::{AtomicUsize, Ordering};

use umpire_checker::{Checker, Refusal};
use umpire_dsl::{CheckOutcome, Machine};

static STEPS: AtomicUsize = AtomicUsize::new(0);

fn stepped() {
    STEPS.fetch_add(1, Ordering::SeqCst);
}

#[umpire_dsl::machine]
mod over {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub(super) struct Model {
        #[starts(0..=1)]
        pub(super) last: u16,
    }

    #[derive(Action, Copy)]
    pub(super) enum User {
        At(#[inputs(0..=4096)] u16),
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            super::stepped();
            match action {
                User::At(value) if value == self.last + 1 => self.last = value,
                User::At(_) => {}
            }
        }
    }
}

#[umpire_dsl::machine]
mod admitted {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub(super) struct Model {
        pub(super) last: u16,
    }

    #[derive(Action, Copy)]
    pub(super) enum User {
        At(#[inputs(0..=4095)] u16),
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::At(value) if value == 1 => self.last = value,
                User::At(_) => {}
            }
        }
    }
}

#[umpire_dsl::machine]
mod product {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub(super) struct Model {
        seen: bool,
    }

    #[derive(Action, Copy)]
    pub(super) enum User {
        At(#[inputs(0..=63)] u8, #[inputs(0..=64)] u8),
        Huge(#[inputs(0..=u64::MAX)] u64, #[inputs(0..=u64::MAX)] u64),
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) {
            super::stepped();
            self.seen = true;
        }
    }
}

#[umpire_dsl::machine]
mod multiplied {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub(super) struct Model {
        seen: bool,
    }

    #[derive(Action, Copy)]
    pub(super) enum User {
        At(#[inputs(0..=63)] u8, #[inputs(0..=64)] u8),
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) {
            super::stepped();
            self.seen = true;
        }
    }
}

#[umpire_dsl::machine]
mod emptied {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub(super) struct Model {
        pub(super) done: bool,
    }

    #[derive(Action, Copy)]
    pub(super) enum User {
        Never(#[inputs([])] u8, #[inputs(0..=u64::MAX)] u64, #[inputs(0..=u64::MAX)] u64),
        Finish,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Never(..) => {}
                User::Finish => self.done = true,
            }
        }
    }
}

#[umpire_dsl::machine]
mod worlds {
    use umpire_dsl::{Action, State, World};

    #[derive(World)]
    pub(super) struct World {
        pub(super) starts: bool,
        pub(super) choices: u8,
    }

    pub(super) fn choices(world: &World) -> impl Iterator<Item = u8> {
        0..world.choices
    }

    #[derive(State)]
    pub(super) struct Model {
        pub(super) done: bool,
    }

    #[derive(Action, Copy)]
    pub(super) enum User {
        Pick(#[domain(choices)] u8),
        Finish,
    }

    impl Model {
        #[starts]
        fn starts(world: &World) -> impl IntoIterator<Item = Self> {
            world.starts.then_some(Self { done: false })
        }
        #[on(User)]
        fn user(&mut self, action: User, _: &World) {
            match action {
                User::Pick(_) => {}
                User::Finish => self.done = true,
            }
        }
    }
}

#[umpire_dsl::machine]
mod fair_values {
    use umpire_dsl::{Action, State, World, path};

    #[derive(World)]
    pub(super) struct World {
        pub(super) fair_true: bool,
    }

    #[derive(State)]
    pub(super) struct Model {
        pub(super) done: bool,
        pub(super) idle: bool,
    }

    #[derive(Action, Copy)]
    pub(super) enum User {
        Set(bool),
    }

    impl Model {
        #[starts]
        fn starts(_: &World) -> impl IntoIterator<Item = Self> {
            [Self { done: false, idle: false }]
        }
        #[on(User)]
        fn user(&mut self, action: User, _: &World) {
            match action {
                User::Set(true) => self.done = true,
                User::Set(false) => self.idle = !self.idle,
            }
        }

        #[progress]
        fn finishes(&self, world: &World) -> LeadsTo {
            path![!self.done, .., self.done].assuming(fair([User::Set(world.fair_true)]))
        }
    }
}

#[test]
fn over_limit_catalog_is_refused_before_any_step() {
    let before = STEPS.load(Ordering::SeqCst);
    let refusal = Checker::explore(&over::world(), Checker::BUDGET).err();
    assert_eq!(refusal, Some(Refusal::CatalogLimit { count: 4097, limit: 4096 }));
    assert_eq!(
        refusal.map(|refusal| refusal.to_string()).as_deref(),
        Some("action catalog has 4097 actions, over limit 4096")
    );
    let multiplied = Checker::explore(&multiplied::world(), Checker::BUDGET).err();
    assert_eq!(multiplied, Some(Refusal::CatalogLimit { count: 64 * 65, limit: 4096 }));
    let overflow = Checker::explore(&product::world(), Checker::BUDGET).err();
    assert_eq!(overflow, Some(Refusal::CatalogOverflow { limit: 4096 }));
    assert_eq!(STEPS.load(Ordering::SeqCst), before, "no handler runs on refused admission");
    assert!(Checker::explore(&admitted::world(), Checker::BUDGET).is_ok());
    let world = over::world();
    let result = Checker::explore_within(&world, world.starts(), Checker::BUDGET, 4097)
        .unwrap_or_else(|error| panic!("{error}"));
    let catalog = world.actions();
    assert_eq!(catalog.len(), 4097);
    assert_eq!(result.counts.evaluated, result.counts.reached * catalog.len());
    let refused = result
        .states()
        .flat_map(|(_, state)| catalog.iter().map(move |action| (state, action)))
        .filter(|(state, action)| world.step(state, (*action).clone()).is_empty())
        .count();
    assert_eq!(result.counts.refused, refused);
    assert_eq!(
        Checker::explore_within(&world, world.starts(), Checker::BUDGET, 4096).err(),
        Some(Refusal::CatalogLimit { count: 4097, limit: 4096 })
    );
}

#[test]
fn an_empty_factor_admits_an_overflowing_partner_and_reports_the_variant() {
    let report = Checker::explore(&emptied::world(), Checker::BUDGET)
        .and_then(|result| result.verify_all())
        .unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(report.warnings, ["variant User::Never has no actions"]);
    assert_eq!(report.counts.evaluated, report.counts.reached);
}

#[test]
fn empty_world_domains_stay_visible_including_with_no_starts() {
    for starts in [true, false] {
        let world = worlds::World::new(starts, 0);
        let report = Checker::explore(&world, Checker::BUDGET)
            .and_then(|result| result.verify_all())
            .unwrap_or_else(|error| panic!("{error}"));
        assert_eq!(report.counts.starts, usize::from(starts));
        assert_eq!(report.counts.evaluated, report.counts.reached);
        assert_eq!(report.warnings[0], "variant User::Pick has no actions");
    }
    let world = worlds::World::new(true, 3);
    let report = Checker::explore(&world, Checker::BUDGET)
        .and_then(|result| result.verify_all())
        .unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(report.counts.evaluated, report.counts.reached * 4);
    assert!(report.warnings.iter().all(|warning| !warning.starts_with("variant")));
}

#[test]
fn progress_revalidates_the_catalog_under_the_explored_limit() {
    let world = fair_values::World::new(true);
    let result = Checker::explore_within(&world, world.starts(), Checker::BUDGET, 2)
        .unwrap_or_else(|error| panic!("{error}"));
    let claim = world.progress().remove(0);
    assert_eq!(
        result.verify_progress(&world, &claim).map(|claim| claim.outcome),
        Ok(CheckOutcome::Verified)
    );
    assert_eq!(
        Checker::explore_within(&world, world.starts(), Checker::BUDGET, 1).err(),
        Some(Refusal::CatalogLimit { count: 2, limit: 1 })
    );
}

#[test]
fn fairness_names_one_concrete_input_not_its_variant() {
    for (fair_true, outcome) in [(true, CheckOutcome::Verified), (false, CheckOutcome::Refuted)] {
        let report = Checker::explore(&fair_values::World::new(fair_true), Checker::BUDGET)
            .and_then(|result| result.verify_all())
            .unwrap_or_else(|error| panic!("{error}"));
        assert_eq!(report.claims[0].outcome, outcome, "fair Set({fair_true})");
    }
}
