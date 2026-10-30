use manyhow::ToTokensError;
use proc_macro2::TokenStream;
use quote::ToTokens;
use syn::{Item, Meta, spanned::Spanned};

mod cases;

#[derive(Clone, Copy)]
enum Entry {
    Action,
    Finite,
    Machine,
    State,
}

#[derive(Debug, PartialEq, Eq)]
struct Diagnostic {
    message: &'static str,
    line: usize,
    column: usize,
}

struct Case {
    name: &'static str,
    source: &'static str,
    entry: Entry,
    expected: &'static [Diagnostic],
}

#[derive(Debug, PartialEq, Eq)]
struct Observed {
    message: String,
    line: usize,
    column: usize,
}

fn diagnostics(error: manyhow::Error) -> Result<Vec<Observed>, String> {
    let emitted =
        syn::parse2::<syn::File>(error.into_token_stream()).map_err(|error| error.to_string())?;
    emitted
        .items
        .into_iter()
        .map(|item| {
            let Item::Macro(item) = item else {
                return Err("diagnostic emitted a non-macro item".into());
            };
            if item.mac.path.segments.last().is_none_or(|part| part.ident != "compile_error") {
                return Err("diagnostic emitted a macro other than compile_error".into());
            }
            let start = item.mac.path.span().start();
            let message = syn::parse2::<syn::LitStr>(item.mac.tokens)
                .map_err(|error| error.to_string())?
                .value();
            Ok(Observed { message, line: start.line, column: start.column + 1 })
        })
        .collect()
}

fn expand(case: &Case) -> Result<Vec<Observed>, String> {
    let file = syn::parse_file(case.source).map_err(|error| error.to_string())?;
    let mut observed = Vec::new();
    let mut selected = 0;
    for item in file.items {
        let expansion = match (case.entry, item) {
            (Entry::Machine, Item::Mod(mut module)) => {
                let Some(index) = module.attrs.iter().position(|attr| {
                    attr.path().segments.last().is_some_and(|part| part.ident == "machine")
                }) else {
                    continue;
                };
                let attribute = module.attrs.remove(index);
                let args = match attribute.meta {
                    Meta::Path(_) => TokenStream::new(),
                    Meta::List(list) => list.tokens,
                    Meta::NameValue(_) => return Err("machine attribute has a name/value".into()),
                };
                super::machine::expand(&args, &module)
            }
            (Entry::Action | Entry::Finite | Entry::State, item) => {
                let derive = match case.entry {
                    Entry::Action => "Action",
                    Entry::Finite => "Finite",
                    Entry::State => "State",
                    Entry::Machine => unreachable!("module expansion handled above"),
                };
                let attrs = match &item {
                    Item::Enum(item) => &item.attrs,
                    Item::Struct(item) => &item.attrs,
                    Item::Union(item) => &item.attrs,
                    _ => continue,
                };
                if !umpire_front::declare::derives(attrs, derive)
                    .map_err(|error| error.to_string())?
                {
                    continue;
                }
                let input = syn::parse2::<syn::DeriveInput>(item.into_token_stream())
                    .map_err(|error| error.to_string())?;
                match case.entry {
                    Entry::Action => super::action::expand(&input),
                    Entry::Finite => super::finite::expand(&input),
                    Entry::State => super::state::expand(&input),
                    Entry::Machine => unreachable!("module expansion handled above"),
                }
            }
            _ => continue,
        };
        selected += 1;
        let error = expansion.err().ok_or_else(|| {
            format!("{}: annotated item {selected} unexpectedly admitted", case.name)
        })?;
        observed.extend(diagnostics(error)?);
    }
    if selected == 0 {
        return Err(format!("{}: no annotated input selected", case.name));
    }
    Ok(observed)
}

#[test]
fn parser_refusals_preserve_messages_and_authored_locations() -> Result<(), String> {
    for case in cases::CASES {
        let observed = expand(case).map_err(|error| format!("{}: {error}", case.name))?;
        let expected: Vec<_> = case
            .expected
            .iter()
            .map(|error| Observed {
                message: error.message.to_owned(),
                line: error.line,
                column: error.column,
            })
            .collect();
        assert!(!expected.is_empty(), "{} has no expected diagnostics", case.name);
        assert_eq!(observed, expected, "{}", case.name);
    }
    Ok(())
}

#[test]
fn diagnostic_capture_keeps_duplicate_messages_at_distinct_locations() -> Result<(), String> {
    let source =
        syn::parse_file("struct First;\nstruct Second;").map_err(|error| error.to_string())?;
    let mut error = syn::Error::new_spanned(&source.items[0], "same refusal");
    error.combine(syn::Error::new_spanned(&source.items[1], "same refusal"));
    assert_eq!(
        diagnostics(error.into())?,
        [
            Observed { message: "same refusal".into(), line: 1, column: 1 },
            Observed { message: "same refusal".into(), line: 2, column: 1 },
        ]
    );
    Ok(())
}
