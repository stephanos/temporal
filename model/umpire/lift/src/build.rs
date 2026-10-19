//! Build-time source admission and generated backend fixture bindings.

/// Generate emitted artifacts and typed fixture bindings from independently read source roots.
#[allow(clippy::print_stdout)]
pub fn generate(
    fixtures: &[(String, std::path::PathBuf)],
    output: &std::path::Path,
) -> std::io::Result<()> {
    use std::fmt::Write;
    let mut source = String::from(
        "#[allow(clippy::unreadable_literal, reason = \"Generated exact integer bounds\")]\nfn lifted(name: &str) -> Result<umpire_lift::Module, String> { match name {\n",
    );
    let mut replay = String::from(
        "#[allow(unused_imports, unused_qualifications)]\nfn check_backend(name: &str, module: &umpire_lift::Module, folder: &std::path::Path, native: &umpire_testing::Files) -> umpire_testing::RunResult { match name {\n",
    );
    let mut admitted = Vec::new();
    for (name, folder) in fixtures {
        let input = folder.join("model.rs");
        println!("cargo:rerun-if-changed={}", input.display());
        match crate::Lift::source(&input).and_then(|lifted| crate::Tla::emit(&lifted)) {
            Ok(module) => {
                admitted.push(name.clone());
                for input in &module.metadata.inputs {
                    println!("cargo:rerun-if-changed={input}");
                }
                let tla = output.join(format!("{name}.tla"));
                let cfg = output.join(format!("{name}.cfg"));
                std::fs::write(&tla, &module.tla)?;
                std::fs::write(&cfg, &module.cfg)?;
                writeln!(source, "{name:?} => Ok(umpire_lift::Module {{ name: {:?}.into(), tla: include_str!({:?}).into(), cfg: include_str!({:?}).into(), metadata: {} }}),", module.name, tla.to_string_lossy(), cfg.to_string_lossy(), metadata(&module.metadata)).map_err(std::io::Error::other)?;
                let mut identifier = String::from("fixture_");
                for byte in name.bytes() {
                    if byte.is_ascii_lowercase() || byte.is_ascii_digit() {
                        identifier.push(char::from(byte));
                    } else {
                        write!(identifier, "_{byte:02x}").map_err(std::io::Error::other)?;
                    }
                }
                writeln!(replay, "{name:?} => umpire_testing::backend::check_machine(&{{ use {identifier}::*; {} }}, module, folder, native),", module.metadata.world_path).map_err(std::io::Error::other)?;
            }
            Err(diagnostic) => {
                println!("cargo:warning={diagnostic}");
                writeln!(source, "{name:?} => Err({:?}.into()),", diagnostic.to_string())
                    .map_err(std::io::Error::other)?;
            }
        }
    }
    source.push_str("_ => Err(format!(\"unregistered source fixture: {name}\")), } }\n");
    replay.push_str("_ => Err(format!(\"unselected backend coverage: {name}\").into()), } }\n");
    source.push_str(&replay);
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

fn metadata(value: &crate::Metadata) -> String {
    let fields = value
        .fields
        .iter()
        .map(|field| {
            format!(
                "umpire_lift::Field {{ name: {:?}.into(), ty: {} }}",
                field.name,
                kind(&field.ty)
            )
        })
        .collect::<Vec<_>>()
        .join(",");
    let actions = value
        .actions
        .iter()
        .map(|action| {
            format!(
                "umpire_lift::Action {{ name: {:?}.into(), operator: {:?}.into(), internal: {} }}",
                action.name, action.operator, action.internal
            )
        })
        .collect::<Vec<_>>()
        .join(",");
    let progress = value.progress.iter().map(|claim| format!("umpire_lift::Progress {{ name: {:?}.into(), operator: {:?}.into(), cfg: {:?}.into(), allowed: {}, fair: {}, bound: {:?} }}", claim.name, claim.operator, claim.cfg, strings(&claim.allowed), strings(&claim.fair), claim.bound)).collect::<Vec<_>>().join(",");
    let observers = value
        .observer_types
        .iter()
        .map(|(name, ty)| format!("({name:?}.into(), {})", kind(ty)))
        .collect::<Vec<_>>()
        .join(",");
    format!(
        "umpire_lift::Metadata {{ inputs: {}, state_name: {:?}.into(), state_tuple: {}, world_path: {:?}.into(), fields: vec![{fields}], actions: vec![{actions}], invariants: {}, postconditions: {}, progress: vec![{progress}], negative: {}, observer_fields: {}, observer_types: std::collections::BTreeMap::from([{observers}]), outcome_type: {}, fact_type: {} }}",
        strings(&value.inputs),
        value.state_name,
        value.state_tuple,
        value.world_path,
        properties(&value.invariants),
        properties(&value.postconditions),
        properties(&value.negative),
        strings(&value.observer_fields),
        kind(&value.outcome_type),
        kind(&value.fact_type)
    )
}
