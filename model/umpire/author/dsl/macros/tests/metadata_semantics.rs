//! Opaque action metadata cannot authorize executable outcome or self-loop semantics.

#![forbid(unsafe_code)]

use umpire_dsl::{
    __lifted::{Lifts, Type},
    CheckOutcome, Machine, Outcome, Rejection,
};

#[umpire_dsl::machine]
mod opaque {
    use umpire_dsl::{Action, State, check};

    #[derive(State)]
    pub(crate) struct Model {
        #[starts(0..=0)]
        pub(crate) value: u8,
    }

    #[derive(Action, Copy)]
    pub(crate) enum User {
        #[metadata(example(reject!(NotFound), nested(stay!())))]
        Advance,
        Idle,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Advance if self.value == 0 => self.value = 1,
                User::Advance | User::Idle => {}
            }
        }

        #[invariant]
        fn bounded(&self) {
            check!(self.value <= 1);
        }
    }
}

#[umpire_dsl::machine]
mod executable {
    use umpire_dsl::{Action, Rejection, State, check, reject, stay};

    #[derive(State)]
    pub(crate) struct Model {
        #[starts(0..=0)]
        pub(crate) value: u8,
    }

    #[derive(Action, Copy)]
    pub(crate) enum User {
        Advance,
        Ping,
        Refuse,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Advance if self.value == 0 => self.value = 1,
                User::Ping => stay!(),
                User::Refuse => reject!(Rejection::NotFound),
                User::Advance => {}
            }
        }

        #[invariant]
        fn bounded(&self) {
            check!(self.value <= 1);
        }
    }
}

fn declaration(name: &str) -> Result<umpire_front::declare::Machine, Box<dyn std::error::Error>> {
    let file = syn::parse_file(include_str!("metadata_semantics.rs"))?;
    let module = file
        .items
        .into_iter()
        .find_map(|item| match item {
            syn::Item::Mod(module) if module.ident == name => Some(module),
            _ => None,
        })
        .ok_or("test machine not found")?;
    Ok(umpire_front::declare::machine(&proc_macro2::TokenStream::new(), &module)?)
}

#[test]
fn opaque_macro_tokens_preserve_unit_outcome_catalog_and_native_lifted_rows()
-> umpire_testing::TestResult {
    fn unit_outcome<M: Machine<Outcome = ()>>(_: &M) {}
    let world = opaque::World;
    unit_outcome(&world);
    let declared = declaration("opaque")?;
    assert!(!declared.rejecting);
    assert!(!declared.staying);
    let catalog = world.outcome_catalog().ok_or("outcome catalog missing")?;
    assert_eq!(catalog.len(), 1);
    assert!(catalog[0].matches(&()));
    assert!(world.checked_step(&opaque::Model { value: 0 }, opaque::User::Idle.into())?.is_empty());
    let model = opaque::World::__umpire_lifted().model.map_err(|error| format!("{error:?}"))?;
    assert_eq!(model.outcome_type, Type::Unit);
    assert!(
        model
            .actions
            .iter()
            .flat_map(|action| &action.branches)
            .all(|branch| !branch.preserve_self_loop)
    );
    let agreement = umpire_testing::interpreted(|| opaque::World, None, 16, None)?;
    assert_eq!((agreement.states, agreement.edges), (2, 1));
    let graph = umpire_testing::Checker::explore(&world, 16)?;
    assert_eq!(graph.verify_all()?.outcome(), CheckOutcome::Verified);
    Ok(())
}

#[test]
fn executable_reject_and_stay_keep_their_native_and_lifted_semantics() -> umpire_testing::TestResult
{
    let declared = declaration("executable")?;
    assert!(declared.rejecting);
    assert!(declared.staying);
    let world = executable::World;
    let state = executable::Model { value: 0 };
    let ping = world.checked_step(&state, executable::User::Ping.into())?;
    assert_eq!(ping.len(), 1);
    assert_eq!(ping[0].0, Outcome::Accepted);
    let refused = world.checked_step(&state, executable::User::Refuse.into())?;
    assert_eq!(refused.len(), 1);
    assert_eq!(refused[0].0, Outcome::Rejected(Rejection::NotFound));
    let agreement = umpire_testing::interpreted(|| executable::World, None, 16, None)?;
    assert_eq!((agreement.states, agreement.edges), (2, 5));
    Ok(())
}
