//! Procedural macro support for the Umpire DSL.

mod action;
mod machine;
mod state;

/// Derive state value traits, Debug/Display formatting and ordered initial states.
#[manyhow::manyhow(proc_macro_derive(State, attributes(starts)))]
pub fn state(input: proc_macro2::TokenStream) -> manyhow::Result<proc_macro2::TokenStream> {
    state::expand(&syn::parse2(input)?)
}

/// Derive unit-action value traits, Debug/Display formatting and an ordered catalog.
#[manyhow::manyhow(proc_macro_derive(Action, attributes(internal)))]
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
