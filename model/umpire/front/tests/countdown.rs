//! Imported countdown admission, typed frames and located refusal controls.
#![forbid(unsafe_code)]

use std::fmt::Write;

const SOURCE: &str = r"
#[machine] mod clock {
    use umpire_dsl::{Action, State, countdown, check};
    use umpire_dsl::countdown::{Ticks, Remaining};
    #[derive(State)] struct Model { configured: Option<Ticks<3>>, left: Option<Remaining<3>> }
    #[derive(Action)] enum User { Arm, Tick }
    impl Model {
        #[on(User)] fn user(&mut self, action: User) {
            match action {
                User::Arm => { self.left = countdown::arm(self.configured); },
                User::Tick => { self.decrement(); }
            }
        }
        #[effect] fn decrement(&mut self) {
            if countdown::armed(self.left) { self.left = countdown::tick(self.left); }
        }
        #[invariant] fn bounds(&self) { countdown::within(self.left, self.configured); }
        #[invariant] fn expiry(&self) { check!(!countdown::expired(self.left) || countdown::armed(self.left)); }
        #[postcondition] fn descent(&self, before: &Self) {
            if self.configured == before.configured { countdown::descends(self.left, before.left); }
        }
    }
}";

fn admit(source: &str) -> Result<umpire_front::Lifted, String> {
    let ast = syn::parse_file(source).map_err(|e| e.to_string())?;
    umpire_front::read(&ast, &[]).map(|(model, _, _)| model).map_err(|e| format!("{e:?}"))
}

#[test]
fn library_functions_and_laws_are_admitted_into_ordinary_ir() -> Result<(), String> {
    let model = admit(SOURCE)?;
    let umpire_front::Type::Optional(configured) = &model.fields[0].ty else {
        panic!("configured")
    };
    let umpire_front::Type::Optional(remaining) = &model.fields[1].ty else { panic!("remaining") };
    assert_eq!((configured.low, configured.high), (1, 3));
    assert_eq!((remaining.low, remaining.high), (0, 3));
    assert_ne!(configured.name, remaining.name);
    let body = format!("{model:?}");
    assert!(body.contains("Sub"), "{body}");
    assert!(body.contains("Checks"), "{body}");
    Ok(())
}

#[test]
fn constructors_bounds_and_explicit_call_arguments_preserve_identity() -> Result<(), String> {
    for bound in [1, 3, 255] {
        let source = SOURCE.replace("<3>", &format!("<{bound}>"));
        admit(&source.replace(
            "countdown::arm(self.configured)",
            &format!("countdown::arm::<{bound}>(self.configured)"),
        ))?;
    }
    for expression in [
        "Some(Remaining::<3> { ticks: 0 })",
        "Some(::umpire_dsl::countdown::Remaining::<3> { ticks: 0 })",
        "countdown::arm(Some(::umpire_dsl::countdown::Ticks::<3>(2)))",
        "countdown::tick::<3>(None)",
        "countdown::arm::<3>(Some(Ticks::<3>(2)))",
        "countdown::arm(None)",
    ] {
        admit(&SOURCE.replace("countdown::arm(self.configured)", expression))?;
    }
    for predicate in ["Ticks::<3>::contains(2)", "Remaining::<3>::contains(0)"] {
        admit(&SOURCE.replace("!countdown::expired(self.left)", predicate))?;
    }
    admit(&SOURCE.replace(
        "countdown::descends(self.left, before.left)",
        "countdown::descends(None, before.left)",
    ))?;
    Ok(())
}

#[test]
#[expect(clippy::too_many_lines, reason = "one table pins every refusal and its source location")]
fn unsupported_forms_have_located_refusal_goldens() {
    let cases = [
        ("zero", SOURCE.replace("<3>", "<0>")),
        ("overflow", SOURCE.replace("<3>", "<256>")),
        ("nested", SOURCE.replace("Option<Ticks<3>>", "Option<Option<Ticks<3>>>")),
        (
            "alias-module",
            SOURCE.replace("State, countdown, check", "State, countdown as clock, check"),
        ),
        ("alias-type", SOURCE.replace("{Ticks, Remaining}", "{Ticks as Budget, Remaining}")),
        (
            "glob",
            SOURCE.replace(
                "use umpire_dsl::countdown::{Ticks, Remaining};",
                "use umpire_dsl::countdown::*;",
            ),
        ),
        ("qualified-macro", SOURCE.replace("check!", "umpire_dsl::check!")),
        (
            "wrong-argument",
            SOURCE.replace("countdown::arm(self.configured)", "countdown::arm(true)"),
        ),
        (
            "wrong-count",
            SOURCE.replace(
                "countdown::arm(self.configured)",
                "countdown::arm(self.configured, self.left)",
            ),
        ),
        (
            "wrong-nominal",
            SOURCE.replace("countdown::arm(self.configured)", "countdown::arm(self.left)"),
        ),
        (
            "different-bound",
            SOURCE
                .replace("countdown::arm(self.configured)", "countdown::arm::<2>(self.configured)"),
        ),
        (
            "zero-call",
            SOURCE.replace("countdown::armed(self.left)", "countdown::armed::<0>(self.left)"),
        ),
        (
            "generic-count",
            SOURCE.replace("countdown::armed(self.left)", "countdown::armed::<3, 3>(self.left)"),
        ),
        (
            "generic-type",
            SOURCE.replace("countdown::armed(self.left)", "countdown::armed::<bool>(self.left)"),
        ),
        (
            "module-generic",
            SOURCE.replace("countdown::armed(self.left)", "countdown::<3>::armed(self.left)"),
        ),
        (
            "value-statement",
            SOURCE.replace(
                "countdown::within(self.left, self.configured);",
                "countdown::armed(self.left);",
            ),
        ),
        (
            "law-expression",
            SOURCE.replace(
                "!countdown::expired(self.left)",
                "countdown::within(self.left, self.configured)",
            ),
        ),
        (
            "alias-function",
            SOURCE.replace(
                "use umpire_dsl::countdown::{Ticks, Remaining};",
                "use umpire_dsl::countdown::{Ticks, Remaining, arm as enable};",
            ),
        ),
        (
            "shadowed-path",
            SOURCE.replace("{Ticks, Remaining}", "{Ticks, Remaining, arm}").replace(
                "self.left = countdown::arm(self.configured)",
                "let arm = true; self.left = arm(self.configured)",
            ),
        ),
        (
            "named-bound",
            SOURCE
                .replace("#[derive(State)]", "const N: u8 = 3; #[derive(State)]")
                .replace("Ticks<3>", "Ticks<N>"),
        ),
        (
            "imported-associated",
            SOURCE.replace("use umpire_dsl::countdown::{Ticks, Remaining};", "use umpire_dsl::countdown::{Ticks, Remaining}; use umpire_dsl::countdown::Ticks::contains;")
                .replace("!countdown::expired(self.left)", "contains::<3>(2)"),
        ),
    ];
    let mut observed = String::new();
    for (name, source) in cases {
        let ast = syn::parse_file(&source).unwrap_or_else(|e| panic!("{name}: {e}"));
        let error = match umpire_front::read(&ast, &[]) {
            Err(umpire_front::Refusal::Model(error)) => error,
            other => panic!("{name}: {other:?}"),
        };
        writeln!(observed, "{name}\t{}:{}\t{}", error.line, error.column, error.message)
            .unwrap_or_else(|e| panic!("{e}"));
    }
    assert_eq!(observed, include_str!("countdown-refusals.txt"));
}

#[test]
fn call_frames_isolate_consts_and_specializations_and_keep_check_indices() -> Result<(), String> {
    fn indices(expr: &umpire_front::expr::Expr, output: &mut Vec<usize>) {
        if let umpire_front::expr::ExprKind::Check(index, _) = expr.kind {
            output.push(index);
        }
        for child in expr.kind.children() {
            indices(child, output);
        }
    }

    let original = admit(SOURCE)?;
    let shadowed = admit(&SOURCE.replace(
        "#[derive(State)]",
        "const N: u8 = 0; const Some: bool = false; const None: bool = false; #[derive(State)]",
    ))?;
    assert_eq!(format!("{:?}", original.actions), format!("{:?}", shadowed.actions));
    assert_eq!(format!("{:?}", original.invariants), format!("{:?}", shadowed.invariants));
    let source = SOURCE.replace("left: Option<Remaining<3>>", "left: Option<Remaining<3>>, other: Option<Remaining<2>>")
        .replace("countdown::within(self.left, self.configured);", "countdown::within(self.left, self.configured); countdown::descends(self.other, self.other); countdown::within(self.left, self.configured);");
    let ast = syn::parse_file(&source).map_err(|e| e.to_string())?;
    let syn::Item::Mod(module) = &ast.items[0] else { return Err("module".into()) };
    let declared = umpire_front::declare::machine(&proc_macro2::TokenStream::new(), module)
        .map_err(|e| e.to_string())?;
    let (model, checks) =
        umpire_front::admit_checks(&declared, module).map_err(|e| format!("{e:?}"))?;
    let mut found = Vec::new();
    for claim in &model.invariants {
        indices(&claim.expr, &mut found);
    }
    for claim in &model.postconditions {
        indices(&claim.expr, &mut found);
    }
    found.sort_unstable();
    assert_eq!(found, (0..checks.len()).collect::<Vec<_>>());
    assert_eq!(checks.len(), 9);
    Ok(())
}

#[test]
fn qualified_library_calls_ignore_caller_value_bindings() -> Result<(), String> {
    let original = admit(SOURCE)?;
    for (name, call) in
        [("countdown", "countdown::arm"), ("umpire_dsl", "::umpire_dsl::countdown::arm")]
    {
        let model = admit(&SOURCE.replace(
            "self.left = countdown::arm(self.configured)",
            &format!("let {name} = false; self.left = {call}(self.configured)"),
        ))?;
        assert_eq!(original.actions[0].branches[0].updates, model.actions[0].branches[0].updates);
    }
    for constructor in [
        "countdown::arm(Some(countdown::Ticks::<3>(2)))",
        "Some(countdown::Remaining::<3> { ticks: 2 })",
    ] {
        let baseline = admit(&SOURCE.replace("countdown::arm(self.configured)", constructor))?;
        let shadowed = admit(&SOURCE.replace(
            "self.left = countdown::arm(self.configured)",
            &format!("let countdown = false; self.left = {constructor}"),
        ))?;
        assert_eq!(
            baseline.actions[0].branches[0].updates,
            shadowed.actions[0].branches[0].updates
        );
    }
    Ok(())
}

#[test]
fn explicit_closure_state_types_preserve_existing_admission() -> Result<(), String> {
    let source = include_str!("../../dev/fixtures/fixtures/phased_rejection/model.rs");
    for state in ["Model", "Self", "_", "self::Model"] {
        admit(&source.replace(
            "closable::terminal_remains(self, before)",
            &format!("closable::terminal_remains::<{state}>(self, before)"),
        ))?;
    }
    for (arguments, expected) in [
        ("Model, Model", "argument count"),
        ("3", "state type argument"),
        ("bool", "state type argument"),
        ("self::Model<3>", "state type argument"),
    ] {
        let source = source.replace(
            "closable::terminal_remains(self, before)",
            &format!("closable::terminal_remains::<{arguments}>(self, before)"),
        );
        let error = admit(&source).err().ok_or_else(|| format!("admitted {arguments}"))?;
        assert!(error.contains(expected), "{arguments}: {error}");
    }
    Ok(())
}
