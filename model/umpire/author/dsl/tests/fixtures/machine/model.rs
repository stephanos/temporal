use std::fmt::Write;

use umpire_dsl::{Machine, CheckOutcome, Verdict, machine};

#[machine]
mod counter {
use umpire_dsl::{Action, State, check, path};
use std::sync::atomic::{AtomicUsize, Ordering};

pub static PROGRESS_CALLS: AtomicUsize = AtomicUsize::new(0);

#[derive(State)]
pub struct Counter {
    #[starts(0..=1)]
    pub value: u8,
}

#[derive(Action, Copy)]
pub enum User { Increment, Reset, MutateRestore }

#[derive(Action, Copy)]
#[internal]
pub enum Clock { Tick }

mod helpers {
    pub(super) struct Counter { pub(super) value: u8 }

    impl std::ops::AddAssign<u8> for Counter {
        fn add_assign(&mut self, increment: u8) { self.value += increment; }
    }
}

struct Helper<T> { value: T }

impl<T: std::ops::AddAssign<u8>> Helper<T> {
    fn advance(&mut self)
    where Self: Sized {
        struct Increment(u8);
        impl Increment {
            fn advance(&mut self) { self.0 += 1; }
        }
        struct Carrier;
        trait Resolve { type Target; }
        impl Resolve for Carrier { type Target = Increment; }
        trait Types { type Assoc; }
        impl Types for Increment { type Assoc = u8; }
        trait Advance: Types { fn apply(&mut self); }
        impl Advance for <Carrier as Resolve>::Target
        where Self::Assoc: Copy {
            fn apply(&mut self) { self.advance(); }
        }
        let mut increment = Increment(0);
        increment.apply();
        self.value += increment.0;
    }
}

impl Counter {
    #[on(Clock)]
    fn advance(&mut self, action: Clock) {
        match action {
            Clock::Tick => { self.value += 1; }
        }
    }

    fn helper(&self) -> bool { self.value <= 2 }

    #[on(User)]
    pub fn dispatch(&mut self, action: User) {
        match action {
            User::Reset => { self.value = 0; }
            User::Increment if self.value < 2 => { self.increment(); }
            User::MutateRestore if self.value < 2 => {
                self.value += 1;
                self.value -= 1;
            }
            _ => {}
        }
    }
    #[effect]
    pub fn increment(&mut self) {
        let mut helper = Helper { value: helpers::Counter { value: self.value } };
        helper.advance();
        self.value = helper.value.value;
    }
    #[invariant]
    pub fn bounded(&self) { check!(self.helper()); }

    #[postcondition(after = User::Reset)]
    pub fn reset_clears(&self) { check!(self.value == 0); }

    #[postcondition]
    fn grows_at_most_one(&self, before: &Self) { check!(self.value <= before.value + 1); }

    #[assumption]
    fn never_reset() -> Assumption {
        without([User::Reset])
    }
    #[progress]
    fn reaches_two(&self) -> LeadsTo {
        PROGRESS_CALLS.fetch_add(1, Ordering::SeqCst);
        path![self.value == 1, ..=2, self.value == 2].assuming(Self::never_reset())
    }
    pub fn unregistered(&self) -> bool { self.helper() }
}
}

use counter::{Counter, User};

fn rendered(value: &impl std::fmt::Display) -> String {
    value.to_string()
}

fn judgment(verdict: Verdict) -> String {
    format!("{:?}", match verdict {
        Verdict::Holds => CheckOutcome::Verified,
        Verdict::Fails => CheckOutcome::Refuted,
        Verdict::Unknown(_) => CheckOutcome::Unknown,
    })
}

#[machine]
mod public_state {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub struct Public {}

    #[derive(Action, Copy)]
    pub enum Nobody {}

    impl Public {
        #[on(Nobody)]
        fn nobody(&mut self, action: Nobody) { match (self, action) {} }
    }
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
    writeln!(output, "direct transition: {enabled} {}", rendered(&direct))?;
    for state in world.starts() {
        write!(output, "{state}")?;
        output.push('\n');
        for action in world.actions() {
            write!(output, "{action}")?;
            write!(output, " internal: {}", world.internal(&action))?;
            let edges = world.step(&state, action.clone());
            write!(output, " edges: {}", edges.len())?;
            for edge in &edges {
                output.push_str(" next: ");
                write!(output, "{}", edge.2)?;
                write!(output, " outcome: {:?} facts: {}", edge.0, edge.1.len())?;
                for claim in world.postconditions() {
                    write!(output, " {}={}", claim.name, judgment(claim.holds(&world, &state, &action, edge)))?;
                }
            }
            output.push('\n');
        }
    }
    let state = Counter { value: 2 };
    writeln!(output, "disabled increment: {}", world.step(&state, User::Increment.into()).len())?;
    writeln!(output, "original state: {}", rendered(&state))?;
    writeln!(output, "order: {}", world.claim_order().join(", "))?;
    for claim in world.invariants() {
        writeln!(output, "{}: {}", claim.name, judgment((claim.predicate)(&world, &state)))?;
    }
    for (name, assumption) in world.assumptions() {
        writeln!(output, "scope {name}: {}", rendered(&assumption.excluded[0]))?;
    }
    for claim in world.progress() {
        let evaluate = claim.evaluator.ok_or("missing generated progress evaluator")?;
        counter::PROGRESS_CALLS.store(0, std::sync::atomic::Ordering::SeqCst);
        let initial = evaluate(&world, &Counter { value: 1 });
        assert_eq!(counter::PROGRESS_CALLS.load(std::sync::atomic::Ordering::SeqCst), 1);
        let final_state = evaluate(&world, &state);
        assert_eq!(initial.bound, Some(2));
        assert_eq!(final_state.bound, Some(2));
        assert_eq!(counter::PROGRESS_CALLS.load(std::sync::atomic::Ordering::SeqCst), 2);
        writeln!(output, "progress {}: {} -> {}; without {}", claim.name,
            judgment(initial.trigger),
            judgment(final_state.target), rendered(&initial.assumption.excluded[0]))?;
    }
    Ok(umpire_testing::Files::from([("values.txt".into(), output)]))
}
