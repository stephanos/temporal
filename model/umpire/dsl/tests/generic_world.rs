//! Generics must form a usable Machine, including actions and typed progress adapters.

use umpire_dsl::{Machine, Verdict};

trait Token: Clone + Eq + std::hash::Hash + std::fmt::Debug + 'static {
    fn seed() -> Self;
}

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
struct Owned(String);

impl Token for Owned {
    fn seed() -> Self {
        Self("seed".into())
    }
}

#[umpire_dsl::machine]
mod generic {
    use super::Token;
    use umpire_dsl::{Action, State, World, check, path};

    #[derive(World)]
    pub(super) struct World<T>
    where
        T: Token,
    {
        pub(super) initial: T,
        pub(super) limit: u8,
    }

    #[derive(State)]
    pub(super) struct Model<T>
    where
        T: Token,
    {
        pub(super) value: T,
        pub(super) count: u8,
    }

    #[derive(Action)]
    pub(super) enum User<U: Token> {
        Replace(#[inputs([U::seed()])] U),
    }

    #[starts]
    impl<T: Token> Model<T> {
        fn starts(w: &World<T>) -> impl IntoIterator<Item = Self> {
            [Self { value: w.initial.clone(), count: 0 }]
        }
    }

    impl<T: Token> Model<T> {
        #[on(User)]
        fn replace(&mut self, action: User<T>, w: &World<T>) {
            match action {
                User::Replace(value) if self.count < w.limit => {
                    self.value = value;
                    self.count += 1;
                }
                User::Replace(_) => {}
            }
        }

        #[invariant]
        fn bounded(&self, w: &World<T>) {
            check!(self.count <= w.limit);
        }

        #[postcondition]
        fn advances(&self, old: &Self, (): &(), w: &World<T>) {
            check!(self.count == old.count + 1 && self.count <= w.limit);
        }

        #[assumption]
        fn fair_user(_: &World<T>) -> Assumption<T> {
            fair([User::Replace(T::seed())])
        }

        #[progress]
        fn finishes(&self, w: &World<T>) -> LeadsTo<T> {
            path![self.count == 0, ..=2, self.count == w.limit].assuming(Self::fair_user(w))
        }
    }
}

#[test]
fn a_generic_model_world_and_actor_compile_and_evaluate_together() {
    let world = generic::World::new(Owned("chosen".into()), 2);
    let state = world.starts().remove(0);
    assert_eq!(state.value.0, "chosen");
    assert_eq!(world.actions().len(), 1);
    let edge = world.step(&state, world.actions().remove(0)).remove(0);
    assert_eq!(edge.2.count, 1);
    assert_eq!(edge.2.value.0, "seed");
    assert_eq!(world.actions()[0].to_string(), "User::Replace(Owned(\"seed\"))");
    assert_eq!(
        world.postconditions()[0].holds(&world, &state, &world.actions()[0], &edge),
        Verdict::Holds
    );
    assert_eq!((world.invariants()[0].predicate)(&world, &edge.2), Verdict::Holds);
    let progress = world.progress();
    let evaluate = progress[0].evaluator.unwrap_or_else(|| panic!("missing progress"));
    assert_eq!(evaluate(&world, &state).trigger, Verdict::Holds);
    assert_eq!(world.assumptions()[0].1.fair.len(), 1);
}

#[umpire_dsl::machine]
mod fixed_actor {
    use umpire_dsl::{Action, State, World, path};

    #[derive(World)]
    pub(super) struct World<const N: usize = 3> {
        limit: u8,
    }
    #[derive(State)]
    pub(super) struct Model<const N: usize = 3> {
        pub(super) count: u8,
        bytes: [u8; N],
    }
    #[derive(Action)]
    pub(super) enum User {
        Advance,
    }
    #[starts]
    impl<const N: usize> Model<N> {
        fn starts(_: &World<N>) -> [Self; 1] {
            [Self { count: 0, bytes: [0; N] }]
        }
    }
    impl<const N: usize> Model<N> {
        #[on(User)]
        fn user(&mut self, _: User, w: &World<N>) {
            if self.count < w.limit {
                self.count += 1;
            }
        }

        #[progress]
        fn advances(&self, w: &World<N>) -> LeadsTo<N> {
            path![self.count == 0, ..=1, self.count == w.limit]
        }
    }
}

#[test]
fn const_generic_states_keep_nongeneric_action_carriers() {
    let world = fixed_actor::World::<3>::new(1);
    let state = world.starts().remove(0);
    assert_eq!(world.step(&state, fixed_actor::User::Advance.into())[0].2.count, 1);
    let claim = world.progress().remove(0);
    assert_eq!(
        claim.evaluator.unwrap_or_else(|| panic!("missing evaluator"))(&world, &state).trigger,
        Verdict::Holds
    );
}

#[umpire_dsl::machine]
mod conditional_bounds {
    use std::marker::PhantomData;
    use umpire_dsl::{Action, State, World};

    #[derive(World)]
    pub(super) struct World<T> {
        marker: PhantomData<T>,
    }
    #[derive(State)]
    pub(super) struct Model<T> {
        marker: PhantomData<T>,
        pub(super) changed: bool,
    }
    #[derive(Action)]
    pub(super) enum User<T> {
        Change(#[inputs([PhantomData::<T>])] PhantomData<T>),
    }
    #[starts]
    impl<T> Model<T> {
        fn starts(_: &World<T>) -> [Self; 1] {
            [Self { marker: PhantomData, changed: false }]
        }
    }
    impl<T> Model<T> {
        #[on(User)]
        fn user(&mut self, _: User<T>, _: &World<T>) {
            self.changed = true;
        }
    }
}

#[test]
fn generated_carriers_bound_field_types_without_requiring_traits_on_parameters() {
    struct NoTraits;
    let world = conditional_bounds::World::<NoTraits>::new(std::marker::PhantomData);
    let copy = world.clone();
    let state = copy.starts().remove(0);
    assert!(world.step(&state, world.actions().remove(0))[0].2.changed);
}

#[umpire_dsl::machine]
mod bounded_actor {
    use super::Token;
    use std::marker::PhantomData;
    use umpire_dsl::{Action, State, World, path};

    #[derive(World)]
    pub(super) struct World<T = super::Owned> {
        marker: PhantomData<T>,
    }
    #[derive(State)]
    pub(super) struct Model<T = super::Owned> {
        marker: PhantomData<T>,
        pub(super) changed: bool,
    }
    #[derive(Action)]
    pub(super) enum User<T: Token> {
        Change(#[inputs([PhantomData::<T>])] PhantomData<T>),
    }
    #[starts]
    impl<T: Token> Model<T> {
        fn starts(_: &World<T>) -> [Self; 1] {
            [Self { marker: PhantomData, changed: false }]
        }
    }
    impl<T: Token> Model<T> {
        #[on(User)]
        fn user(&mut self, _: User<T>, _: &World<T>) {
            self.changed = true;
        }

        #[progress]
        fn changes(&self, _: &World<T>) -> LeadsTo<T> {
            path![!self.changed, ..=1, self.changed].assuming(fair([User::Change(PhantomData)]))
        }
    }
}

#[test]
fn progress_helpers_include_actor_bounds_absent_from_state_declarations() {
    let world = bounded_actor::World::<Owned>::new(std::marker::PhantomData);
    let state = world.starts().remove(0);
    let next = world.step(&state, world.actions().remove(0)).remove(0).2;
    let claim = world.progress().remove(0);
    let evaluate = claim.evaluator.unwrap_or_else(|| panic!("missing evaluator"));
    assert_eq!(evaluate(&world, &state).trigger, Verdict::Holds);
    assert_eq!(evaluate(&world, &next).target, Verdict::Holds);
}

#[umpire_dsl::machine]
mod implementation_bounds {
    use super::Token;
    use umpire_dsl::{Action, State, World};

    #[derive(World)]
    pub(super) struct World<U> {
        initial: U,
    }
    #[derive(State)]
    pub(super) struct Model<T> {
        pub(super) value: T,
    }
    #[derive(Action)]
    pub(super) enum User {
        Replace,
    }
    #[starts]
    impl<V: Token> Model<V> {
        fn starts(w: &World<V>) -> [Self; 1] {
            [Self { value: w.initial.clone() }]
        }
    }
    impl<V: Token> Model<V> {
        #[on(User)]
        fn user(&mut self, _: User, _: &World<V>) {
            self.value = V::seed();
        }
    }
}

#[test]
fn authored_implementation_bounds_govern_the_generated_machine() {
    let world = implementation_bounds::World::new(Owned("chosen".into()));
    let state = world.starts().remove(0);
    assert_eq!(state.value.0, "chosen");
    assert_eq!(world.step(&state, world.actions().remove(0))[0].2.value.0, "seed");
}
