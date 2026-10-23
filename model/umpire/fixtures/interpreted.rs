//! Controls of the differential check outside the fixture folders: the lifter's own admitted
//! Models, checked arithmetic, seeded mutant trees that must disagree, and a first timing of
//! interpreted against compiled exploration.

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

use sample_kettle::kettle;
use umpire_checker::{Checker, Interpreted};
use umpire_dsl::__lifted::{
    Lifted, Lifts,
    expr::{Expr, ExprKind},
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

fn replace(expr: &mut Expr, from: i128, to: i128) {
    let children: Vec<&mut Expr> = match &mut expr.kind {
        ExprKind::Int(value) if *value == from => {
            *value = to;
            Vec::new()
        }
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
        replace(child, from, to);
    }
}

/// A tree that heats two degrees at a time, and one whose claim expects 99 degrees, each
/// explore differently from the compiled Kettle; the differential check must say where.
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
    for (mutant, expected) in
        [(heats_twice, "reached: 286"), (boils_early, "interpreted: outcome: Refuted")]
    {
        let world = Interpreted::from_tree(kettle::world(), &mutant)?;
        let disagreement = compare(&kettle::world(), &world, None, Checker::BUDGET, None)
            .err()
            .ok_or("a mutant tree agreed with the compiled Kettle")?
            .to_string();
        assert!(disagreement.contains(expected), "{disagreement}");
    }
    Ok(())
}

/// A first timing of interpreted against compiled exploration on Kettle and on Kettle's wide
/// shape, 2,001 starts. Run alone, in release:
/// `cargo test --release -p umpire-fixtures --test interpreted -- --ignored --nocapture`.
#[test]
#[ignore = "timing; run alone in release"]
#[allow(clippy::print_stderr, clippy::cast_precision_loss)]
fn timing() -> TestResult {
    use umpire_dsl::Machine;
    let world = kettle::world();
    let interpreted = Interpreted::new(kettle::world())?;
    let wide: Vec<_> = (-1000..=1000)
        .map(|temperature| kettle::Model { temperature, heating_on: false, beeping: false })
        .collect();
    for (shape, starts, repetitions) in [("kettle", world.starts(), 2_000), ("wide", wide, 200)] {
        let mut compiled = Vec::with_capacity(repetitions);
        let mut evaluated = Vec::with_capacity(repetitions);
        for _ in 0..repetitions {
            let agreement =
                compare(&world, &interpreted, Some(starts.clone()), Checker::BUDGET, None)?;
            compiled.push(agreement.compiled);
            evaluated.push(agreement.interpreted);
        }
        compiled.sort();
        evaluated.sort();
        let (compiled, evaluated) = (compiled[repetitions / 2], evaluated[repetitions / 2]);
        eprintln!(
            "{shape}: {repetitions} runs, median compiled {compiled:?}, interpreted {evaluated:?}, ratio {:.2}",
            evaluated.as_secs_f64() / compiled.as_secs_f64()
        );
    }
    Ok(())
}
