// Option G (library): stateright 0.31 as the engine, its Model trait as the authoring surface.
// Does not compile only because the claim extensions are hypothetical; the Model impl is real
// stateright API. Note: stateright's last release and commit are July 2025, single maintainer;
// treat it as code to read or vendor, not a dependency to build on.
//
// stateright gives: `Model { init_states, actions, next_state, properties }`, BFS/DFS checking
// with a shared visited set, `Property::always` and `Property::sometimes`, counterexample paths,
// symmetry reduction, and an actor layer. No DSL: everything is a method.
//
// What it costs us: `properties` are state predicates only. `before`, `after(action)`, scopes and
// progress with fairness do not exist and would be added around it. The shape of `next_state`
// (match with guards, `None` refuses) is exactly our transitions, which is why this is the
// reference design for the checker even if the crate is not depended on.

use stateright::{Checker, Model, Property};

const BOILING_TEMPERATURE: i32 = 100;
const COLDEST_TEMPERATURE: i32 = -40;

#[derive(Clone, PartialEq, Eq, Hash, Debug)]
pub struct Kettle {
    temperature: i32,
    heating_on: bool,
    beeping: bool,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug)]
pub enum Act {
    UserPressButton,
    UserDepressButton,
    HeaterHeat,
    HeaterFailover,
}

pub struct KettleModel;

impl Model for KettleModel {
    type State = Kettle;
    type Action = Act;

    fn init_states(&self) -> Vec<Kettle> {
        (COLDEST_TEMPERATURE..=40)
            .map(|temperature| Kettle { temperature, heating_on: false, beeping: false })
            .collect()
    }

    fn actions(&self, _state: &Kettle, actions: &mut Vec<Act>) {
        actions.extend([Act::UserPressButton, Act::UserDepressButton, Act::HeaterHeat, Act::HeaterFailover]);
    }

    fn next_state(&self, k: &Kettle, action: Act) -> Option<Kettle> {
        let mut next = k.clone();
        match action {
            Act::UserPressButton if !k.heating_on => {
                next.heating_on = true;
                next.beeping = false;
            }
            Act::HeaterFailover if k.heating_on && k.temperature >= BOILING_TEMPERATURE => {
                next.heating_on = false;
                next.beeping = true;
            }
            Act::HeaterHeat if k.heating_on && k.temperature < BOILING_TEMPERATURE => {
                next.temperature += 1;
            }
            Act::UserDepressButton if k.heating_on => {
                next.heating_on = false;
                next.beeping = false;
            }
            _ => return None,
        }
        Some(next)
    }

    fn properties(&self) -> Vec<Property<Self>> {
        vec![
            Property::always("beeps_only_when_off", |_, k: &Kettle| !k.beeping || !k.heating_on),
            // stateright has no step claims: warms_only_when_on and fails_over_at_boiling need a
            // history variable (previous state, last action) folded into State, or an extension.
            // Progress "boils" under `without(DepressButton)` needs fairness and a scoped action
            // set, neither of which exists; `sometimes("beeps", ..)` is the closest built-in.
            Property::sometimes("beeps", |_, k: &Kettle| k.beeping),
        ]
    }
}

// tests/kettle.rs
//   #[test]
//   fn kettle() {
//       let checker = KettleModel.checker().spawn_bfs().join();
//       checker.assert_properties();
//   }
