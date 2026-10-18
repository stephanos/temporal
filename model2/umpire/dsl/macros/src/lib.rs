//! Procedural macro support for the Umpire DSL.

mod action;
mod state;

/// Derive state value traits, canonical spelling and ordered initial states.
#[manyhow::manyhow(proc_macro_derive(State, attributes(starts)))]
pub fn state(input: proc_macro2::TokenStream) -> manyhow::Result<proc_macro2::TokenStream> {
    state::expand(&syn::parse2(input)?)
}

/// Derive unit-action value traits, canonical spelling and an ordered catalog.
#[manyhow::manyhow(proc_macro_derive(Action, attributes(internal)))]
pub fn action(input: proc_macro2::TokenStream) -> manyhow::Result<proc_macro2::TokenStream> {
    action::expand(&syn::parse2(input)?)
}
