//! Procedural macro support for the Umpire DSL.

mod action;
mod machine;
mod path;
mod state;
mod world;

/// Parses the typed path language behind `path!`.
#[doc(hidden)]
#[manyhow::manyhow(proc_macro)]
pub fn __path_parse(input: proc_macro2::TokenStream) -> syn::Result<proc_macro2::TokenStream> {
    path::expand(input)
}

/// Derive state value traits, Debug/Display formatting and ordered initial states.
#[manyhow::manyhow(proc_macro_derive(State, attributes(starts, __umpire_computed_starts)))]
pub fn state(input: proc_macro2::TokenStream) -> manyhow::Result<proc_macro2::TokenStream> {
    state::expand(&syn::parse2(input)?)
}

/// Derive fixed-world value traits, Debug/Display formatting and a field-order constructor.
#[manyhow::manyhow(proc_macro_derive(World))]
pub fn world(input: proc_macro2::TokenStream) -> manyhow::Result<proc_macro2::TokenStream> {
    world::expand(&syn::parse2(input)?)
}

/// Derive action value traits, Debug/Display formatting and a complete ordered catalog.
#[manyhow::manyhow(proc_macro_derive(Action, attributes(internal, fault, inputs)))]
pub fn action(input: proc_macro2::TokenStream) -> manyhow::Result<proc_macro2::TokenStream> {
    action::expand(&syn::parse2(input)?)
}

/// Register an inline module as a world with ordered actions and grouped claims.
#[manyhow::manyhow(proc_macro_attribute)]
pub fn machine(
    args: proc_macro2::TokenStream,
    input: proc_macro2::TokenStream,
) -> manyhow::Result<proc_macro2::TokenStream> {
    machine::expand(&args, &syn::parse2(input)?)
}
