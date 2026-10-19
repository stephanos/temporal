use umpire_dsl::{Machine, Steps};

#[derive(Clone, PartialEq, Eq, Hash, Debug)]
struct State;

struct World;

impl Machine for World {
    type State = State;
    type Action = u8;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<State> {
        vec![State]
    }
    fn actions(&self) -> Vec<u8> {
        vec![0]
    }
    fn internal(&self, _: &u8) -> bool {
        false
    }
    fn step(&self, _: &State, _: u8) -> Steps<State, (), ()> {
        Steps::new()
    }
}

fn main() {}
