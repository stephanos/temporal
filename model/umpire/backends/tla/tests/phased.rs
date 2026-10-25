//! Phase access, rejecting outcomes and closure-law admission, and their located refusals.
#![forbid(unsafe_code)]

use umpire_tla::Lift;

const JOB: &str = include_str!("../../../dev/fixtures/fixtures/phased_rejection/model.rs");

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
        vec![("impl umpire::dsl::Phase for Phase", "impl umpire::dsl::Finite for Phase")],
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

/// A handler with `returns` sequential conditional returns, each followed by an update that
/// the later conditions read, and a final update of `last`.
fn sequential_returns(returns: usize, last: &str) -> String {
    let body = "if self.count == 3 {\n                        reject!(Rejection::NotFound);\n                    }\n                    self.count += 1;\n                    "
        .repeat(returns);
    format!(
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
        pub {last}: bool,
    }}

    impl Model {{
        #[on(Client)]
        fn client(&mut self, action: Client) {{
            match action {{
                Client::Step => {{
                    {body}self.{last} = true;
                }}
            }}
        }}
    }}
}}
"
    )
}

/// The emitted step operator of a source.
fn step(source: &str) -> Result<String, Box<dyn std::error::Error>> {
    let module = umpire_tla::Tla::emit(&Lift::parse("gate.rs", source)?)?;
    let step = module.tla.lines().find(|line| line.starts_with("Client_Step ==")).ok_or("step")?;
    Ok(step.to_owned())
}

#[test]
fn emitted_size_is_linear_in_sequential_conditional_returns()
-> Result<(), Box<dyn std::error::Error>> {
    let [sixteen, thirty_two] = [16, 32].map(|returns| step(&sequential_returns(returns, "done")));
    let (sixteen, thirty_two) = (sixteen?, thirty_two?);
    assert!(thirty_two.len() < sixteen.len() * 2 + 1024, "{} {}", sixteen.len(), thirty_two.len());
    Ok(())
}

#[test]
fn shared_definitions_never_take_an_authored_name() -> Result<(), Box<dyn std::error::Error>> {
    let source = sequential_returns(0, "__umpire_l1").replace(
        "self.__umpire_l1 = true;",
        "if self.count == 2 { reject!(Rejection::NotFound); } let __umpire_l_1_1 = self.count + 1; self.count = __umpire_l_1_1; self.__umpire_l1 = true;",
    );
    let step = step(&source)?;
    assert!(step.contains("__umpire_l_2_1 == "), "{step}");
    assert!(!step.contains("__umpire_l1 =="), "{step}");
    assert!(!step.contains("__umpire_l_1_1 =="), "{step}");
    if std::env::var("UMPIRE_BACKENDS").as_deref() == Ok("require") {
        use umpire_tlc::{State, Value};
        let module = umpire_tla::Tla::emit(&Lift::parse("collision.rs", &source)?)?;
        let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
        let result = umpire_tlc::Tlc::discover(&root)?.check_module(&module)?;
        let expected = [(0, false), (1, true), (2, true)]
            .map(|(count, done)| {
                State::from([
                    ("count".into(), Value::Int(count)),
                    ("__umpire_l1".into(), Value::Bool(done)),
                ])
            })
            .into_iter()
            .collect();
        assert_eq!(result.states, expected);
    }
    Ok(())
}

#[path = "fixtures/linear_returns.rs"]
mod compiled;

#[test]
fn large_handlers_compile_with_their_admitted_data() -> Result<(), Box<dyn std::error::Error>> {
    use umpire_dsl::__lifted::Lifts;
    let embedded = compiled::linear::World::__umpire_lifted();
    let tree = embedded.model.map_err(|error| format!("{error:?}"))?;
    assert_eq!(tree.actions.len(), 2);
    assert!(tree.actions.iter().all(|action| !action.branches[0].definitions.is_empty()));
    Ok(())
}

#[test]
#[allow(clippy::print_stderr)]
fn admitted_tree_and_constructor_data_grow_linearly() -> Result<(), Box<dyn std::error::Error>> {
    use umpire_dsl::__lifted::print::Print;
    for pattern in ["returns", "updates", "vectors"] {
        let mut sizes = Vec::new();
        for count in [8, 16, 32] {
            let mut source = sequential_returns(count, "done");
            if pattern != "returns" {
                let body = "if self.count == 3 { reject!(Rejection::NotFound); } self.count += 1;";
                source = sequential_returns(0, "done").replace(
                    "self.done = true;",
                    &format!("{body} {} self.done = true;", "self.count += 1;".repeat(count)),
                );
            }
            if pattern == "vectors" {
                source = source
                    .replace("pub count: u8,", "pub count: u8, pub values: imbl::Vector<u8>,")
                    .replace("Client::Step =>", "Client::Step if self.values.len() < 64 =>")
                    .replace("self.count += 1;", "self.values.push_back(1);");
            }
            let started = std::time::Instant::now();
            let tree = Lift::parse("linear.rs", &source)?;
            let nodes = format!("{tree:?}").matches("Expr {").count();
            let bytes = tree.print().to_string().len();
            eprintln!(
                "linear pattern={pattern} count={count}: nodes={nodes} constructor_bytes={bytes} elapsed={:?}",
                started.elapsed()
            );
            sizes.push((nodes, bytes));
        }
        for pair in sizes.windows(2) {
            assert!(pair[1].0 <= pair[0].0 * 2 + 32, "nodes: {sizes:?}");
            assert!(pair[1].1 <= pair[0].1 * 2 + 4096, "embedded data: {sizes:?}");
        }
    }
    Ok(())
}
