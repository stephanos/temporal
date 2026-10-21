//! Build-time source admission and generated backend fixture bindings.

use quote::{format_ident, quote};
use std::{
    collections::BTreeMap,
    path::{Path, PathBuf},
};

/// Generate emitted artifacts and typed fixture bindings from independently read source roots,
/// given each fixture's name, model module and folder.
pub fn generate(fixtures: &[(String, String, PathBuf)], output: &Path) -> std::io::Result<()> {
    generate_bound(fixtures, output).map(drop)
}

/// Like [`generate`], and also write compiler-checked copies for fixtures whose authored starts
/// require a selected World, returning them as discovery input overrides.
#[allow(clippy::print_stdout)]
pub fn generate_bound(
    fixtures: &[(String, String, PathBuf)],
    output: &Path,
) -> std::io::Result<BTreeMap<(String, String), PathBuf>> {
    use std::fmt::Write;
    let mut overrides = BTreeMap::new();
    let mut source = String::from(
        "#[allow(clippy::unreadable_literal, reason = \"Generated exact integer bounds\")]\nfn lifted(name: &str) -> Result<umpire_lift::Module, String> { match name {\n",
    );
    let mut replay = String::from(
        "#[allow(unused_imports, unused_qualifications)]\nfn check_backend(name: &str, module: &umpire_lift::Module, folder: &std::path::Path, native: &umpire_testing::Files) -> umpire_testing::RunResult { match name {\n",
    );
    // A module evaluates each world path against its fixture's glob alone. Discovery never names
    // a fixture module `fixture_lifted`: `lifted` stays plain and only non-plain names escape.
    let mut bindings = String::from("mod fixture_lifted {\n");
    let mut admitted = Vec::new();
    let mut bound = String::new();
    for (name, identifier, folder) in fixtures {
        let input = folder.join("model.rs");
        println!("cargo:rerun-if-changed={}", input.display());
        println!("cargo:rerun-if-changed={}", folder.join("tests.rs").display());
        let lifted = crate::Lift::source(&input);
        if let Ok(lifted) = &lifted
            && lifted.requires_binding()
        {
            let binding = Bound::read(folder, identifier, lifted)?;
            for (input, file) in [("model.rs", &binding.model), ("tests.rs", &binding.tests)] {
                let path = output.join(format!("{name}-bound-{input}"));
                std::fs::write(&path, prettyplease::unparse(file))?;
                overrides.insert((name.clone(), input.to_owned()), path);
            }
            let module_fn = binding.module_fn;
            let backend_fn = binding.backend_fn;
            admitted.push(name.clone());
            writeln!(source, "{name:?} => {identifier}_tests::{module_fn}(),")
                .map_err(std::io::Error::other)?;
            writeln!(bound, "{name:?} => Some({identifier}_tests::{backend_fn}(folder)),")
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
                    "{name:?} => fixture_lifted::{identifier}::check(module, folder, native),"
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
    if bindings == "mod fixture_lifted {\n" {
        replay = "fn check_backend(name: &str, _module: &umpire_lift::Module, _folder: &std::path::Path, _native: &umpire_testing::Files) -> umpire_testing::RunResult { Err(format!(\"unselected backend coverage: {name}\").into()) }\n".into();
    } else {
        replay.push_str("_ => Err(format!(\"unselected backend coverage: {name}\").into()), } }\n");
    }
    source.push_str(&replay);
    bindings.push_str("}\n");
    source.push_str(&bindings);
    // Always present, so a refused bound fixture reports through lift.txt rather than failing
    // to compile its caller. Fixture crates without a bound route need not call it.
    source.push_str("#[allow(dead_code)]\n");
    if bound.is_empty() {
        source.push_str("fn bound_backend(_name: &str, _folder: &std::path::Path) -> Option<umpire_testing::RunResult> { None }\n");
    } else {
        source.push_str("fn bound_backend(name: &str, folder: &std::path::Path) -> Option<umpire_testing::RunResult> { match name {\n");
        source.push_str(&bound);
        source.push_str("_ => None, } }\n");
    }
    writeln!(source, "fn admitted() -> &'static [&'static str] {{ &{admitted:?} }}")
        .map_err(std::io::Error::other)?;
    std::fs::write(output.join("lifted.rs"), source)?;
    Ok(overrides)
}

struct Bound {
    model: syn::File,
    tests: syn::File,
    module_fn: syn::Ident,
    backend_fn: syn::Ident,
}

impl Bound {
    #[expect(
        clippy::too_many_lines,
        reason = "Generated fixture glue preserves one selected World across consumers"
    )]
    fn read(folder: &Path, identifier: &str, lifted: &crate::Lifted) -> std::io::Result<Self> {
        let mut model = read_rust(&folder.join("model.rs"))?;
        let mut tests = read_rust(&folder.join("tests.rs"))?;
        let original = quote!(#model #tests);
        let prefix = (0..=original.to_string().len())
            .map(|index| format!("__umpire_bound_{index}_"))
            .find(|prefix| !contains_prefix(original.clone(), prefix))
            .ok_or_else(|| std::io::Error::other("generated binding name unavailable"))?;
        let project = format_ident!("{prefix}project");
        let module_fn = format_ident!("{prefix}module");
        let backend_fn = format_ident!("{prefix}backend");
        let select = format_ident!("{prefix}select");
        let bind = format_ident!("{prefix}bind");
        let native = format_ident!("{prefix}native");
        let world = format_ident!("{prefix}world");
        let starts = format_ident!("{prefix}starts");
        let state = format_ident!("{prefix}state");
        let graph = format_ident!("{prefix}graph");
        let files = format_ident!("{prefix}files");
        let space = format_ident!("{prefix}space");
        let module = format_ident!("{prefix}emitted");
        let folder_arg = format_ident!("{prefix}folder");
        let fixture_module = format_ident!("{identifier}");
        let machine: syn::Ident =
            syn::parse_str(lifted.machine_module()).map_err(std::io::Error::other)?;
        let state_type = format_ident!("{}", lifted.state_name);
        let world_type = quote!(crate::#fixture_module::#machine::World);
        let model_type = quote!(<#world_type as umpire_dsl::Machine>::State);
        let declarations = model
            .items
            .iter()
            .find_map(|item| match item {
                syn::Item::Mod(item) if item.ident == machine => {
                    item.content.as_ref().map(|(_, items)| items)
                }
                _ => None,
            })
            .ok_or_else(|| std::io::Error::other("bound machine module missing"))?;
        let world_fields =
            project_fields(&lifted.world_fields(), &quote!(#world), declarations, "World")?;
        let state_fields = project_fields(
            &lifted.state_fields(),
            &quote!(#state),
            declarations,
            &lifted.state_name,
        )?;
        let projection: syn::Item = syn::parse2(quote! {
            #[allow(unused_qualifications, reason = "Generated hygienic paths must not depend on authored imports")]
            pub(crate) fn #project(#world: &World, #starts: &[#state_type]) -> umpire_lift::Binding {
                umpire_lift::Binding {
                    world: vec![#(#world_fields),*],
                    starts: #starts.iter().map(|#state| vec![#(#state_fields),*]).collect(),
                }
            }
        }).map_err(std::io::Error::other)?;
        let machine_items = model
            .items
            .iter_mut()
            .find_map(|item| match item {
                syn::Item::Mod(item) if item.ident == machine => {
                    item.content.as_mut().map(|(_, items)| items)
                }
                _ => None,
            })
            .ok_or_else(|| std::io::Error::other("bound machine module missing"))?;
        machine_items.push(projection);
        let hooks = tests
            .items
            .iter()
            .enumerate()
            .filter_map(|(index, item)| match item {
                syn::Item::Macro(item)
                    if item
                        .mac
                        .path
                        .segments
                        .last()
                        .is_some_and(|part| part.ident == "fixture") =>
                {
                    Some((index, item))
                }
                _ => None,
            })
            .collect::<Vec<_>>();
        let [(hook_index, hook)] = hooks.as_slice() else {
            return Err(std::io::Error::other("bound fixture requires one tests.rs fixture! hook"));
        };
        let arguments = hook
            .mac
            .parse_body_with(
                syn::punctuated::Punctuated::<syn::Expr, syn::Token![,]>::parse_terminated,
            )
            .map_err(std::io::Error::other)?
            .into_iter()
            .collect::<Vec<_>>();
        let [selected, budget, rest @ ..] = arguments.as_slice() else {
            return Err(std::io::Error::other("bound fixture requires a World and state budget"));
        };
        let (negative, extra) = match rest {
            [] => (None, None),
            [negative] => (Some(negative), None),
            [negative, syn::Expr::Assign(extra)] if matches!(extra.left.as_ref(), syn::Expr::Path(path) if path.path.is_ident("extra")) => {
                (Some(negative), Some(extra.right.as_ref()))
            }
            _ => return Err(std::io::Error::other("unsupported bound fixture! arguments")),
        };
        let negative_code = negative.map(|negative| negative_runtime(&prefix, negative));
        let extra_code = extra.map(|extra| quote! {
            for (#module, #folder_arg) in { let #graph: umpire_testing::Files = #extra; #graph } {
                if #files.contains_key(&#module) {
                    return Err(format!("duplicate fixture output: {}", #module).into());
                }
                #files.insert(#module, #folder_arg);
            }
        });
        let input = std::fs::canonicalize(folder.join("model.rs"))?.to_string_lossy().into_owned();
        let mut generated: syn::File = syn::parse2(quote! {
            fn #select() -> umpire_suite::Space<#model_type, std::rc::Rc<#world_type>> {
                let #world = #selected;
                let #starts = umpire_dsl::Machine::starts(&#world);
                umpire_suite::Space::with_world(#world, #starts)
            }
            fn #bind(#space: &umpire_suite::Space<#model_type, std::rc::Rc<#world_type>>) -> Result<umpire_lift::Module, String> {
                let #world = umpire_suite::SpaceBinding::world(#space);
                let #starts = umpire_suite::SpaceBinding::starts(#space);
                let #module = crate::#fixture_module::#machine::#project(#world.as_ref(), #starts);
                umpire_lift::Lift::source(#input)
                    .and_then(|#graph| #graph.bind(&#module))
                    .and_then(|#graph| umpire_lift::Tla::emit(&#graph))
                    .map_err(|#module| #module.to_string())
            }
            pub(crate) fn #module_fn() -> Result<umpire_lift::Module, String> {
                #bind(&#select())
            }
            fn #native(#space: &umpire_suite::Space<#model_type, std::rc::Rc<#world_type>>) -> Result<(umpire_checker::Result<#world_type>, umpire_testing::Files), Box<dyn std::error::Error>> {
                let #world = umpire_suite::SpaceBinding::world(#space);
                let #starts = umpire_suite::SpaceBinding::starts(#space);
                let #graph = umpire_checker::Checker::explore_from(#world.as_ref(), #starts.to_vec(), #budget)?;
                let mut #files = umpire_testing::render(&#graph)?;
                #files.insert("rows.txt".into(), umpire_testing::render_rows(&#graph)?);
                #negative_code
                #extra_code
                Ok((#graph, #files))
            }
            pub(crate) fn run() -> umpire_testing::RunResult {
                #native(&#select()).map(|(_, #files)| #files)
            }
            pub(crate) fn #backend_fn(#folder_arg: &std::path::Path) -> umpire_testing::RunResult {
                let #space = #select();
                let #module = #bind(&#space)?;
                let (#graph, #files) = #native(&#space)?;
                let #world = umpire_suite::SpaceBinding::world(&#space);
                umpire_testing::backend::check_exploration(
                    #world.as_ref(), umpire_suite::SpaceBinding::starts(&#space),
                    &#graph, &#module, #folder_arg, &#files,
                )
            }
        }).map_err(std::io::Error::other)?;
        for item in &mut generated.items {
            let attributes = match item {
                syn::Item::Fn(item) => &mut item.attrs,
                syn::Item::Struct(item) => &mut item.attrs,
                syn::Item::Impl(item) => &mut item.attrs,
                _ => continue,
            };
            attributes.push(syn::parse_quote!(#[allow(unused_qualifications, reason = "Generated hygienic paths must not depend on authored imports")]));
        }
        let index = *hook_index;
        tests.items.splice(index..=index, generated.items);
        Ok(Self { model, tests, module_fn, backend_fn })
    }
}

fn read_rust(path: &Path) -> std::io::Result<syn::File> {
    syn::parse_file(&std::fs::read_to_string(path)?).map_err(std::io::Error::other)
}

fn contains_prefix(tokens: proc_macro2::TokenStream, prefix: &str) -> bool {
    tokens.into_iter().any(|token| match token {
        proc_macro2::TokenTree::Ident(ident) => ident.to_string().starts_with(prefix),
        proc_macro2::TokenTree::Group(group) => contains_prefix(group.stream(), prefix),
        _ => false,
    })
}

fn project_fields(
    fields: &[crate::Field],
    owner: &proc_macro2::TokenStream,
    declarations: &[syn::Item],
    type_name: &str,
) -> std::io::Result<Vec<proc_macro2::TokenStream>> {
    let authored = declarations.iter().find_map(|item| match item {
        syn::Item::Struct(item) if item.ident == type_name => Some(&item.fields),
        _ => None,
    });
    fields
        .iter()
        .map(|field| {
            let name = &field.name;
            let ident: syn::Ident = syn::parse_str(name).map_err(std::io::Error::other)?;
            let rust_type = authored
                .and_then(|fields| fields.iter().find(|field| field.ident.as_ref() == Some(&ident)))
                .map(|field| &field.ty)
                .ok_or_else(|| {
                    std::io::Error::other(
                        "admitted binding field missing from original declaration",
                    )
                })?;
            let value = project_value(&field.ty, &quote!(__umpire_value))?;
            Ok(quote!((#name.into(), {
                let __umpire_value: &#rust_type = &#owner.#ident;
                #value
            })))
        })
        .collect()
}

fn project_value(
    ty: &crate::Type,
    value: &proc_macro2::TokenStream,
) -> std::io::Result<proc_macro2::TokenStream> {
    Ok(match ty {
        crate::Type::Bool => quote!(umpire_lift::Literal::Bool(*#value)),
        crate::Type::Int { .. } => quote!(umpire_lift::Literal::Int(i128::from(*#value))),
        crate::Type::Enum { variants, .. } => {
            let arms = variants
                .iter()
                .map(|variant| {
                    let path: syn::Path = syn::parse_str(variant).map_err(std::io::Error::other)?;
                    Ok(quote!(#path => #variant.into()))
                })
                .collect::<std::io::Result<Vec<_>>>()?;
            quote!(umpire_lift::Literal::Enum(match #value { #(#arms),* }))
        }
        crate::Type::Seq(inner) => {
            let inner = project_value(inner, &quote!(item))?;
            quote!(umpire_lift::Literal::Sequence((#value).iter().map(|item| #inner).collect()))
        }
        _ => return Err(std::io::Error::other("unsupported bound fixture projection type")),
    })
}

fn negative_runtime(prefix: &str, negative: &syn::Expr) -> proc_macro2::TokenStream {
    let world = format_ident!("{prefix}world");
    let starts = format_ident!("{prefix}starts");
    let files = format_ident!("{prefix}files");
    let report = format_ident!("{prefix}report");
    // As `fixture!`: declared claims plus the supplemental one, under the default budget.
    quote! {
        let #report = umpire_checker::Checker::refuted_from(#world.as_ref(), #starts.to_vec(), #negative)?.verify_all()?;
        #files.insert("negative_report.txt".into(), #report.to_string());
        umpire_testing::add_paths(&mut #files, #report);
    }
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
        "umpire_lift::Metadata {{ state_name: {:?}.into(), world_path: {:?}.into(), fields: vec![{fields}], actions: vec![{actions}], invariants: {}, postconditions: {}, progress: vec![{progress}], negative: {}, observer_fields: {}, observer_types: std::collections::BTreeMap::from([{observers}]), outcome_type: {} }}",
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
    fn declared_world_generates_runtime_binding_instead_of_eager_artifacts() -> std::io::Result<()>
    {
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
    #[starts]
    impl Model {
        fn starts(w: &World) -> impl IntoIterator<Item = Self> {
            [Self { value: w.initial }]
        }
    }
    impl Model {
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
        std::fs::write(
            fixture.join("tests.rs"),
            r"
use crate::counter::counter;
fn selected() -> counter::World { counter::World::new(1, 3) }
umpire_testing::fixture!(selected(), 64);
",
        )?;
        super::generate(&[("counter".into(), "counter".into(), fixture)], &output)?;
        let generated = std::fs::read_to_string(output.join("lifted.rs"))?;
        assert!(generated.contains("fn bound_backend("), "{generated}");
        assert!(!output.join("counter.tla").exists());
        std::fs::remove_dir_all(root)
    }

    #[test]
    fn refused_sources_still_generate_the_bound_backend_fallback() -> std::io::Result<()> {
        let root =
            std::env::temp_dir().join(format!("umpire-refused-build-{}", std::process::id()));
        let fixture = root.join("refused");
        let output = root.join("out");
        std::fs::create_dir_all(&fixture)?;
        std::fs::create_dir_all(&output)?;
        std::fs::write(fixture.join("model.rs"), "pub mod refused {}\n")?;
        std::fs::write(fixture.join("tests.rs"), "")?;
        let overrides =
            super::generate_bound(&[("refused".into(), "refused".into(), fixture)], &output)?;
        assert!(overrides.is_empty());
        let generated = std::fs::read_to_string(output.join("lifted.rs"))?;
        assert!(generated.contains("fn bound_backend("), "{generated}");
        assert!(generated.contains("\"refused\" => Err("), "{generated}");
        std::fs::remove_dir_all(root)
    }
}
