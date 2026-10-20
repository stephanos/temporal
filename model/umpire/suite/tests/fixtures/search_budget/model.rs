use std::fmt::Write;

use umpire_dsl::{Gap, Machine, Registered, Steps, Waypoint};
use umpire_suite::{Run, Space, find, path};

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
        let targets = match (action, state.at) {
            (Action::Turn, 0..=2) => vec![(state.at + 1) % 3],
            (Action::Turn, 4) => vec![0],
            (Action::Turn, 10) => vec![11, 12],
            (Action::Turn, 11) => vec![13],
            (Action::Turn, 12 | 13) => vec![14],
            (Action::Turn, 14) => vec![15],
            (Action::Escape, 2) if state.open => vec![3],
            _ => Vec::new(),
        };
        targets.into_iter().map(|at| ((), Vec::new(), Node { at, open: state.open })).collect()
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
    let entry = Node { at: 4, open: false };
    write!(out, "unreachable, bounded by a billion steps, budget 4:\n{}", outcome(find(&Space::from([entry.clone()]).budget(4), path![entry, ..=1_000_000_000, false])))?;
    let long = Waypoint::Is(start.clone()).then_judged(Gap::AtMost(1), |node| node.at == 1);
    let long = (2..=12).fold(long, |path, index| path.then_judged(Gap::AtMost(1), move |node| u32::from(node.at) == index % 3));
    let answer = find(&space, long);
    writeln!(out, "twelve steps around three Model states, budget 3: {}", answer.map_or_else(|error| format!("refused: {error}"), |path| format!("{} steps", path.steps.len())))?;
    let along = Run::along_with_budget(path![start, ..=1, |n: &Node| n.at == 1, ..=1, |n| n.at == 2, ..=1, |n| n.at == 0], 3);
    write!(out, "along one lap, budget 3:\n{}", outcome(along))?;
    for budget in [3, 4] {
        let start = Node { at: 0, open: true };
        let pattern = path![start.clone(), ..=1, |n: &Node| n.at == 1 || n.at == 3, ..=1, |n| n.at == 2 || n.at == 3, ..=1, |n| n.at == 0 || n.at == 3, ..=1, |n| n.at == 1 || n.at == 3];
        write!(out, "loop witness can hide shorter escape, budget {budget}:\n{}", outcome(find(&Space::from([start]).budget(budget), pattern)))?;
    }
    let start = Node { at: 0, open: true };
    write!(out, "loop witness at the cut frontier, budget 3:\n{}", outcome(find(&Space::from([start.clone()]).budget(3), path![start, ..=1, |n: &Node| n.at == 1, ..=1, |n| n.at == 2, ..=1, |n| n.at == 0])))?;
    let starts = Space::from([Node { at: 0, open: false }, Node { at: 1, open: false }]).budget(1);
    write!(out, "zero-step answer despite start cut:\n{}", outcome(find(&starts, path![|n: &Node| n.at == 0, ..=0, true])))?;
    write!(out, "omitted start cannot prove absence:\n{}", outcome(find(&starts, path![|n: &Node| n.at == 1, ..=0, true])))?;
    let space = Space::from([Node { at: 10, open: false }]);
    write!(out, "shorter queued match survives a later younger arrival:\n{}", outcome(find(&space, path![|n: &Node| n.at == 10 || n.at == 13, ..=3, |n| n.at == 15])))?;
    Ok(umpire_testing::Files::from([("messages.txt".into(), out)]))
}
