//! Procedural attributes for descriptor-admitted Temporal realization handlers.

use std::path::Path;

use quote::quote;
use umpire_temporal_annotate_syntax::{realization::Schema, realize::expand_in_source};

#[cfg(test)]
use umpire_temporal_annotate_syntax::realize::expand;

fn schema() -> Result<(Schema, Vec<std::path::PathBuf>), String> {
    Schema::load(
        &Path::new(env!("CARGO_MANIFEST_DIR")).join("../.."),
        Path::new(concat!(env!("OUT_DIR"), "/realization-api.binpb")),
    )
}

/// Generate `Realize` dispatch from typed observation and callback handlers.
#[proc_macro_attribute]
pub fn realization(
    args: proc_macro::TokenStream,
    input: proc_macro::TokenStream,
) -> proc_macro::TokenStream {
    let model = syn::parse_macro_input!(args as syn::Path);
    let item = syn::parse_macro_input!(input as syn::ItemImpl);
    let result = schema().map_err(|error| syn::Error::new_spanned(&item, error)).and_then(
        |(schema, mut inputs)| {
            let span = item.impl_token.span;
            let source = span.local_file().ok_or_else(|| {
                syn::Error::new(span, "realization requires its Rust source scope")
            })?;
            let source = std::fs::canonicalize(source).map_err(|error| {
                syn::Error::new(span, format!("cannot locate realization source scope: {error}"))
            })?;
            let text = std::fs::read_to_string(&source).map_err(|error| {
                syn::Error::new(span, format!("cannot read realization source scope: {error}"))
            })?;
            let expanded = expand_in_source(&model, item, &schema, &text, span.start())?;
            inputs.push(source);
            let tracked =
                inputs.iter().map(|input| input.to_string_lossy().into_owned()).map(|input| {
                    quote!(
                        const _: &[u8] = include_bytes!(#input);
                    )
                });
            Ok(quote!(#(#tracked)* #expanded))
        },
    );
    result.unwrap_or_else(syn::Error::into_compile_error).into()
}

fn misplaced(name: &str) -> proc_macro::TokenStream {
    syn::Error::new(
        proc_macro::Span::call_site().into(),
        format!("{name} belongs inside #[realization]"),
    )
    .into_compile_error()
    .into()
}

/// Declare one observed RPC handler inside `#[realization]`.
#[proc_macro_attribute]
pub fn observe(_: proc_macro::TokenStream, _: proc_macro::TokenStream) -> proc_macro::TokenStream {
    misplaced("observe")
}

/// Declare one incoming callback handler inside `#[realization]`.
#[proc_macro_attribute]
pub fn answer(_: proc_macro::TokenStream, _: proc_macro::TokenStream) -> proc_macro::TokenStream {
    misplaced("answer")
}

/// Delegate deterministic proposal selection inside `#[realization]`.
#[proc_macro_attribute]
pub fn propose(_: proc_macro::TokenStream, _: proc_macro::TokenStream) -> proc_macro::TokenStream {
    misplaced("propose")
}

#[cfg(test)]
mod tests {
    #[test]
    fn answer_contexts_follow_rpc_descriptor_edges() -> Result<(), Box<dyn std::error::Error>> {
        let (schema, _) = super::schema()?;
        for (task, produced, expected) in [
            (
                "PollActivityTaskQueueRequest",
                "RespondActivityTaskCompletedRequest",
                "callback task must be an RPC response",
            ),
            (
                "PollActivityTaskQueueResponse",
                "PollActivityTaskQueueResponse",
                "produced request must be an RPC input",
            ),
            ("PollActivityTaskQueueResponse", "Failure", "produced request must be an RPC input"),
        ] {
            let item = syn::parse_str(&format!(
                "impl Binding {{ #[answer({task}, produces({produced}, fields()))] fn respond(task: {task}) -> Result<Reply<Worker>, Refusal> {{ todo!() }} }}"
            ))?;
            let error = super::expand(&syn::parse_str("activity")?, item, &schema)
                .err()
                .ok_or("expected RPC edge admission refusal")?;
            assert!(error.to_string().contains(expected), "{error}");
        }
        Ok(())
    }

    #[test]
    fn field_declarations_use_the_admitted_catalog_graph() -> Result<(), Box<dyn std::error::Error>>
    {
        let (schema, _) = super::schema()?;
        for (declarations, expected) in [
            ("reads(info), ignores(info.status = \"overlap\")", "overlapping field paths"),
            ("reads(info.priority.name)", "opaque field stops nested path"),
            ("ignores(info.status = \" \" )", "ignore reason must not be empty"),
        ] {
            let item = syn::parse_str(&format!(
                "impl Binding {{ #[observe(WorkflowService::DescribeActivityExecution, {declarations})] fn describe(response: DescribeActivityExecutionResponse) -> Result<Projection<Action>, Refusal> {{ todo!() }} }}"
            ))?;
            let error = super::expand(&syn::parse_str("activity")?, item, &schema)
                .err()
                .ok_or("expected field admission refusal")?;
            assert!(error.to_string().contains(expected), "{error}");
        }
        Ok(())
    }

    #[test]
    fn activity_expansion() -> Result<(), Box<dyn std::error::Error>> {
        let source = include_str!("../../../feature/activity/temporal.rs");
        let file = syn::parse_file(source)?;
        let mut item = file
            .items
            .into_iter()
            .find_map(|item| match item {
                syn::Item::Impl(item)
                    if item
                        .attrs
                        .iter()
                        .any(|attribute| attribute.path().is_ident("realization")) =>
                {
                    Some(item)
                }
                _ => None,
            })
            .ok_or("activity realization impl is missing")?;
        item.attrs.retain(|attribute| !attribute.path().is_ident("realization"));
        let (schema, _) = super::schema()?;
        let location = item.impl_token.span.start();
        let expanded = umpire_temporal_annotate_syntax::realize::expand_in_source(
            &syn::parse_str("activity")?,
            item,
            &schema,
            source,
            location,
        )?;
        let file: syn::File = syn::parse2(expanded)?;
        let [
            syn::Item::Use(import),
            syn::Item::Impl(authored),
            syn::Item::Impl(metadata),
            syn::Item::Impl(dispatch),
        ] = file.items.as_slice()
        else {
            return Err("unexpected realization expansion items".into());
        };
        let import = quote::ToTokens::to_token_stream(import).to_string();
        if import
            != quote::quote!(
                use ::umpire_runtime::AnyMessage as _;
            )
            .to_string()
        {
            return Err(format!("unexpected realization import: {import}").into());
        }
        let [syn::ImplItem::Const(contexts)] = metadata.items.as_slice() else {
            return Err("unexpected realization metadata items".into());
        };
        if !file.attrs.is_empty()
            || file.shebang.is_some()
            || authored.trait_.is_some()
            || metadata.trait_.is_some()
            || dispatch.trait_.is_none()
            || contexts.ident != "REALIZATION"
        {
            return Err("unexpected realization expansion structure".into());
        }
        let render = |item: &syn::ItemImpl| {
            let mut section = file.clone();
            section.items = vec![syn::Item::Impl(item.clone())];
            prettyplease::unparse(&section)
        };
        insta::assert_snapshot!("activity-realization-authored", render(authored));
        insta::assert_snapshot!("activity-realization-metadata", render(metadata));
        insta::assert_snapshot!("activity-realization-dispatch", render(dispatch));
        Ok(())
    }
}
