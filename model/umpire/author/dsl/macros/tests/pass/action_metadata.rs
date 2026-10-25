#![forbid(unsafe_code)]
#![allow(dead_code)]

use umpire_dsl::Action;

#[derive(Action)]
enum Generic<T: Clone + 'static> {
    #[metadata(example(arbitrary => [tokens]))]
    Tuple(T),
    Named {
        value: bool,
    },
    Unit,
}

#[derive(Action)]
enum Empty {}

macro_rules! generic_inventory {
    (model { $model:path } actors {
        $(#[$attr:meta])* enum Generic<T: Clone + 'static> {
            #[metadata(example(arbitrary => [tokens]))] Tuple(T),
            Named { value: bool $(,)? },
            Unit,
        }
    } body { $name:ident }) => {
        const $name: usize = 3;
    };
}

macro_rules! empty_inventory {
    (model { $model:path } actors { $(#[$attr:meta])* enum Empty {} } body {}) => {};
}

__umpire_metadata_Generic!(generic_inventory; Generic<bool>; { GENERIC_VARIANTS });
__umpire_metadata_Empty!(empty_inventory; Empty; {});

#[umpire_dsl::machine]
mod model {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub struct Model {
        enabled: bool,
    }

    #[derive(Action)]
    pub enum User {
        #[metadata(example(opaque = path::to::item))]
        Toggle,
        MissingMetadata,
    }

    #[derive(Action)]
    pub enum Environment {}

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Toggle => self.enabled = !self.enabled,
                User::MissingMetadata => {}
            }
        }

        #[on(Environment)]
        fn environment(&mut self, action: Environment) {
            match action {}
        }
    }
}

macro_rules! machine_inventory {
    (model { $model:path } actors {
        $(#[$user_attr:meta])* pub enum User {
            #[metadata(example(opaque = path::to::item))] Toggle,
            MissingMetadata,
        }
        $(#[$environment_attr:meta])* pub enum Environment {}
    } body { $name:ident }) => {
        const $name: usize = 2;
    };
}

model::action_metadata!(machine_inventory; model; { MACHINE_ACTORS });

fn main() -> Result<(), Box<dyn std::error::Error>> {
    assert_eq!(GENERIC_VARIANTS, 3);
    assert_eq!(MACHINE_ACTORS, 2);
    assert_eq!(Generic::<bool>::actions().len(), 5);
    assert!(Empty::actions().is_empty());
    use umpire_dsl::Machine;
    let world = model::world();
    assert_eq!(world.actions().len(), 2);
    assert_eq!(world.checked_step(&world.starts()[0], model::User::Toggle.into())?.len(), 1);
    Ok(())
}
