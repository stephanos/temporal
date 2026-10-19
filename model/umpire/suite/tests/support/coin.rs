//! A manual Model whose one action emits two steps, registered by hand.

use umpire_dsl::{Machine, Registered, Steps};

/// A coin, unseen or showing a side.
#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub(crate) enum Coin {
    Unseen,
    Heads,
    Tails,
}

impl std::fmt::Display for Coin {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(out, "Coin::{self:?}")
    }
}

/// Tossing an unseen coin.
#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub(crate) struct Toss;

impl std::fmt::Display for Toss {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str("Toss")
    }
}

pub(crate) struct World;

impl Machine for World {
    type State = Coin;
    type Action = Toss;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<Coin> {
        vec![Coin::Unseen]
    }
    fn actions(&self) -> Vec<Toss> {
        vec![Toss]
    }
    fn internal(&self, _: &Toss) -> bool {
        false
    }
    fn step(&self, state: &Coin, _: Toss) -> Steps<Coin, (), ()> {
        let mut steps = Steps::new();
        if *state == Coin::Unseen {
            steps.push(((), Vec::new(), Coin::Heads));
            steps.push(((), Vec::new(), Coin::Tails));
        }
        steps
    }
}

impl Registered for Coin {
    type World = World;
    fn world() -> World {
        World
    }
}
