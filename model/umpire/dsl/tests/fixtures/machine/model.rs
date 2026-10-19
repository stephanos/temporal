use std::fmt::Write;

use umpire_dsl::{Machine, Outcome, Spell, Verdict, machine};

#[machine]
mod counter {
use umpire_dsl::{Action, State, check};
use std::sync::atomic::{AtomicUsize, Ordering};

pub static PROGRESS_CALLS: AtomicUsize = AtomicUsize::new(0);

#[derive(State)]
pub struct Counter {
    #[starts(0..=1)]
    pub value: u8,
}

#[derive(Action)]
pub enum User { Increment, Reset, MutateRestore }

#[derive(Action)]
#[internal]
pub enum Clock { Tick }

#[transitions]
impl Counter {
    #[on(Clock::Tick)]
    fn tick(&mut self) { self.value += 1; }

    #[on(User::Reset)]
    fn reset(&mut self) { self.value = 0; }

    #[on(User::Increment)]
    pub fn increment(&mut self) {
        if self.value < 2 { self.value += 1; }
    }

    #[on(User::MutateRestore)]
    fn mutate_restore(&mut self) {
        if self.value < 2 {
            self.value += 1;
            self.value -= 1;
        }
    }

    fn helper(&self) -> bool { self.value <= 2 }
}

#[invariants]
impl Counter {
    pub fn bounded(&self) { check!(self.helper()); }
}

#[postconditions]
impl Counter {
    #[after(User::Reset)]
    pub fn reset_clears(&self) { check!(self.value == 0); }

    fn grows_at_most_one(&self, before: &Self) { check!(self.value <= before.value + 1); }
}

#[assumptions]
impl Counter {
    fn never_reset() -> Assumption {
        without([User::Reset])
    }
}

#[progress]
impl Counter {
    fn reaches_two(&self) -> Progress {
        PROGRESS_CALLS.fetch_add(1, Ordering::SeqCst);
        let trigger = self.value == 1;
        let progress = whenever(trigger);
        if self.value == 2 {
            return progress.eventually(true).assuming(Self::never_reset());
        }
        progress.eventually(false).assuming(Self::never_reset())
    }
}

impl Counter {
    pub fn unregistered(&self) -> bool { self.helper() }
}
}

use counter::{Counter, User};

fn spelled(value: &impl Spell) -> String {
    let mut out = String::new();
    value.spell(&mut out);
    out
}

fn judgment(verdict: Verdict) -> String {
    spelled(&match verdict {
        Verdict::Holds => Outcome::Verified,
        Verdict::Fails => Outcome::Refuted,
        Verdict::Unknown(_) => Outcome::Unknown,
    })
}

#[machine]
mod public_state {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub struct Public {}

    #[derive(Action)]
    pub enum Nobody {}

}

pub(crate) fn run() -> umpire_testing::RunResult {
    let world = counter::world();
    let mut output = String::new();
    writeln!(output, "public state starts: {}", public_state::world().starts().len())?;
    let mut direct = Counter { value: 0 };
    assert!(direct.unregistered());
    direct.bounded();
    direct.reset_clears();
    Counter { value: 2 }.reset_clears();
    let before = direct.clone();
    direct.increment();
    let enabled = direct != before;
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
                    write!(output, " {}={}", claim.name, judgment(claim.holds(&world, &state, &action, edge)))?;
                }
            }
            output.push('\n');
        }
    }
    let state = Counter { value: 2 };
    writeln!(output, "disabled increment: {}", world.step(&state, User::Increment.into()).len())?;
    writeln!(output, "original state: {}", spelled(&state))?;
    writeln!(output, "order: {}", world.claim_order().join(", "))?;
    for claim in world.invariants() {
        writeln!(output, "{}: {}", claim.name, judgment((claim.predicate)(&world, &state)))?;
    }
    for (name, assumption) in world.assumptions() {
        writeln!(output, "scope {name}: {}", spelled(&assumption.excluded[0]))?;
    }
    for claim in world.progress() {
        let evaluate = claim.evaluator.ok_or("missing generated progress evaluator")?;
        counter::PROGRESS_CALLS.store(0, std::sync::atomic::Ordering::SeqCst);
        let initial = evaluate(&world, &Counter { value: 1 });
        assert_eq!(counter::PROGRESS_CALLS.load(std::sync::atomic::Ordering::SeqCst), 1);
        let final_state = evaluate(&world, &state);
        assert_eq!(counter::PROGRESS_CALLS.load(std::sync::atomic::Ordering::SeqCst), 2);
        writeln!(output, "progress {}: {} -> {}; without {}", claim.name,
            judgment(initial.trigger),
            judgment(final_state.target), spelled(&initial.assumption.excluded[0]))?;
    }
    Ok(umpire_testing::Files::from([("values.txt".into(), output)]))
}
