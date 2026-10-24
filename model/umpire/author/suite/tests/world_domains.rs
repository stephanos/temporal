//! World-bound action domains feed suite queries, and every suite route admits the catalog.

use umpire_checker::Refusal as CheckerRefusal;
use umpire_dsl::{Machine, Path, Pattern, Waypoint};
use umpire_suite::{Cases, Refusal, Space, cover, find, refuted, sample, verify_all};

#[umpire_dsl::machine]
mod picks {
    use umpire_dsl::{Action, State, World};

    #[derive(World)]
    pub(super) struct World {
        pub(super) choices: u8,
    }

    pub(super) fn choices(world: &World) -> impl Iterator<Item = u8> {
        0..world.choices
    }

    #[derive(State)]
    pub(super) struct Model {
        pub(super) last: Option<u8>,
    }

    #[derive(Action, Copy)]
    pub(super) enum User {
        Pick(#[domain(choices)] u8),
    }

    impl Model {
        #[starts]
        fn starts(_: &World) -> impl IntoIterator<Item = Self> {
            [Self { last: None }]
        }
        #[on(User)]
        fn user(&mut self, action: User, _: &World) {
            match action {
                User::Pick(value) if self.last.is_none() => self.last = Some(value),
                User::Pick(_) => {}
            }
        }
    }
}

fn picked(world: &picks::World) -> Path<picks::Model, picks::Action, ()> {
    Path::pattern(Pattern::Sequence(vec![
        Pattern::state(Waypoint::judged(|state: &picks::Model| state.last.is_none())),
        Pattern::action(picks::User::actions_in(world).map(picks::Action::from), None, None),
    ]))
}

#[test]
fn contextual_domains_take_the_selected_world() {
    for choices in [1, 3] {
        let world = picks::World::new(choices);
        let path = picked(&world);
        let covered = cover(&Space::new(world), Cases(usize::from(choices)), path)
            .unwrap_or_else(|error| panic!("{error}"));
        assert_eq!(covered.coverage().actions.len(), usize::from(choices));
        assert!(covered.coverage().actions.iter().all(|entry| entry.touched));
    }
    let space = Space::new(picks::World::new(2));
    let trace =
        find(&space, picked(&picks::World::new(2))).unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(trace.steps.len(), 1);
    assert_eq!(trace.steps[0].action.to_string(), "User::Pick(0)");
}

#[test]
fn every_suite_route_admits_the_catalog_under_the_space_limit() {
    let refusal = || Refusal::Checker(CheckerRefusal::CatalogLimit { count: 3, limit: 2 });
    let limited = || Space::new(picks::World::new(3)).catalog_limit(2);
    let world = picks::World::new(3);
    assert_eq!(verify_all(&limited()).err(), Some(refusal()));
    assert_eq!(
        refuted(&limited(), |state: &picks::Model| state.last != Some(2)).err(),
        Some(refusal())
    );
    assert_eq!(find(&limited(), picked(&world)).err(), Some(refusal()));
    assert_eq!(cover(&limited(), Cases(3), picked(&world)).err(), Some(refusal()));
    assert_eq!(sample(&limited(), picked(&world), 1, 7).err(), Some(refusal()));
    let state_only =
        umpire_dsl::path![picks::Model { last: None }, .., picks::Model { last: Some(2) }];
    assert_eq!(find(&limited(), state_only).err(), Some(refusal()));
    let admitted = Space::new(picks::World::new(3)).catalog_limit(3);
    assert!(verify_all(&admitted).is_ok());
    assert!(refuted(&admitted, |state: &picks::Model| state.last != Some(2)).is_ok());
    assert!(picks::World::new(3).checked_catalog(3).is_ok());
}
