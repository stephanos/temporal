use std::fmt::Write;

use umpire_dsl::{Action, Machine, Progress, Scope, Spell, State, invariant, machine, on, postcondition, progress, scope};

#[derive(State)]
struct Counter {
    #[starts(0..=1)]
    value: u8,
}

#[derive(Action)]
enum User { Increment, Reset }

#[derive(Action)]
enum Clock { #[internal] Tick }

#[machine(actions = (User, Clock))]
impl Counter {
    #[on(Clock::Tick)]
    fn tick(&mut self) -> Option<()> { self.value += 1; None }

    #[postcondition(after = User::Reset)]
    fn reset_clears(&self, _before: &Self) -> bool { self.value == 0 }

    #[on(User::Reset)]
    #[allow(clippy::unnecessary_wraps, reason = "transition handlers return Option even when always enabled")]
    fn reset(&mut self) -> Option<()> { self.value = 0; Some(()) }

    #[invariant]
    fn bounded(&self) -> bool { self.value <= 2 }

    #[scope]
    fn never_reset() -> Scope<CounterAction> {
        Scope::without(&[CounterAction::User(User::Reset)])
    }

    #[progress(within = Self::never_reset())]
    fn reaches_two() -> Progress<CounterWorld> {
        Progress::<CounterWorld>::new("replaced_name")
            .whenever(|_, s| (s.value == 1).into())
            .eventually(|_, s| (s.value == 2).into())
    }

    #[postcondition]
    fn grows_at_most_one(&self, before: &Self) -> bool { self.value <= before.value + 1 }

    #[on(User::Increment)]
    fn increment(&mut self) -> Option<()> {
        (self.value < 2).then(|| { self.value += 1; })
    }
}

fn spelled(value: &impl Spell) -> String {
    let mut out = String::new();
    value.spell(&mut out);
    out
}

mod public_state {
    use umpire_dsl::{Action, Machine, State, machine};

    #[derive(State)]
    pub struct Public {}

    #[derive(Action)]
    pub enum Nobody {}

    #[machine(actions = (Nobody,))]
    impl Public {}

    pub(super) fn starts() -> usize { PublicWorld.starts().len() }
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let world = CounterWorld;
    let mut output = String::new();
    writeln!(output, "public state starts: {}", public_state::starts())?;
    let mut direct = Counter { value: 0 };
    let enabled = direct.increment().is_some();
    writeln!(output, "direct transition: {enabled} {}", spelled(&direct))?;
    for state in world.starts() {
        state.spell(&mut output);
        output.push('\n');
        for action in world.actions() {
            action.spell(&mut output);
            write!(output, " internal: {}", world.internal(&action))?;
            let edges = world.step(&state, action.clone());
            write!(output, " edges: {}", edges.len())?;
            for edge in &edges {
                output.push_str(" next: ");
                edge.2.spell(&mut output);
                write!(output, " outcome: {} facts: {}", spelled(&edge.0), edge.1.len())?;
                for claim in world.postconditions() {
                    write!(output, " {}={:?}", claim.name, claim.holds(&world, &state, &action, edge))?;
                }
            }
            output.push('\n');
        }
    }
    let state = Counter { value: 2 };
    writeln!(output, "disabled increment: {}", world.step(&state, CounterAction::User(User::Increment)).len())?;
    writeln!(output, "original state: {}", spelled(&state))?;
    writeln!(output, "order: {}", world.claim_order().join(", "))?;
    for claim in world.invariants() {
        writeln!(output, "{}: {:?}", claim.name, (claim.predicate)(&world, &state))?;
    }
    for (name, scope) in world.scopes() {
        writeln!(output, "scope {name}: {}", spelled(&scope.excluded[0]))?;
    }
    for claim in world.progress() {
        writeln!(output, "progress {}: {:?} -> {:?}; without {}", claim.name,
            (claim.trigger)(&world, &Counter { value: 1 }),
            (claim.target)(&world, &state), spelled(&claim.scope.excluded[0]))?;
    }
    Ok(umpire_testing::Files::from([("values.txt".into(), output)]))
}
