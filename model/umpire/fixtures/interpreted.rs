//! Controls of the differential check outside the fixture folders: the lifter's and the
//! backends' own admitted Models, checked arithmetic, lazy start domains, length arithmetic, the
//! claim cap, seeded mutant trees that must disagree, and a first timing of interpreted against
//! compiled exploration.

#[path = "fixtures/sample_kettle/model.rs"]
#[allow(unreachable_pub, missing_docs)]
mod sample_kettle;

#[path = "../lift/tests/fixtures/linear_returns.rs"]
#[allow(unreachable_pub, missing_docs)]
mod linear_returns;
#[path = "../lift/tests/fixtures/remainder_i16.rs"]
#[allow(unreachable_pub, missing_docs)]
mod remainder_i16;
#[path = "../lift/tests/fixtures/remainder_i8.rs"]
#[allow(unreachable_pub, missing_docs)]
mod remainder_i8;
#[path = "../lift/tests/fixtures/sequential_checks.rs"]
#[allow(unreachable_pub, missing_docs)]
mod sequential_checks;
#[path = "../lift/tests/fixtures/shadowing.rs"]
#[allow(unreachable_pub, missing_docs)]
mod shadowing;
#[path = "../lift/tests/fixtures/vector.rs"]
#[allow(unreachable_pub, missing_docs)]
mod vector;
#[path = "../lift/tests/fixtures/vector_fallback.rs"]
#[allow(unreachable_pub, missing_docs)]
mod vector_fallback;

#[path = "../backends/apalache/tests/fixtures/arithmetic.rs"]
#[allow(unreachable_pub, missing_docs)]
mod apalache_arithmetic;
#[path = "../backends/apalache/tests/fixtures/categories.rs"]
#[allow(unreachable_pub, missing_docs)]
mod apalache_categories;
#[path = "../backends/apalache/tests/fixtures/empty_condition.rs"]
#[allow(unreachable_pub, missing_docs)]
mod apalache_empty_condition;
#[path = "../backends/apalache/tests/fixtures/arithmetic_prefix.rs"]
#[allow(unreachable_pub, missing_docs)]
mod arithmetic_prefix;
#[path = "crowded.rs"]
#[allow(unreachable_pub, missing_docs)]
mod many_claims;
#[path = "../backends/apalache/tests/fixtures/replay.rs"]
#[allow(unreachable_pub, missing_docs)]
mod replay;
#[path = "../backends/tlc/tests/fixtures/scaling_model.rs"]
#[allow(unreachable_pub, missing_docs)]
mod scaling_model;
#[path = "fixtures/shared_returns/model.rs"]
#[allow(unreachable_pub, missing_docs)]
mod shared_returns;
#[path = "../backends/apalache/tests/fixtures/start_empty_first.rs"]
#[allow(unreachable_pub, missing_docs)]
mod start_empty_first;
#[path = "../backends/apalache/tests/fixtures/start_fault_first.rs"]
#[allow(unreachable_pub, missing_docs)]
mod start_fault_first;
#[path = "../backends/apalache/tests/fixtures/start_later_fault.rs"]
#[allow(unreachable_pub, missing_docs)]
mod start_later_fault;
#[path = "lengths.rs"]
#[allow(unreachable_pub, missing_docs)]
mod usize_lengths;

use sample_kettle::kettle;
use umpire_checker::{Checker, Interpreted};
use umpire_dsl::__lifted::{
    Lifted, Lifts,
    expr::{Expr, ExprKind},
    tree::Start,
};
use umpire_testing::{TestResult, compare, interpreted};

/// Every compiled failure of checked arithmetic, which the interpreter must spell as Rust does.
#[allow(clippy::assign_op_pattern, reason = "the subset admits only +=, -= and *=")]
#[umpire_dsl::machine]
pub mod arithmetic {
    use umpire_dsl::{Action, State};

    /// Operands near the ends of their types.
    #[derive(State)]
    pub struct Model {
        /// Overflows upward and downward.
        #[starts(250..=255)]
        pub count: u8,
        /// Divides by zero and by minus one.
        #[starts(-1..=1)]
        pub divisor: i8,
        /// Starts at and next to `i8::MIN`.
        #[starts(-128..=-127)]
        pub low: i8,
    }

    /// One operation each.
    #[derive(Action, Copy)]
    pub enum User {
        /// `count += 1`.
        Add,
        /// `count -= 251`.
        Subtract,
        /// `low *= 2`.
        Multiply,
        /// `low / divisor`.
        Divide,
        /// `low % divisor`.
        Remainder,
        /// `-low`.
        Negate,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Add => self.count += 1,
                User::Subtract => self.count -= 251,
                User::Multiply => self.low *= 2,
                User::Divide => self.low = self.low / self.divisor,
                User::Remainder => self.low = self.low % self.divisor,
                User::Negate => self.low = -self.low,
            }
        }
    }
}

#[test]
fn lifter_controls_agree() -> TestResult {
    let budget = Checker::BUDGET;
    interpreted(|| linear_returns::linear::World, None, budget, None)?;
    interpreted(|| remainder_i16::remainder::World, None, budget, None)?;
    interpreted(|| remainder_i8::remainder::World, None, budget, None)?;
    interpreted(|| sequential_checks::sequential::World, None, budget, None)?;
    interpreted(|| shadowing::lexical::World, None, budget, None)?;
    interpreted(|| vector::collection::World, None, budget, None)?;
    interpreted(|| vector_fallback::fallback::World, None, budget, None)?;
    Ok(())
}

#[test]
fn backend_controls_agree() -> TestResult {
    let budget = Checker::BUDGET;
    for mode in 0..=10 {
        for start in [-128, -1, 0, 1, 127] {
            for divisor in [-1, 0, 1, 2] {
                interpreted(
                    || apalache_arithmetic::arithmetic::World::new(mode, start, divisor),
                    None,
                    budget,
                    None,
                )
                .map_err(|error| format!("arithmetic World {mode} {start} {divisor}: {error}"))?;
            }
        }
    }
    for divisor in 0..=2 {
        for enabled in [false, true] {
            interpreted(
                || apalache_empty_condition::empty_condition::World::new(divisor, enabled),
                None,
                budget,
                None,
            )?;
        }
    }
    for limit in [2, 3] {
        interpreted(|| replay::inputs::World::new(limit), None, budget, None)?;
    }
    interpreted(|| arithmetic_prefix::prefix::World, None, budget, None)?;
    interpreted(|| apalache_categories::categories::World, None, budget, None)?;
    interpreted(|| scaling_model::counter::World, None, budget, None)?;
    interpreted(|| shared_returns::gate::World::new(2), None, budget, None)?;
    Ok(())
}

/// A start domain is evaluated only once every earlier field has a value, as the compiled
/// nested loops do: an empty earlier domain skips a later failing one, and a failing domain
/// fails the same way on both sides.
#[test]
fn start_domains_evaluate_lazily() -> TestResult {
    let budget = Checker::BUDGET;
    let agreement = interpreted(|| start_empty_first::empty_first::World, None, budget, None)?;
    assert_eq!(agreement.states, 0);
    interpreted(|| start_fault_first::faulty_first::World, None, budget, None)?;
    interpreted(|| start_later_fault::later_fault::World, None, budget, None)?;
    Ok(())
}

/// A sum or product of a length is refused at its span, so no back end reads it; the Model
/// still checks natively.
#[test]
fn length_sums_and_products_are_refused() -> TestResult {
    let refusal = usize_lengths::lengths::World::__umpire_lifted()
        .model
        .err()
        .ok_or("a length product was admitted")?;
    assert!(
        refusal.message.starts_with("`+` and `*` on a Vector length are refused"),
        "{refusal:?}"
    );
    assert_eq!((refusal.line, refusal.column), (40, 20), "{refusal:?}");
    let report = Checker::explore(&usize_lengths::lengths::world(), Checker::BUDGET)?
        .verify_all()?
        .to_string();
    assert!(report.contains("outcome: Verified"), "{report}");
    Ok(())
}

/// A difference, quotient or remainder of lengths means the same compiled, interpreted and in
/// TLC, including a difference below zero, which every back end reports in the same state.
#[test]
#[allow(clippy::print_stderr)]
fn length_differences_agree_on_every_back_end() -> TestResult {
    use usize_lengths::{differences, underflow};
    interpreted(differences::world, None, Checker::BUDGET, None)?;
    interpreted(underflow::world, None, Checker::BUDGET, None)?;
    let native = Checker::explore(&differences::world(), Checker::BUDGET)?.verify_all()?;
    let verdicts: Vec<_> = native
        .claims
        .iter()
        .map(|claim| (claim.name.clone(), format!("{:?}", claim.outcome)))
        .collect();
    assert_eq!(
        verdicts,
        [("halves".into(), "Verified".into()), ("below_two".into(), "Refuted".into())]
    );
    let report = Checker::explore(&underflow::world(), Checker::BUDGET)?.verify_all()?.to_string();
    assert!(report.contains("attempt to subtract with overflow"), "{report}");
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        eprintln!(
            "length controls checked natively; no TLC evidence (set UMPIRE_BACKENDS=require)"
        );
        return Ok(());
    }
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../..");
    let tlc = umpire_tlc::Tlc::discover(&root)?;
    // rustc names this `#[path]` module's file relative to the workspace, not to this test.
    let source = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("lengths.rs");
    let lift = |mut embedded: umpire_dsl::__lifted::Embedded| {
        embedded.file = source.display().to_string();
        umpire_testing::lift(Some(&embedded), &source)
    };
    let module = lift(differences::World::__umpire_lifted())?;
    let result = tlc.check_module(&module)?;
    assert_eq!(result.states.len(), 4);
    let tlc_verdicts: Vec<_> = ["Invariant_halves", "Invariant_below_two"]
        .iter()
        .map(|operator| result.properties.get(*operator).map(|property| property.verdict))
        .collect();
    assert_eq!(
        tlc_verdicts,
        [Some(umpire_tlc::Verdict::Verified), Some(umpire_tlc::Verdict::Refuted)]
    );
    let module = lift(underflow::World::__umpire_lifted())?;
    let Err(umpire_tlc::Error::Evaluation(error)) = tlc.check_module(&module) else {
        return Err("TLC did not report the length underflow as an evaluation error".into());
    };
    assert_eq!(error.property.as_deref(), Some("Invariant_remaining"), "{error}");
    assert!(error.message.contains("checked Rust integer overflow"), "{error}");
    assert!(
        error.state.as_deref().is_some_and(|state| state.contains("values = <<1, 1>>")),
        "{error}"
    );
    Ok(())
}

#[test]
fn checked_arithmetic_fails_as_compiled() -> TestResult {
    interpreted(arithmetic::world, None, Checker::BUDGET, None)?;
    let files = umpire_testing::render(&Checker::explore(&arithmetic::world(), Checker::BUDGET)?)?;
    let failures: String = files.values().map(String::as_str).collect();
    for failure in [
        "attempt to add with overflow",
        "attempt to subtract with overflow",
        "attempt to multiply with overflow",
        "attempt to divide by zero",
        "attempt to divide with overflow",
        "attempt to calculate the remainder with a divisor of zero",
        "attempt to calculate the remainder with overflow",
        "attempt to negate with overflow",
    ] {
        assert!(failures.contains(failure), "the control never fails with {failure}");
    }
    Ok(())
}

fn kettle_tree() -> Result<Lifted, String> {
    kettle::World::__umpire_lifted().model.map_err(|refusal| refusal.message)
}

/// Applies `change` to every node of `expr`.
fn rewrite(expr: &mut Expr, change: &mut dyn FnMut(&mut ExprKind)) {
    change(&mut expr.kind);
    let children: Vec<&mut Expr> = match &mut expr.kind {
        ExprKind::Unary(_, value) | ExprKind::Length(value) | ExprKind::Check(_, value) => {
            vec![value]
        }
        ExprKind::Binary(_, left, right) | ExprKind::Append(left, right) => vec![left, right],
        ExprKind::If(first, second, third) | ExprKind::GetOr(first, second, third) => {
            vec![first, second, third]
        }
        ExprKind::Tuple(values) | ExprKind::Sequence(values) | ExprKind::Checks(values) => {
            values.iter_mut().collect()
        }
        _ => Vec::new(),
    };
    for child in children {
        rewrite(child, change);
    }
}

fn replace(expr: &mut Expr, from: i128, to: i128) {
    rewrite(expr, &mut |kind| {
        if *kind == ExprKind::Int(from) {
            *kind = ExprKind::Int(to);
        }
    });
}

fn disagreement<M>(world: impl Fn() -> M, mutant: &Lifted) -> Result<String, String>
where
    M: umpire_dsl::__lifted::Interprets + umpire_dsl::Values,
    M::Action: umpire_dsl::ActionValues<M>,
{
    let interpreted = Interpreted::from_tree(world(), mutant)?;
    compare(&world(), &interpreted, None, Checker::BUDGET, None)
        .err()
        .map(|error| error.to_string())
        .ok_or_else(|| "a mutant tree agreed with its compiled Model".to_owned())
}

/// Each seeded change to a tree explores differently from the compiled Model, and the
/// differential check names the file that shows it: an update, a postcondition, a start, a
/// progress target, an outcome, and a `check!` text.
#[test]
fn seeded_mutant_trees_disagree() -> TestResult {
    let mut heats_twice = kettle_tree()?;
    let heat = heats_twice
        .actions
        .iter_mut()
        .find(|action| action.name == "Heater::Heat")
        .ok_or("Kettle has no Heater::Heat")?;
    let branch = &mut heat.branches[0];
    for (_, definition) in &mut branch.definitions {
        replace(definition, 1, 2);
    }
    for update in branch.updates.values_mut() {
        replace(update, 1, 2);
    }
    let mut boils_early = kettle_tree()?;
    let claim = boils_early
        .postconditions
        .iter_mut()
        .find(|claim| claim.name == "fails_over_at_boiling")
        .ok_or("Kettle has no fails_over_at_boiling")?;
    replace(&mut claim.expr, 100, 99);
    let mut starts_heating = kettle_tree()?;
    let heating = starts_heating
        .fields
        .iter_mut()
        .find(|field| field.name == "heating_on")
        .ok_or("Kettle has no heating_on")?;
    heating.start = Start::Value(Expr::bool(true));
    let mut never_boils = kettle_tree()?;
    let boils =
        never_boils.progress.iter_mut().find(|claim| claim.name == "boils").ok_or("no boils")?;
    boils.target = Expr::bool(false);
    for (mutant, expected) in [
        (heats_twice, "reached: 286"),
        (boils_early, "interpreted: outcome: Refuted"),
        (starts_heating, "states.txt line 1:"),
        (never_boils, "paths/boils.txt"),
    ] {
        let found = disagreement(kettle::world, &mutant)?;
        assert!(found.contains(expected), "{found}");
    }

    let gate = || shared_returns::gate::World::new(2);
    let mut not_found =
        shared_returns::gate::World::__umpire_lifted().model.map_err(|r| r.message)?;
    let bump = not_found
        .actions
        .iter_mut()
        .find(|action| action.name == "Client::Bump")
        .ok_or("no Client::Bump")?;
    for branch in &mut bump.branches {
        rewrite(&mut branch.outcome, &mut |kind| {
            if *kind == ExprKind::Enum("Outcome::Rejected(Unavailable)".into()) {
                *kind = ExprKind::Enum("Outcome::Rejected(NotFound)".into());
            }
        });
    }
    let found = disagreement(gate, &not_found)?;
    assert!(
        found.starts_with("interpreted exploration disagrees with compiled: rows.txt"),
        "{found}"
    );
    assert!(found.contains("outcome=Rejected(NotFound)"), "{found}");

    let mut misspelled =
        vector::collection::World::__umpire_lifted().model.map_err(|r| r.message)?;
    let never_two = misspelled
        .invariants
        .iter_mut()
        .find(|claim| claim.name == "never_two")
        .ok_or("no never_two")?;
    rewrite(&mut never_two.expr, &mut |kind| {
        if let ExprKind::Check(index, _) = kind {
            *index = 0;
        }
    });
    let found = disagreement(|| vector::collection::World, &misspelled)?;
    assert!(found.contains("paths/never_two.txt line"), "{found}");
    assert!(found.contains("check!(self.values.len() <= 2)"), "{found}");
    Ok(())
}

/// `#[machine]` registers exactly one claim callback per claim, so a Model with more claims
/// than a fixed table would hold still interprets and agrees; a tree whose claims the compiled
/// Model does not register is refused.
#[test]
fn every_claim_has_its_own_callback() -> TestResult {
    let agreement = interpreted(many_claims::crowded::world, None, Checker::BUDGET, None)?;
    assert_eq!(agreement.states, 3);
    let mut crowded = kettle_tree()?;
    crowded.invariants = vec![crowded.invariants[0].clone(); 2];
    let Err(refusal) = Interpreted::from_tree(kettle::world(), &crowded) else {
        return Err("a tree with an unregistered invariant was interpreted".into());
    };
    assert_eq!(refusal, "the tree has 2 invariants; the compiled Model registers 1");
    Ok(())
}

/// One timed exploration.
fn timed<N: umpire_dsl::Machine>(
    world: &N,
    starts: &[N::State],
) -> Result<std::time::Duration, Box<dyn std::error::Error>> {
    let started = std::time::Instant::now();
    let graph =
        Checker::explore_within(world, starts.to_vec(), Checker::BUDGET, Checker::CATALOG_LIMIT)?;
    let elapsed = started.elapsed();
    std::hint::black_box(graph.counts);
    Ok(elapsed)
}

/// Release timing of interpreted against compiled exploration (`Checker::explore_within`,
/// claims and progress included) on Kettle and on Kettle's wide shape, 2,001 starts. Runs
/// alternate which side goes first; the report gives medians and their ratio, and with
/// `UMPIRE_TIMING_OUT` set it is also written to that file. `UMPIRE_TIMING_SCALE` multiplies the
/// repetitions. Run alone:
/// `cargo test --release -p umpire-fixtures --test interpreted -- --ignored --nocapture timing`.
#[test]
#[ignore = "timing; run alone in release"]
#[allow(clippy::print_stderr, clippy::cast_precision_loss)]
fn timing() -> TestResult {
    use std::fmt::Write;
    use umpire_dsl::Machine;
    let world = kettle::world();
    let interpreted = Interpreted::new(kettle::world())?;
    let scale: usize =
        std::env::var("UMPIRE_TIMING_SCALE").ok().and_then(|s| s.parse().ok()).unwrap_or(1);
    let wide: Vec<_> = (-1000..=1000)
        .map(|temperature| kettle::Model { temperature, heating_on: false, beeping: false })
        .collect();
    let mut report = String::new();
    for (shape, starts, repetitions) in
        [("kettle", world.starts(), 2_000 * scale), ("wide", wide, 200 * scale)]
    {
        compare(&world, &interpreted, Some(starts.clone()), Checker::BUDGET, None)?;
        let mut compiled = Vec::with_capacity(repetitions);
        let mut evaluated = Vec::with_capacity(repetitions);
        for run in 0..repetitions {
            if run % 2 == 0 {
                compiled.push(timed(&world, &starts)?);
                evaluated.push(timed(&interpreted, &starts)?);
            } else {
                evaluated.push(timed(&interpreted, &starts)?);
                compiled.push(timed(&world, &starts)?);
            }
        }
        compiled.sort();
        evaluated.sort();
        let (compiled, evaluated) = (compiled[repetitions / 2], evaluated[repetitions / 2]);
        writeln!(
            report,
            "{shape}: {repetitions} interleaved runs, median compiled {compiled:?}, interpreted {evaluated:?}, ratio {:.2}",
            evaluated.as_secs_f64() / compiled.as_secs_f64()
        )?;
    }
    eprint!("{report}");
    if let Ok(path) = std::env::var("UMPIRE_TIMING_OUT") {
        std::fs::write(path, report)?;
    }
    Ok(())
}
