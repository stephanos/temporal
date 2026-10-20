use std::fmt::Write;

use umpire_dsl::{Gap, Machine, Registered, Steps, Waypoint};
use umpire_suite::{Space, find, path};

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub(crate) struct Node {
    at: u8,
    open: bool,
}

impl std::fmt::Display for Node {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(out, "Node({}, open={})", self.at, self.open)
    }
}

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub(crate) enum Action {
    Turn,
    Escape,
}

impl std::fmt::Display for Action {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(out, "{self:?}")
    }
}

pub(crate) struct World;

impl Machine for World {
    type State = Node;
    type Action = Action;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<Node> {
        vec![Node { at: 0, open: false }]
    }
    fn actions(&self) -> Vec<Action> {
        vec![Action::Turn, Action::Escape]
    }
    fn internal(&self, _: &Action) -> bool {
        false
    }
    fn step(&self, state: &Node, action: Action) -> Steps<Node, (), ()> {
        let target = match action {
            Action::Turn if state.at < 3 => Some((state.at + 1) % 3),
            Action::Escape if state.open && state.at == 2 => Some(3),
            _ => None,
        };
        target.into_iter().map(|at| ((), Vec::new(), Node { at, open: state.open })).collect()
    }
}

impl Registered for Node {
    type World = World;
    fn world() -> World {
        World
    }
}

fn outcome<T: std::fmt::Display, E: std::fmt::Display>(answer: Result<T, E>) -> String {
    answer.map_or_else(|refusal| format!("refused: {refusal}\n"), |path| path.to_string())
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let mut out = String::new();
    let start = Node { at: 0, open: false };
    let space = Space::from([start.clone()]).budget(3);
    write!(out, "unreachable, bounded by a billion steps, budget 3:\n{}", outcome(find(&space, path![start.clone(), ..=1_000_000_000, false])))?;
    let long = Waypoint::Is(start.clone()).then_judged(Gap::AtMost(1), |node| node.at == 1);
    let long = (2..=12).fold(long, |path, index| path.then_judged(Gap::AtMost(1), move |node| u32::from(node.at) == index % 3));
    let answer = find(&space, long);
    writeln!(out, "twelve steps around three Model states, budget 3: {}", answer.map_or_else(|error| format!("refused: {error}"), |path| format!("{} steps", path.steps.len())))?;
    for budget in [3, 4] {
        let start = Node { at: 0, open: true };
        let pattern = path![start.clone(), ..=1, |n: &Node| n.at == 1 || n.at == 3, ..=1, |n| n.at == 2 || n.at == 3, ..=1, |n| n.at == 0 || n.at == 3, ..=1, |n| n.at == 1 || n.at == 3];
        write!(out, "loop witness can hide shorter escape, budget {budget}:\n{}", outcome(find(&Space::from([start]).budget(budget), pattern)))?;
    }
    Ok(umpire_testing::Files::from([("messages.txt".into(), out)]))
}
