//! Procedural macro support for the Umpire DSL.

mod action;
mod finite;
mod lifted;
mod machine;
mod path;
mod roots;
mod state;
mod world;

/// Parses the typed path language behind `path!`.
#[doc(hidden)]
#[manyhow::manyhow(proc_macro)]
pub fn __path_parse(input: proc_macro2::TokenStream) -> syn::Result<proc_macro2::TokenStream> {
    path::expand(input).map(roots::resolve)
}

/// Derive state value traits, Debug/Display formatting and ordered initial states.
#[manyhow::manyhow(proc_macro_derive(
    State,
    attributes(starts, __umpire_computed_starts, __umpire_world, __umpire_values)
))]
pub fn state(input: proc_macro2::TokenStream) -> manyhow::Result<proc_macro2::TokenStream> {
    state::expand(&syn::parse2(input)?).map(roots::resolve)
}

/// Derive fixed-world value traits, Debug/Display formatting and a field-order constructor.
#[manyhow::manyhow(proc_macro_derive(World, attributes(__umpire_world)))]
pub fn world(input: proc_macro2::TokenStream) -> manyhow::Result<proc_macro2::TokenStream> {
    world::expand(&syn::parse2(input)?).map(roots::resolve)
}

/// Derive action value traits, Debug/Display formatting and a complete ordered catalog.
#[manyhow::manyhow(proc_macro_derive(
    Action,
    attributes(internal, fault, inputs, domain, __umpire_world, __umpire_values)
))]
pub fn action(input: proc_macro2::TokenStream) -> manyhow::Result<proc_macro2::TokenStream> {
    action::expand(&syn::parse2(input)?).map(roots::resolve)
}

/// Derive a finite value domain for a fieldless enum or a bounded integer newtype.
#[manyhow::manyhow(proc_macro_derive(Finite, attributes(inputs)))]
pub fn finite(input: proc_macro2::TokenStream) -> manyhow::Result<proc_macro2::TokenStream> {
    finite::expand(&syn::parse2(input)?).map(roots::resolve)
}

/// Register an inline module as a world with ordered actions and grouped claims.
#[manyhow::manyhow(proc_macro_attribute)]
pub fn machine(
    args: proc_macro2::TokenStream,
    input: proc_macro2::TokenStream,
) -> manyhow::Result<proc_macro2::TokenStream> {
    let input = syn::parse2(input)?;
    let declared = umpire_front::declare::machine(&args, &input)?;
    let mut module = machine::generate(&declared, &input)?;
    if let Some((_, items)) = &mut module.content {
        items.extend(lifted::embed(&declared, &input));
    }
    Ok(roots::resolve(quote::quote!(#module)))
}
