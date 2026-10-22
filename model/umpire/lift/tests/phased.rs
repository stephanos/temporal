//! Phase access, rejecting outcomes and closure-law admission, and their located refusals.
use umpire_lift::Lift;

const JOB: &str = include_str!("../../tests/fixtures/phased_rejection/model.rs");

fn mutant(edits: &[(&str, &str)]) -> Result<String, String> {
    let mut source = JOB.to_owned();
    for (from, to) in edits {
        if source.matches(from).count() != 1 {
            return Err(format!("mutation target is not unique: {from}"));
        }
        source = source.replace(from, to);
    }
    Ok(source)
}

#[test]
fn unsupported_phase_rejection_and_law_forms_refuse_at_their_location()
-> Result<(), Box<dyn std::error::Error>> {
    let mut refusals = Vec::new();
    for edits in [
        // A trait other than Umpire's Phase and Phased.
        vec![("impl umpire_dsl::Phase for Phase", "impl umpire_dsl::Finite for Phase")],
        // Phased for anything but the State.
        vec![("impl Phased for Model", "impl Phased for Phase")],
        vec![("type Phase = Phase;", "type Phase = u8;")],
        vec![(
            "        fn phase(&self) -> Self::Phase {\n            self.phase\n        }\n",
            "        fn phase(&self) -> Self::Phase {\n            self.phase\n        }\n        const OPEN: bool = true;\n",
        )],
        // A library call that is not a closure law, and a law given a field for a state.
        vec![(
            "closable::terminal_remains(self, before);\n        }\n\n        #[postcondition]\n        fn closed_rejects",
            "closable::reopened(self, before);\n        }\n\n        #[postcondition]\n        fn closed_rejects",
        )],
        vec![(
            "closable::terminal_remains(self, before);\n        }\n\n        #[postcondition]\n        fn closed_rejects",
            "closable::terminal_remains(self, self.phase);\n        }\n\n        #[postcondition]\n        fn closed_rejects",
        )],
        vec![(
            "closable::closed_rejects(self, before, outcome, Rejection::NotFound);\n        }\n\n        #[postcondition(after = Client::Drop)]",
            "closable::closed_rejects(self, before, &Outcome::Accepted, Rejection::NotFound);\n        }\n\n        #[postcondition(after = Client::Drop)]",
        )],
        // A rejecting Model's outcome is the stock Outcome.
        vec![(
            "fn closed_rejects(&self, before: &Self, outcome: &Outcome)",
            "fn closed_rejects(&self, before: &Self, outcome: &())",
        )],
        // reject! takes one rejection, read by Umpire's own name.
        vec![(
            "reject!(Rejection::Unavailable)",
            "reject!(Rejection::Unavailable, Rejection::NotFound)",
        )],
        vec![("reject!(Rejection::Unavailable)", "umpire_dsl::reject!(Rejection::Unavailable)")],
        vec![("reject!(Rejection::Unavailable)", "reject!(Phase::Done)")],
        // Only Umpire's own path! is a path claim.
        vec![("path![self.phase == Phase::Queued", "crate::path![self.phase == Phase::Queued")],
        // An inherent method shadowing a Phased or Phase method that library bodies call.
        vec![(
            "    impl Model {\n        #[on(Worker)]",
            "    impl Model {\n        fn closed(&self) -> bool {\n            false\n        }\n\n        #[on(Worker)]",
        )],
        vec![(
            "    #[derive(Action, Copy)]\n    pub enum Worker {",
            "    impl Phase {\n        fn closed(&self) -> bool {\n            false\n        }\n    }\n\n    #[derive(Action, Copy)]\n    pub enum Worker {",
        )],
    ] {
        let source = mutant(&edits)?;
        let error = Lift::parse("phased.rs", &source)
            .err()
            .ok_or_else(|| format!("unsupported source was admitted: {edits:?}"))?;
        refusals.push(format!("{error}\n"));
    }
    assert_eq!(refusals.concat(), include_str!("expected/phased_refusals.txt"));
    Ok(())
}

#[test]
fn rejections_lower_to_typed_outcomes_and_kept_self_loops() -> Result<(), Box<dyn std::error::Error>>
{
    let lifted = Lift::parse("phased.rs", JOB)?;
    let outcomes = |action: &str| -> Vec<String> {
        lifted
            .actions
            .iter()
            .find(|candidate| candidate.name == action)
            .map(|action| {
                action.branches.iter().map(|branch| format!("{:?}", branch.outcome.kind)).collect()
            })
            .unwrap_or_default()
    };
    assert_eq!(
        outcomes("Worker::Claim"),
        [
            "Enum(\"Outcome::Rejected(NotFound)\")",
            "Enum(\"Outcome::Rejected(FailedPrecondition)\")",
            "Enum(\"Outcome::Accepted\")",
            "Enum(\"Outcome::Accepted\")",
        ]
    );
    let claim =
        lifted.actions.iter().find(|action| action.name == "Worker::Claim").ok_or("Claim")?;
    // Fixed rejections are kept self-loops; accepted branches still require a change.
    assert_eq!(
        claim.branches.iter().map(|branch| branch.preserve_self_loop).collect::<Vec<_>>(),
        [true, true, false, false]
    );
    // Drop's outcome depends on its helper: the first rejection wins over the caller's.
    let drop = lifted.actions.iter().find(|action| action.name == "Client::Drop").ok_or("Drop")?;
    let outcome = format!("{:?}", drop.branches[0].outcome);
    assert!(outcome.contains("FailedPrecondition") && outcome.contains("NotFound"), "{outcome}");
    assert!(!drop.branches[0].preserve_self_loop);
    // A comparison spelled like the macro is not a rejection.
    let source = mutant(&[("attempt != self.attempt", "attempt != self.attempt && !reject")])?;
    assert!(
        Lift::parse("phased.rs", &source)
            .is_err_and(|error| error.message.contains("unresolved name `reject`"))
    );
    Ok(())
}

/// The emitted step of a handler with `returns` sequential conditional returns, each followed
/// by an update that the later conditions read.
fn sequential_returns(returns: usize) -> Result<String, Box<dyn std::error::Error>> {
    let body = "if self.count == 3 {\n                        reject!(Rejection::NotFound);\n                    }\n                    self.count += 1;\n                    "
        .repeat(returns);
    let source = format!(
        r"use umpire_dsl::machine;

#[machine]
pub mod gate {{
    use umpire_dsl::{{Action, Rejection, State, reject}};

    #[derive(Action, Copy)]
    pub enum Client {{
        Step,
    }}

    #[derive(State)]
    pub struct Model {{
        pub count: u8,
        pub done: bool,
    }}

    impl Model {{
        #[on(Client)]
        fn client(&mut self, action: Client) {{
            match action {{
                Client::Step => {{
                    {body}self.done = true;
                }}
            }}
        }}
    }}
}}
"
    );
    let module = umpire_lift::Tla::emit(&Lift::parse("gate.rs", &source)?)?;
    let step = module.tla.lines().find(|line| line.starts_with("Client_Step ==")).ok_or("step")?;
    Ok(step.to_owned())
}

#[test]
fn emitted_size_is_linear_in_sequential_conditional_returns()
-> Result<(), Box<dyn std::error::Error>> {
    // Inlined, each return multiplied the step about thirteenfold (2,356 to 706,669,127 bytes
    // from one to six returns); shared, each further return adds the same few hundred bytes.
    let sizes = (3..=5)
        .map(|returns| sequential_returns(returns).map(|step| step.len()))
        .collect::<Result<Vec<_>, _>>()?;
    let [three, four, five] = sizes[..] else { return Err("three sizes".into()) };
    assert!((four - three).abs_diff(five - four) < 8 && five - four < 1_000, "{sizes:?}");
    Ok(())
}
