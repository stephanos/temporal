//! Procedural macro support for the Umpire DSL.

mod action;
mod machine;
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

/// Register an inherent impl as a world with ordered actions and named claims.
#[manyhow::manyhow(proc_macro_attribute)]
pub fn machine(
    args: proc_macro2::TokenStream,
    input: proc_macro2::TokenStream,
) -> manyhow::Result<proc_macro2::TokenStream> {
    machine::expand(args, &syn::parse2(input)?)
}

/// Mark an action handler without changing the Rust method.
#[proc_macro_attribute]
pub fn on(_: proc_macro::TokenStream, input: proc_macro::TokenStream) -> proc_macro::TokenStream {
    input
}

/// Mark a state predicate without changing the Rust method.
#[proc_macro_attribute]
pub fn invariant(
    _: proc_macro::TokenStream,
    input: proc_macro::TokenStream,
) -> proc_macro::TokenStream {
    input
}

/// Mark an edge predicate without changing the Rust method.
#[proc_macro_attribute]
pub fn postcondition(
    _: proc_macro::TokenStream,
    input: proc_macro::TokenStream,
) -> proc_macro::TokenStream {
    input
}

/// Mark a named scope builder without changing the Rust method.
#[proc_macro_attribute]
pub fn scope(
    _: proc_macro::TokenStream,
    input: proc_macro::TokenStream,
) -> proc_macro::TokenStream {
    input
}

/// Mark a progress builder without changing the Rust method.
#[proc_macro_attribute]
pub fn progress(
    _: proc_macro::TokenStream,
    input: proc_macro::TokenStream,
) -> proc_macro::TokenStream {
    input
}
