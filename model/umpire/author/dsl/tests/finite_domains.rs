//! Finite input domains, World-bound action domains and checked catalog admission.

#![forbid(unsafe_code)]

use umpire_dsl::{Action, ActorValue, CatalogError, CatalogVariant, Finite, Machine};

#[derive(Finite, Clone, Copy, Debug, PartialEq, Eq, Hash)]
enum Speed {
    Slow,
    Fast,
}

#[derive(Finite, Clone, Copy, Debug, PartialEq, Eq, Hash)]
struct Level(#[inputs(1..=3)] u8);

#[derive(Finite, Clone, Copy, Debug, PartialEq, Eq, Hash)]
struct Offset {
    #[inputs(-2..=1)]
    raw: i8,
}

#[derive(Finite)]
enum Never {}

#[test]
fn finite_domains_enumerate_in_declaration_order() {
    assert_eq!(bool::domain().collect::<Vec<_>>(), [false, true]);
    assert_eq!(Speed::domain().collect::<Vec<_>>(), [Speed::Slow, Speed::Fast]);
    assert_eq!(Level::domain().collect::<Vec<_>>(), [Level(1), Level(2), Level(3)]);
    assert_eq!(Offset::domain().map(|offset| offset.raw).collect::<Vec<_>>(), [-2, -1, 0, 1]);
    assert_eq!(Never::domain().count(), 0);
}

#[test]
fn one_range_declaration_supplies_checked_construction() {
    assert_eq!(Level::try_new(0), None);
    assert_eq!(Level::try_new(1), Some(Level(1)));
    assert_eq!(Level::try_new(3), Some(Level(3)));
    assert_eq!(Level::try_new(4), None);
    assert!(Offset::contains(-2) && Offset::contains(1));
    assert!(!Offset::contains(-3) && !Offset::contains(2));
    // Enumeration is not validation: a raw value outside the range is still constructible.
    assert!(!Level::contains(Level(9).0));
}

#[derive(Action)]
enum Static {
    Toggle(bool),
    Mixed {
        speed: Speed,
        #[inputs([2, 2])]
        count: u8,
        level: Level,
    },
}

#[test]
fn finite_fields_mix_with_explicit_domains_last_field_fastest() {
    let actions: Vec<_> = Static::actions().iter().map(ToString::to_string).collect();
    assert_eq!(actions.len(), 2 + 2 * 2 * 3);
    assert_eq!(
        actions[..3],
        [
            "Static::Toggle(false)",
            "Static::Toggle(true)",
            "Static::Mixed { speed: Slow, count: 2, level: Level(1) }"
        ]
    );
    assert_eq!(actions[4], "Static::Mixed { speed: Slow, count: 2, level: Level(3) }");
    assert_eq!(actions[5], "Static::Mixed { speed: Slow, count: 2, level: Level(1) }");
    assert_eq!(actions[13], "Static::Mixed { speed: Fast, count: 2, level: Level(3) }");
    let lazy = Static::__umpire_path_actions();
    assert_eq!(lazy.count(), Ok(actions.len()));
    assert_eq!(
        (0..actions.len()).map(|index| lazy.get(index).to_string()).collect::<Vec<_>>(),
        actions
    );
}

#[derive(Action)]
enum Generic<T: Finite> {
    Pick(T),
}

#[test]
fn generic_finite_fields_keep_authored_bounds() {
    assert_eq!(Generic::<Speed>::actions().len(), 2);
    assert_eq!(Generic::<bool>::__umpire_path_actions().count(), Ok(2));
}

#[umpire_dsl::machine]
mod sized {
    use super::Speed;
    use umpire_dsl::{Action, State, World};

    #[derive(World)]
    pub(super) struct World {
        pub(super) sizes: u8,
        pub(super) empty: bool,
    }

    pub(super) fn sizes(world: &World) -> impl Iterator<Item = u8> {
        0..world.sizes
    }

    pub(super) fn picks(world: &World) -> impl Iterator<Item = bool> {
        (!world.empty).then_some(true).into_iter()
    }

    #[derive(State)]
    pub(super) struct Model {
        pub(super) last: u8,
    }

    #[derive(Action, Copy)]
    pub(super) enum User {
        Set(#[domain(sizes)] u8, Speed),
        Choose {
            flag: bool,
            #[domain(picks)]
            pick: bool,
            #[inputs(7..=8)]
            code: u8,
        },
        Reset,
    }

    #[derive(Action, Copy)]
    pub(super) enum Clock {
        Tick,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User, world: &World) {
            match action {
                User::Set(size, _) if size < world.sizes => self.last = size,
                User::Set(..) | User::Choose { .. } => {}
                User::Reset => self.last = 0,
            }
        }

        #[on(Clock)]
        fn clock(&mut self, action: Clock) {
            match action {
                Clock::Tick => self.last = 0,
            }
        }
    }
}

fn labels(actions: &[sized::Action]) -> Vec<String> {
    actions.iter().map(ToString::to_string).collect()
}

#[test]
fn two_worlds_of_one_type_select_their_own_domains() {
    let two = sized::World::new(2, false);
    let one = sized::World::new(1, true);
    assert_eq!(
        labels(&two.actions()),
        [
            "User::Set(0, Slow)",
            "User::Set(0, Fast)",
            "User::Set(1, Slow)",
            "User::Set(1, Fast)",
            "User::Choose { flag: false, pick: true, code: 7 }",
            "User::Choose { flag: false, pick: true, code: 8 }",
            "User::Choose { flag: true, pick: true, code: 7 }",
            "User::Choose { flag: true, pick: true, code: 8 }",
            "User::Reset",
            "Clock::Tick",
        ]
    );
    assert_eq!(
        labels(&one.actions()),
        ["User::Set(0, Slow)", "User::Set(0, Fast)", "User::Reset", "Clock::Tick"]
    );
    for world in [&two, &one] {
        let catalog = world
            .checked_catalog(umpire_dsl::CATALOG_LIMIT)
            .unwrap_or_else(|error| panic!("{error}"));
        assert_eq!(catalog.actions(), world.actions());
        let domain = sized::User::actions_in(world);
        let concrete: Vec<_> = world
            .actions()
            .into_iter()
            .filter_map(|action| match action {
                sized::Action::User(user) => Some(user),
                sized::Action::Clock(_) => None,
            })
            .collect();
        assert_eq!(domain.count(), Ok(concrete.len()));
        assert_eq!(
            (0..concrete.len()).map(|index| domain.get(index)).collect::<Vec<_>>(),
            concrete
        );
    }
    let fault = two.fault_catalog().unwrap_or_default();
    assert_eq!(fault, []);
    assert!(!sized::User::Reset.internal() && !ActorValue::fault(&sized::User::Reset));
}

#[test]
fn checked_catalog_records_declared_variant_counts_including_empty() {
    let variants = |world: &sized::World| {
        world.checked_catalog(16).map(|catalog| catalog.variants().map(<[CatalogVariant]>::to_vec))
    };
    let count = |name: &str, count| CatalogVariant { name: name.into(), count };
    assert_eq!(
        variants(&sized::World::new(0, true)),
        Ok(Some(vec![
            count("User::Set", 0),
            count("User::Choose", 0),
            count("User::Reset", 1),
            count("Clock::Tick", 1)
        ]))
    );
    assert_eq!(
        sized::World::new(3, false).checked_catalog(12).map(|catalog| catalog.actions().len()),
        Ok(12)
    );
    assert_eq!(
        sized::World::new(3, false).checked_catalog(11),
        Err(CatalogError::TooLarge { count: 12, limit: 11 })
    );
}

#[umpire_dsl::machine]
mod limits {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub(super) struct Model {
        touched: bool,
    }

    #[derive(Action, Copy)]
    pub(super) enum Admitted {
        At(#[inputs(0..=4095)] u16),
    }

    #[derive(Action, Copy)]
    pub(super) enum Over {
        At(#[inputs(0..=4096)] u16),
    }

    #[derive(Action, Copy)]
    pub(super) enum Product {
        At(#[inputs(0..=63)] u8, #[inputs(0..=64)] u8),
    }

    #[derive(Action, Copy)]
    pub(super) enum Overflow {
        At(#[inputs(0..=u64::MAX)] u64, #[inputs(0..=u64::MAX)] u64),
    }

    #[derive(Action, Copy)]
    pub(super) enum Empty {
        At(#[inputs([])] u8, #[inputs(0..=u64::MAX)] u64, #[inputs(0..=u64::MAX)] u64),
    }

    impl Model {
        #[on(Admitted)]
        fn admitted(&mut self, _: Admitted) {
            self.touched = true;
        }
        #[on(Over)]
        fn over(&mut self, _: Over) {
            self.touched = true;
        }
        #[on(Product)]
        fn product(&mut self, _: Product) {
            self.touched = true;
        }
        #[on(Overflow)]
        fn overflow(&mut self, _: Overflow) {
            self.touched = true;
        }
        #[on(Empty)]
        fn empty(&mut self, _: Empty) {
            self.touched = true;
        }
    }
}

#[test]
fn admission_counts_every_domain_before_materializing() {
    use limits::{Admitted, Empty, Over, Overflow, Product};
    use umpire_dsl::path_build::ActorCatalog;
    fn total<A: 'static>(
        variants: Vec<(String, umpire_dsl::ActionDomain<A>)>,
    ) -> Vec<Result<usize, umpire_dsl::PatternError>> {
        variants.into_iter().map(|(_, domain)| domain.count()).collect()
    }
    assert_eq!(total(<Admitted as ActorCatalog<()>>::catalog(&(), 0)), [Ok(4096)]);
    assert_eq!(total(<Over as ActorCatalog<()>>::catalog(&(), 0)), [Ok(4097)]);
    assert_eq!(total(<Product as ActorCatalog<()>>::catalog(&(), 0)), [Ok(64 * 65)]);
    assert!(total(<Overflow as ActorCatalog<()>>::catalog(&(), 0))[0].is_err());
    assert_eq!(total(<Empty as ActorCatalog<()>>::catalog(&(), 0)), [Ok(0)]);
    let world = limits::world();
    // Over, Product and Overflow together overflow before any action is built.
    assert_eq!(
        world.checked_catalog(usize::MAX).map(|_| ()),
        Err(CatalogError::Overflow { limit: usize::MAX })
    );
}

#[test]
fn concrete_world_dependent_values_infer_typed_paths() {
    let path = umpire_dsl::path![sized::Model { last: 0 }, sized::User::Set(1, Speed::Fast)];
    let alternatives = path.alternatives(8).unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(alternatives.len(), 1);
    assert_eq!(alternatives[0].choices, ["User::Set(1, Fast)"]);
}
