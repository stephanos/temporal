//! Build-time source admission and generated backend fixture bindings.

use std::path::{Path, PathBuf};

/// Generate emitted artifacts and typed fixture bindings from independently read source roots,
/// given each fixture's name, model module and folder. A fixture whose source declares a World
/// is bound and emitted at test time from its `tests.rs` hook's `bound()`.
#[allow(clippy::print_stdout)]
pub fn generate(fixtures: &[(String, String, PathBuf)], output: &Path) -> std::io::Result<()> {
    use std::fmt::Write;
    let mut source = String::from(
        "#[allow(clippy::unreadable_literal, reason = \"Generated exact integer bounds\")]\nfn lifted(name: &str) -> Result<umpire_lift::Module, String> { match name {\n",
    );
    let mut replay = String::from(
        "#[allow(unused_imports, unused_qualifications, unused_variables, clippy::match_single_binding)]\nfn check_backend(name: &str, folder: &std::path::Path, run: fn() -> umpire_testing::RunResult) -> umpire_testing::RunResult { match name {\n",
    );
    // A module evaluates each world path against its fixture's glob alone. Discovery never names
    // a fixture module `fixture_lifted`: `lifted` stays plain and only non-plain names escape.
    let mut bindings = String::from("mod fixture_lifted {\n");
    let mut admitted = Vec::new();
    for (name, identifier, folder) in fixtures {
        let input = folder.join("model.rs");
        println!("cargo:rerun-if-changed={}", input.display());
        let lifted = crate::Lift::source(&input);
        if let Ok(lifted) = &lifted
            && lifted.requires_binding()
        {
            let input = std::fs::canonicalize(&input)?.to_string_lossy().into_owned();
            admitted.push(name.clone());
            writeln!(source, "{name:?} => {identifier}_tests::bound().lift({input:?}),")
                .map_err(std::io::Error::other)?;
            writeln!(replay, "{name:?} => {identifier}_tests::bound().check(folder),")
                .map_err(std::io::Error::other)?;
            continue;
        }
        match lifted.and_then(|lifted| crate::Tla::emit(&lifted)) {
            Ok(module) => {
                admitted.push(name.clone());
                let tla = output.join(format!("{name}.tla"));
                let cfg = output.join(format!("{name}.cfg"));
                std::fs::write(&tla, &module.tla)?;
                std::fs::write(&cfg, &module.cfg)?;
                writeln!(source, "{name:?} => Ok(umpire_lift::Module {{ name: {:?}.into(), tla: include_str!({:?}).into(), cfg: include_str!({:?}).into(), metadata: {} }}),", module.name, tla.to_string_lossy(), cfg.to_string_lossy(), metadata(&module.metadata)).map_err(std::io::Error::other)?;
                writeln!(
                    replay,
                    "{name:?} => fixture_lifted::{identifier}::check(&lifted(name)?, folder, &run()?),"
                )
                .map_err(std::io::Error::other)?;
                writeln!(bindings, "pub(super) mod {identifier} {{ #[allow(unused_imports)] use crate::{identifier}::*; pub(crate) fn check(module: &umpire_lift::Module, folder: &std::path::Path, native: &umpire_testing::Files) -> umpire_testing::RunResult {{ umpire_testing::backend::check_machine(&{}, module, folder, native) }} }}", module.metadata.world_path).map_err(std::io::Error::other)?;
            }
            Err(diagnostic) => {
                println!("cargo:warning={diagnostic}");
                writeln!(source, "{name:?} => Err({:?}.into()),", diagnostic.to_string())
                    .map_err(std::io::Error::other)?;
            }
        }
    }
    source.push_str("_ => Err(format!(\"unregistered source fixture: {name}\")), } }\n");
    // An unselected fixture reports its lift refusal first.
    replay.push_str("_ => { lifted(name)?; Err(format!(\"unselected backend coverage: {name}\").into()) } } }\n");
    source.push_str(&replay);
    bindings.push_str("}\n");
    source.push_str(&bindings);
    writeln!(source, "fn admitted() -> &'static [&'static str] {{ &{admitted:?} }}")
        .map_err(std::io::Error::other)?;
    std::fs::write(output.join("lifted.rs"), source)
}

fn strings(values: &[String]) -> String {
    format!(
        "vec![{}]",
        values.iter().map(|value| format!("{value:?}.into()")).collect::<Vec<_>>().join(",")
    )
}

fn kind(value: &crate::Type) -> String {
    use crate::Type;
    match value {
        Type::Bool => "umpire_lift::Type::Bool".into(),
        Type::Int { min, max } => format!("umpire_lift::Type::Int {{ min: {min}, max: {max} }}"),
        Type::Text => "umpire_lift::Type::Text".into(),
        Type::Unit => "umpire_lift::Type::Unit".into(),
        Type::Enum { name, variants } => format!(
            "umpire_lift::Type::Enum {{ name: {name:?}.into(), variants: {} }}",
            strings(variants)
        ),
        Type::Tuple(types) => format!(
            "umpire_lift::Type::Tuple(vec![{}])",
            types.iter().map(kind).collect::<Vec<_>>().join(",")
        ),
        Type::Seq(inner) => format!("umpire_lift::Type::Seq(Box::new({}))", kind(inner)),
    }
}

fn properties(values: &[crate::Property]) -> String {
    format!("vec![{}]", values.iter().map(|value| format!("umpire_lift::Property {{ name: {:?}.into(), operator: {:?}.into(), cfg: {:?}.into() }}", value.name, value.operator, value.cfg)).collect::<Vec<_>>().join(","))
}

fn fields_code(fields: &[crate::Field]) -> String {
    fields
        .iter()
        .map(|field| {
            format!(
                "umpire_lift::Field {{ name: {:?}.into(), ty: {} }}",
                field.name,
                kind(&field.ty)
            )
        })
        .collect::<Vec<_>>()
        .join(",")
}

fn literal_code(value: &crate::Literal) -> String {
    match value {
        crate::Literal::Bool(value) => format!("umpire_lift::Literal::Bool({value})"),
        crate::Literal::Int(value) => format!("umpire_lift::Literal::Int({value})"),
        crate::Literal::Enum(value) => format!("umpire_lift::Literal::Enum({value:?}.into())"),
        crate::Literal::Sequence(values) => format!(
            "umpire_lift::Literal::Sequence(vec![{}])",
            values.iter().map(literal_code).collect::<Vec<_>>().join(",")
        ),
    }
}

fn metadata(value: &crate::Metadata) -> String {
    let fields = fields_code(&value.fields);
    let actions = value
        .actions
        .iter()
        .map(|action| {
            format!(
                "umpire_lift::Action {{ name: {:?}.into(), operator: {:?}.into(), internal: {}, fields: vec![{}] }}",
                action.name,
                action.operator,
                action.internal,
                fields_code(&action.fields)
            )
        })
        .collect::<Vec<_>>()
        .join(",");
    let progress = value.progress.iter().map(|claim| format!("umpire_lift::Progress {{ name: {:?}.into(), operator: {:?}.into(), cfg: {:?}.into(), allowed: {}, fair: {}, allowed_inputs: vec!{:?}, fair_inputs: vec!{:?}, bound: {:?} }}", claim.name, claim.operator, claim.cfg, strings(&claim.allowed), strings(&claim.fair), claim.allowed_inputs, claim.fair_inputs, claim.bound)).collect::<Vec<_>>().join(",");
    let catalog = value
        .catalog
        .iter()
        .map(|entry| {
            format!(
                "umpire_lift::Concrete {{ action: {:?}.into(), fields: vec![{}] }}",
                entry.action,
                entry
                    .fields
                    .iter()
                    .map(|(name, value)| format!("({name:?}.into(), {})", literal_code(value)))
                    .collect::<Vec<_>>()
                    .join(",")
            )
        })
        .collect::<Vec<_>>()
        .join(",");
    let observers = value
        .observer_types
        .iter()
        .map(|(name, ty)| format!("({name:?}.into(), {})", kind(ty)))
        .collect::<Vec<_>>()
        .join(",");
    format!(
        "umpire_lift::Metadata {{ state_name: {:?}.into(), world_path: {:?}.into(), fields: vec![{fields}], actions: vec![{actions}], catalog: vec![{catalog}], invariants: {}, postconditions: {}, progress: vec![{progress}], negative: {}, observer_fields: {}, observer_types: std::collections::BTreeMap::from([{observers}]), outcome_type: {} }}",
        value.state_name,
        value.world_path,
        properties(&value.invariants),
        properties(&value.postconditions),
        properties(&value.negative),
        strings(&value.observer_fields),
        kind(&value.outcome_type)
    )
}

#[cfg(test)]
mod tests {
    #[test]
    fn declared_world_binds_at_test_time_instead_of_eager_artifacts() -> std::io::Result<()> {
        let root = std::env::temp_dir().join(format!("umpire-bound-build-{}", std::process::id()));
        let fixture = root.join("counter");
        let output = root.join("out");
        std::fs::create_dir_all(&fixture)?;
        std::fs::create_dir_all(&output)?;
        std::fs::write(
            fixture.join("model.rs"),
            r"
use umpire_dsl::machine;
#[machine]
pub mod counter {
    use umpire_dsl::{Action, State, World};
    #[derive(World)]
    pub struct World { initial: u8, limit: u8 }
    #[derive(State)]
    pub struct Model { value: u8 }
    #[derive(Action)]
    pub enum User { Advance }
    impl Model {
        #[starts]
        fn initial(w: &World) -> impl IntoIterator<Item = Self> {
            [Self { value: w.initial }]
        }
        #[on(User)]
        fn user(&mut self, action: User, w: &World) {
            match action {
                User::Advance if self.value < w.limit => { self.value += 1; }
                _ => {}
            }
        }
    }
}
",
        )?;
        super::generate(&[("counter".into(), "counter".into(), fixture)], &output)?;
        let generated = std::fs::read_to_string(output.join("lifted.rs"))?;
        assert!(generated.contains("\"counter\" => counter_tests::bound().lift("), "{generated}");
        assert!(
            generated.contains("\"counter\" => counter_tests::bound().check(folder),"),
            "{generated}"
        );
        assert!(!output.join("counter.tla").exists());
        std::fs::remove_dir_all(root)
    }

    #[test]
    fn refused_sources_report_their_diagnostic_through_the_backend() -> std::io::Result<()> {
        let root =
            std::env::temp_dir().join(format!("umpire-refused-build-{}", std::process::id()));
        let fixture = root.join("refused");
        let output = root.join("out");
        std::fs::create_dir_all(&fixture)?;
        std::fs::create_dir_all(&output)?;
        std::fs::write(fixture.join("model.rs"), "pub mod refused {}\n")?;
        super::generate(&[("refused".into(), "refused".into(), fixture)], &output)?;
        let generated = std::fs::read_to_string(output.join("lifted.rs"))?;
        assert!(generated.contains("_ => { lifted(name)?;"), "{generated}");
        assert!(generated.contains("\"refused\" => Err("), "{generated}");
        std::fs::remove_dir_all(root)
    }
}
