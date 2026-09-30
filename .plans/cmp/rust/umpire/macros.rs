//! # umpire-macros: the DSL entry points
//!
//! Crate `umpire-macros` (`proc-macro = true`), re-exported by `umpire`. One `#[proc_macro]` per
//! block of the Lean grammar, parsed with `syn` and emitted with `quote`.
//!
//! Two kinds of error, both pinned to the author's token:
//!
//! 1. What the block can check about itself (an `unobservable:` that is not a timer, a timer with
//!    no `steps:` line, a duplicate key, a `set!` whose purpose and fields disagree, a `holds:`
//!    closure with the wrong arity for its claim): `syn::Error::new_spanned(token, msg)`, turned
//!    into `compile_error!` at that span. The message is ours.
//! 2. What lives in another item (an undeclared action on a `steps:` line, a step function whose
//!    signature disagrees with the action's inputs, a misspelled phase or fact): the macro emits
//!    the reference with `quote_spanned!(ident.span() => ..)` and lets rustc resolve it. The
//!    message is rustc's (E0433, E0599, E0593), but the caret is under the same token.
//!
//! A proc-macro sees one invocation and nothing else, so (2) is not laziness: `machine!` cannot
//! know which `action!` blocks exist. Making every cross-item reference an ordinary Rust path is
//! how the compiler checks it for us.
//!
//! Bodies are sketched; the parse structs and `check` functions are the part that matters.

use proc_macro::TokenStream;
use proc_macro2::Span;
use quote::{format_ident, quote, quote_spanned};
use syn::parse::{Parse, ParseStream};
use syn::punctuated::Punctuated;
use syn::spanned::Spanned;
use syn::{braced, bracketed, parse_macro_input, Expr, ExprClosure, Ident, LitStr, Path, Token, Type};

/// The block keywords. `syn::custom_keyword!` gives each a parseable type with a span, so an
/// error on a misplaced `map:` line can point at the `map` token itself.
mod kw {
    syn::custom_keyword!(party);
    syn::custom_keyword!(creates);
    syn::custom_keyword!(on);
    syn::custom_keyword!(schema);
    syn::custom_keyword!(input);
    syn::custom_keyword!(results);
    syn::custom_keyword!(examples);
    syn::custom_keyword!(refer);
    syn::custom_keyword!(key);
    syn::custom_keyword!(read);
    syn::custom_keyword!(state);
    syn::custom_keyword!(outcome);
    syn::custom_keyword!(facts);
    syn::custom_keyword!(refines);
    syn::custom_keyword!(map);
    syn::custom_keyword!(starts);
    syn::custom_keyword!(ends);
    syn::custom_keyword!(timers);
    syn::custom_keyword!(unobservable);
    syn::custom_keyword!(evidence);
    syn::custom_keyword!(steps);
    syn::custom_keyword!(from);
    syn::custom_keyword!(restrict);
    syn::custom_keyword!(machine);
    syn::custom_keyword!(when);
    syn::custom_keyword!(holds);
    syn::custom_keyword!(model);
    syn::custom_keyword!(actions);
    syn::custom_keyword!(search);
    syn::custom_keyword!(find);
    syn::custom_keyword!(verify);
    syn::custom_keyword!(limits);
    syn::custom_keyword!(purpose);
    syn::custom_keyword!(bind);
    syn::custom_keyword!(repeat);
    syn::custom_keyword!(queries);
    syn::custom_keyword!(cover);
    syn::custom_keyword!(budget);
    syn::custom_keyword!(members);
    syn::custom_keyword!(sync);
}

// ---------------------------------------------------------------------------------------------
// #[derive(Finite)]
// ---------------------------------------------------------------------------------------------

/// `Finite` for an enum (payload variants expand over their fields' `all()`, in field order) or a
/// struct (the product of its fields, first field slowest). A struct with a field named `phase`
/// also gets `Phased`. An enum with no variants gets `CARDINALITY = 0`.
#[proc_macro_derive(Finite)]
pub fn derive_finite(input: TokenStream) -> TokenStream {
    let item = parse_macro_input!(input as syn::DeriveInput);
    let name = &item.ident;
    let expanded = match &item.data {
        syn::Data::Enum(e) => {
            let cardinality = e.variants.iter().map(|v| {
                let tys = v.fields.iter().map(|f| &f.ty);
                quote! { (1 #(* <#tys as ::umpire::Finite>::CARDINALITY)*) }
            });
            let all = e.variants.iter().map(|_v| quote! { /* nested all() over payload fields */ });
            let key = e.variants.iter().map(|v| {
                let ident = &v.ident;
                let lower = lower_camel(&ident.to_string());
                quote! { Self::#ident { .. } => ::std::string::String::from(#lower) /* + payload keys */ }
            });
            quote! {
                impl ::umpire::Finite for #name {
                    const CARDINALITY: usize = 0 #(+ #cardinality)*;
                    fn all() -> ::std::vec::Vec<Self> { let mut out = ::std::vec::Vec::new(); #(#all)* out }
                    fn key(&self) -> ::std::string::String { match *self { #(#key,)* } }
                }
            }
        }
        syn::Data::Struct(s) => {
            let fields: Vec<_> = s.fields.iter().map(|f| f.ident.clone().unwrap()).collect();
            let tys: Vec<_> = s.fields.iter().map(|f| &f.ty).collect();
            let phased = fields.iter().position(|f| f == "phase").map(|i| {
                let phase_ty = tys[i];
                let others = fields.iter().filter(|f| *f != "phase");
                quote! {
                    impl ::umpire::Phased for #name {
                        type Phase = #phase_ty;
                        fn phase(&self) -> Self::Phase { self.phase.clone() }
                        fn at(phase: Self::Phase) -> Self {
                            Self { phase, #(#others: ::umpire::Finite::first(),)* }
                        }
                    }
                }
            });
            quote! {
                impl ::umpire::Finite for #name {
                    const CARDINALITY: usize = 1 #(* <#tys as ::umpire::Finite>::CARDINALITY)*;
                    fn all() -> ::std::vec::Vec<Self> { todo!("cartesian product over fields") }
                    fn key(&self) -> ::std::string::String {
                        [#(::umpire::Finite::key(&self.#fields)),*].join("-")
                    }
                }
                #phased
            }
        }
        syn::Data::Union(u) => {
            return syn::Error::new(u.union_token.span, "Finite cannot be derived for a union")
                .to_compile_error()
                .into()
        }
    };
    expanded.into()
}

// ---------------------------------------------------------------------------------------------
// entity! / observation! / limits!
// ---------------------------------------------------------------------------------------------

/// `entity! { operation refer: { caller: workflow } key: scheduledEvent }` ->
/// `pub static operation: Entity = Entity { .. }`.
#[proc_macro]
pub fn entity(input: TokenStream) -> TokenStream {
    struct EntityInput { name: Ident, refer: Vec<(Ident, Ident)>, key: Option<Ident> }
    impl Parse for EntityInput {
        fn parse(input: ParseStream) -> syn::Result<Self> {
            let name = input.parse()?;
            let mut refer = Vec::new();
            let mut key = None;
            while !input.is_empty() {
                if input.peek(kw::refer) {
                    input.parse::<kw::refer>()?; input.parse::<Token![:]>()?;
                    let content; braced!(content in input);
                    for pair in Punctuated::<KeyIdent, Token![,]>::parse_terminated(&content)? {
                        refer.push((pair.key, pair.value));
                    }
                } else if input.peek(kw::key) {
                    input.parse::<kw::key>()?; input.parse::<Token![:]>()?;
                    key = Some(input.parse()?);
                } else {
                    return Err(input.error("expected `refer:` or `key:`"));
                }
            }
            Ok(Self { name, refer, key })
        }
    }
    let EntityInput { name, refer, key } = parse_macro_input!(input as EntityInput);
    let name_s = name.to_string();
    let key = match key { Some(k) => { let k = k.to_string(); quote!(Some(#k)) } None => quote!(None) };
    let refer = refer.iter().map(|(role, target)| {
        let role = role.to_string();
        // The target entity is an ordinary static: an undeclared one is E0425 on this token.
        let target = quote_spanned!(target.span()=> &#target);
        quote!((#role, #target))
    });
    quote! {
        #[allow(non_upper_case_globals)]
        pub static #name: ::umpire::Entity = ::umpire::Entity {
            name: #name_s, key: #key, refer: &[#(#refer),*],
        };
    }
    .into()
}

#[proc_macro]
pub fn observation(input: TokenStream) -> TokenStream {
    let _ = input;
    todo!("`observation! { pendingAttempts on: operation read: attempts }` -> `pub static pendingAttempts: Observation`")
}

/// `limits! { two steps: 2 actions: 2 search: 512 }` -> a `const`-constructed `static`.
#[proc_macro]
pub fn limits(input: TokenStream) -> TokenStream {
    let _ = input;
    todo!()
}

// ---------------------------------------------------------------------------------------------
// action!
// ---------------------------------------------------------------------------------------------

struct ActionInput {
    name: Ident,
    party: Ident,
    binding: Option<(bool, Ident)>, // (creates?, entity)
    schema: Option<LitStr>,
    inputs: Vec<(Ident, Type)>,
    results: Option<Type>,
    examples: Vec<(Expr, LitStr)>,
}

impl Parse for ActionInput {
    fn parse(input: ParseStream) -> syn::Result<Self> {
        let _ = input;
        todo!("keyword loop like `entity!`; `input: { a: T, b: U }` and `examples: { expr => \"..\" }`")
    }
}

/// `action! { handlerReply party: handler on: operation input: { reply: Reply } .. }` expands to
///
/// ```ignore
/// #[allow(non_camel_case_types)] pub struct handlerReply;
/// impl Action for handlerReply { const NAME = "handlerReply"; type Input = (Reply,); .. }
/// impl handlerReply {
///     pub fn class(reply: Reply) -> ActionClass { .. }
///     pub fn any() -> ActionPattern { ActionPattern::Any("handlerReply") }
///     pub fn bind<S, O, F>(f: impl Fn(&S, Reply) -> Vec<Step<S, O, F>> + ..) -> Vec<BoundStep<S, O, F>> {
///         <Reply as Finite>::all().into_iter().map(|reply| BoundStep {
///             class: Self::class(reply.clone()),
///             run: Box::new(move |s| f(s, reply.clone())),
///         }).collect()
///     }
/// }
/// ```
///
/// `bind`'s parameter list is the declared inputs, so a step function whose signature disagrees
/// is E0593 (or E0631 for a wrong argument type) where `machine!` names it.
#[proc_macro]
pub fn action(input: TokenStream) -> TokenStream {
    let a = parse_macro_input!(input as ActionInput);
    let name = &a.name;
    let name_s = name.to_string();
    let party = format_ident!("{}", upper_camel(&a.party.to_string()));
    let (arg_names, arg_tys): (Vec<_>, Vec<_>) = a.inputs.iter().cloned().unzip();
    let binding = match &a.binding {
        Some((true, e)) => quote_spanned!(e.span()=> Some(::umpire::Binding::Creates(&#e))),
        Some((false, e)) => quote_spanned!(e.span()=> Some(::umpire::Binding::On(&#e))),
        None => quote!(None),
    };
    let schema = match &a.schema { Some(s) => quote!(Some(#s)), None => quote!(None) };
    let examples = a.examples.iter().map(|(class, value)| quote!(((#class,), #value)));
    quote! {
        #[allow(non_camel_case_types)]
        #[derive(Clone, Copy, Debug)]
        pub struct #name;

        impl ::umpire::Action for #name {
            const NAME: &'static str = #name_s;
            const PARTY: ::umpire::Party = ::umpire::Party::#party;
            type Input = (#(#arg_tys,)*);
            fn binding() -> Option<::umpire::Binding> { #binding }
            const SCHEMA: Option<&'static str> = #schema;
            fn examples() -> Vec<(Self::Input, &'static str)> { vec![#(#examples),*] }
        }

        impl #name {
            pub fn class(#(#arg_names: #arg_tys),*) -> ::umpire::ActionClass {
                ::umpire::ActionClass {
                    action: #name_s,
                    party: ::umpire::Party::#party,
                    inputs: vec![#(::umpire::Finite::key(&#arg_names)),*],
                }
            }
            pub fn any() -> ::umpire::ActionPattern { ::umpire::ActionPattern::Any(#name_s) }
            pub fn bind<S, O, F>(
                f: impl Fn(&S, #(#arg_tys),*) -> Vec<::umpire::Step<S, O, F>> + Clone + Send + Sync + 'static,
            ) -> Vec<::umpire::BoundStep<S, O, F>>
            where S: ::umpire::Finite, O: ::umpire::Finite, F: ::umpire::Finite,
            {
                <(#(#arg_tys,)*) as ::umpire::Finite>::all().into_iter().map(|(#(#arg_names,)*)| {
                    let f = f.clone();
                    ::umpire::BoundStep {
                        class: Self::class(#(#arg_names.clone()),*),
                        run: Box::new(move |s| f(s, #(#arg_names.clone()),*)),
                    }
                }).collect()
            }
        }
    }
    .into()
}

// ---------------------------------------------------------------------------------------------
// machine!
// ---------------------------------------------------------------------------------------------

/// One `key: value` pair.
struct KeyIdent { key: Ident, value: Ident }
impl Parse for KeyIdent {
    fn parse(input: ParseStream) -> syn::Result<Self> {
        let key = input.parse()?;
        input.parse::<Token![:]>()?;
        Ok(Self { key, value: input.parse()? })
    }
}

struct KeyPath { key: Ident, value: Path }
impl Parse for KeyPath {
    fn parse(input: ParseStream) -> syn::Result<Self> {
        let key = input.parse()?;
        input.parse::<Token![:]>()?;
        Ok(Self { key, value: input.parse()? })
    }
}

enum MachineInput {
    /// `machine! { handlerWorker from: polling restrict: [workerStop, serve] }`
    Restricted { name: Ident, from: Path, restrict: Vec<Path> },
    Full(FullMachine),
}

struct FullMachine {
    name: Ident,
    entity: Ident,
    state: Type,
    outcome: Type,
    facts: Type,
    refines: Option<Ident>,
    map: Option<Path>,
    starts: Vec<Ident>,
    ends: Vec<Ident>,
    timers: Vec<Ident>,
    unobservable: Vec<Ident>,
    evidence: Vec<KeyIdent>,
    steps: Vec<KeyPath>,
    /// Kept for the error on a `refines:` without `map:` (and vice versa).
    refines_kw: Option<kw::refines>,
    map_kw: Option<kw::map>,
}

impl Parse for MachineInput {
    fn parse(input: ParseStream) -> syn::Result<Self> {
        let name: Ident = input.parse()?;
        if input.peek(kw::from) {
            input.parse::<kw::from>()?; input.parse::<Token![:]>()?;
            let from = input.parse()?;
            input.parse::<kw::restrict>()?; input.parse::<Token![:]>()?;
            let content; bracketed!(content in input);
            let restrict = Punctuated::<Path, Token![,]>::parse_terminated(&content)?.into_iter().collect();
            return Ok(Self::Restricted { name, from, restrict });
        }
        let _ = name;
        todo!("keyword loop: `for:`, `state:`, `outcome:`, `facts:`, `refines:`, `map:`, `starts:`, `ends:`, `timers:`, `unobservable:`, `evidence: { .. }`, `steps: { .. }`")
    }
}

/// What the block can say about itself. Every error is `new_spanned` on the offending token.
fn check_machine(m: &FullMachine) -> syn::Result<()> {
    let timers: Vec<String> = m.timers.iter().map(Ident::to_string).collect();

    for u in &m.unobservable {
        if !timers.contains(&u.to_string()) {
            return Err(syn::Error::new_spanned(
                u,
                format!("`{u}` is listed under `unobservable:` but is not one of this machine's timers ({})", timers.join(", ")),
            ));
        }
    }

    for t in &m.timers {
        if !m.steps.iter().any(|s| s.key == *t) {
            return Err(syn::Error::new_spanned(
                t,
                format!("timer `{t}` has no `steps:` line; a timer is a system action the machine owns, so it needs a step function"),
            ));
        }
    }

    let mut seen = std::collections::HashSet::new();
    for s in &m.steps {
        if !seen.insert(s.key.to_string()) {
            return Err(syn::Error::new_spanned(&s.key, format!("`{}` is bound twice under `steps:`", s.key)));
        }
    }

    match (&m.refines_kw, &m.map_kw) {
        (Some(r), None) => return Err(syn::Error::new_spanned(r, "`refines:` needs a `map:` line naming the abstraction function")),
        (None, Some(k)) => return Err(syn::Error::new_spanned(k, "`map:` without `refines:`: which machine does this map to?")),
        _ => {}
    }
    Ok(())
}

/// See the crate docs for the two error tiers. The expansion:
///
/// ```ignore
/// #[allow(non_camel_case_types)] pub struct backoff;            // one per `timers:` entry
/// impl Action for backoff { PARTY = System, Input = (), .. }     // and its class()/any()/bind()
///
/// #[allow(non_camel_case_types)] pub enum nexusProtocol {}       // the type-namespace twin
/// impl Typed for nexusProtocol { type S = ProtocolState; type O = ..; type F = ..; }
///
/// #[allow(non_upper_case_globals)]
/// pub static nexusProtocol: LazyLock<Machine<ProtocolState, ProtocolOutcome, ProtocolFact>> =
///     LazyLock::new(|| {
///         type S = ProtocolState;
///         MachineBuilder::new("nexusProtocol", &operation)
///             .starts(vec![S::at(<<S as Phased>::Phase>::Unscheduled)])
///             .ends(|s| matches!(s.phase(), <<S as Phased>::Phase>::Succeeded | ..))
///             .timers(vec!["backoff", ..])
///             .unobservable(vec!["backoff"])
///             .evidence(|f| matches!(f, ProtocolFact::NexusOperationScheduled { .. }), "nexusOperationScheduled")
///             .steps(schedule::bind(schedule_step))                // span of `schedule` and of `schedule_step`
///             .steps(handlerReply::bind(protocol_handler_reply_step))
///             ..
///             .refines(&nexusProduct, product_of)
///             .build()
///     });
/// ```
#[proc_macro]
pub fn machine(input: TokenStream) -> TokenStream {
    let input = parse_macro_input!(input as MachineInput);
    let m = match input {
        MachineInput::Restricted { name, from, restrict } => {
            // `<workerStop as Action>::NAME` rather than a string: an undeclared action is E0433 here.
            let names = restrict.iter().map(|a| quote_spanned!(a.span()=> <#a as ::umpire::Action>::NAME));
            return quote! {
                #[allow(non_camel_case_types)] pub enum #name {}
                impl ::umpire::Typed for #name {
                    type S = <#from as ::umpire::Typed>::S;
                    type O = <#from as ::umpire::Typed>::O;
                    type F = <#from as ::umpire::Typed>::F;
                }
                #[allow(non_upper_case_globals)]
                pub static #name: ::std::sync::LazyLock<::umpire::Machine<
                    <#from as ::umpire::Typed>::S, <#from as ::umpire::Typed>::O, <#from as ::umpire::Typed>::F>> =
                    ::std::sync::LazyLock::new(|| #from.restrict(&[#(#names),*]));
            }
            .into();
        }
        MachineInput::Full(m) => m,
    };
    if let Err(e) = check_machine(&m) {
        return e.to_compile_error().into();
    }

    let FullMachine { name, entity, state, outcome, facts, refines, map, starts, ends, timers, unobservable, evidence, steps, .. } = m;
    let name_s = name.to_string();

    let timer_items = timers.iter().map(|t| {
        let t_s = t.to_string();
        quote! {
            #[allow(non_camel_case_types)] #[derive(Clone, Copy, Debug)] pub struct #t;
            impl ::umpire::Action for #t {
                const NAME: &'static str = #t_s;
                const PARTY: ::umpire::Party = ::umpire::Party::System;
                type Input = ();
                fn binding() -> Option<::umpire::Binding> { None }
                const SCHEMA: Option<&'static str> = None;
            }
            impl #t {
                pub fn class() -> ::umpire::ActionClass {
                    ::umpire::ActionClass { action: #t_s, party: ::umpire::Party::System, inputs: vec![] }
                }
                pub fn any() -> ::umpire::ActionPattern { ::umpire::ActionPattern::Any(#t_s) }
                pub fn bind<S, O, F>(f: impl Fn(&S) -> Vec<::umpire::Step<S, O, F>> + Send + Sync + 'static)
                    -> Vec<::umpire::BoundStep<S, O, F>>
                where S: ::umpire::Finite, O: ::umpire::Finite, F: ::umpire::Finite,
                {
                    vec![::umpire::BoundStep { class: Self::class(), run: Box::new(f) }]
                }
            }
        }
    });

    let starts = starts.iter().map(|p| quote_spanned!(p.span()=> S::at(<<S as ::umpire::Phased>::Phase>::#p)));
    let ends = ends.iter().map(|p| quote_spanned!(p.span()=> <<S as ::umpire::Phased>::Phase>::#p));
    let timer_names = timers.iter().map(Ident::to_string);
    let unobservable = unobservable.iter().map(Ident::to_string);
    let evidence = evidence.iter().map(|KeyIdent { key, value }| {
        let recorded = value.to_string();
        // The fact variant at its own span: a misspelled fact is E0599 on this token.
        let pat = quote_spanned!(key.span()=> #facts::#key { .. });
        quote! { .evidence(|f| matches!(f, #pat), #recorded) }
    });
    let steps = steps.iter().map(|KeyPath { key, value }| {
        // The action ident at its own span: an undeclared action is E0433 on this token, and a
        // step function whose signature disagrees with the action's inputs is rustc's closure
        // error on `value` (E0593 for the wrong arity, E0631 for a wrong argument type).
        let action = quote_spanned!(key.span()=> #key);
        quote! { .steps(#action::bind(#value)) }
    });
    let refines = match (refines, map) {
        (Some(target), Some(map)) => quote_spanned!(target.span()=> .refines(&#target, #map)),
        _ => quote!(),
    };

    quote! {
        #(#timer_items)*

        #[allow(non_camel_case_types)] pub enum #name {}
        impl ::umpire::Typed for #name { type S = #state; type O = #outcome; type F = #facts; }

        #[allow(non_upper_case_globals)]
        pub static #name: ::std::sync::LazyLock<::umpire::Machine<#state, #outcome, #facts>> =
            ::std::sync::LazyLock::new(|| {
                type S = #state;
                ::umpire::MachineBuilder::<#state, #outcome, #facts>::new(#name_s, &#entity)
                    .starts(vec![#(#starts),*])
                    .ends(|s| matches!(::umpire::Phased::phase(s), #(#ends)|*))
                    .timers(vec![#(#timer_names),*])
                    .unobservable(vec![#(#unobservable),*])
                    #(#evidence)*
                    #(#steps)*
                    #refines
                    .build()
            });
    }
    .into()
}

// ---------------------------------------------------------------------------------------------
// property! / scenario! / query!
// ---------------------------------------------------------------------------------------------

struct PropertyInput { name: Ident, machine: Ident, when: Option<Expr>, holds: ExprClosure }

impl Parse for PropertyInput {
    fn parse(input: ParseStream) -> syn::Result<Self> {
        let _ = input;
        todo!("`machine:` ident, optional `when:` expr (`handlerReply(SyncSuccess)` or bare `handlerReply`), `holds:` closure")
    }
}

/// The arity of `holds:` follows the claim: `when:` present means a same-step claim `|step|`,
/// absent means a transition claim `|before, after|`. Checked here, on the closure's tokens.
fn check_property(p: &PropertyInput) -> syn::Result<()> {
    let arity = p.holds.inputs.len();
    match (&p.when, arity) {
        (Some(_), 1) | (None, 2) => Ok(()),
        (Some(_), n) => Err(syn::Error::new_spanned(
            &p.holds,
            format!("a same-step claim (`when:` present) takes one argument, the step; this closure takes {n}"),
        )),
        (None, n) => Err(syn::Error::new_spanned(
            &p.holds,
            format!("a transition claim (no `when:`) takes two arguments, the step before and the step after; this closure takes {n}. Add `when:` to make it a same-step claim."),
        )),
    }
}

/// `when: handlerReply(SyncSuccess)` -> `handlerReply::class(SyncSuccess).into()`;
/// `when: scheduleToStart` -> `scheduleToStart::any()`. Both are ordinary paths.
fn when_pattern(when: &Expr) -> proc_macro2::TokenStream {
    match when {
        Expr::Call(call) => {
            let f = &call.func;
            let args = &call.args;
            quote_spanned!(f.span()=> ::umpire::ActionPattern::from(#f::class(#args)))
        }
        Expr::Path(p) => quote_spanned!(p.span()=> #p::any()),
        other => syn::Error::new_spanned(other, "expected an action, with or without inputs").to_compile_error(),
    }
}

#[proc_macro]
pub fn property(input: TokenStream) -> TokenStream {
    let p = parse_macro_input!(input as PropertyInput);
    if let Err(e) = check_property(&p) {
        return e.to_compile_error().into();
    }
    let PropertyInput { name, machine, when, holds } = p;
    let name_s = name.to_string();
    let (s, o, f) = typed_of(&machine);
    let claim = match when {
        Some(when) => {
            let when = when_pattern(&when);
            quote! { ::umpire::Claim::SameStep { when: #when, holds: #holds } }
        }
        None => quote! { ::umpire::Claim::Transition { holds: #holds } },
    };
    quote! {
        #[allow(non_upper_case_globals)]
        pub static #name: ::std::sync::LazyLock<::umpire::Property<#s, #o, #f>> =
            ::std::sync::LazyLock::new(|| ::umpire::Property { name: #name_s, machine: &#machine, claim: #claim });
    }
    .into()
}

/// `<nexusProtocol as Typed>::S` and friends: the static's type without knowing the state type.
fn typed_of(machine: &Ident) -> (proc_macro2::TokenStream, proc_macro2::TokenStream, proc_macro2::TokenStream) {
    (
        quote_spanned!(machine.span()=> <#machine as ::umpire::Typed>::S),
        quote_spanned!(machine.span()=> <#machine as ::umpire::Typed>::O),
        quote_spanned!(machine.span()=> <#machine as ::umpire::Typed>::F),
    )
}

/// `scenario! { syncReplied model: nexusProtocol starts: Unscheduled actions: [schedule(Unset, Unset, Unset), handlerReply(SyncSuccess)] }`
///
/// An action line is rewritten `name(args)` -> `name::class(args)`, bare `name` -> `name::class()`,
/// `member.name(args)` -> `ActionClass::member("member", name::class(args))`. Every one is a Rust
/// path at the author's span. The scenario also gets a `Typed` twin so `query!` can spell its type.
#[proc_macro]
pub fn scenario(input: TokenStream) -> TokenStream {
    let _ = input;
    todo!()
}

struct QueryInput { name: Ident, kind: (Ident, bool), property: Ident, scenario: Ident, limits: Ident }

/// `find:` or `verify:`, exactly one.
impl Parse for QueryInput {
    fn parse(input: ParseStream) -> syn::Result<Self> {
        let name = input.parse()?;
        let kind: Ident = input.parse()?;
        let is_find = match kind.to_string().as_str() {
            "find" => true,
            "verify" => false,
            other => return Err(syn::Error::new_spanned(&kind, format!("expected `find:` or `verify:`, found `{other}:`"))),
        };
        input.parse::<Token![:]>()?;
        let property = input.parse()?;
        input.parse::<Token![in]>()?; input.parse::<Token![:]>()?;
        let scenario = input.parse()?;
        input.parse::<kw::limits>()?; input.parse::<Token![:]>()?;
        let limits = input.parse()?;
        Ok(Self { name, kind: (kind, is_find), property, scenario, limits })
    }
}

#[proc_macro]
pub fn query(input: TokenStream) -> TokenStream {
    let QueryInput { name, kind: (_, is_find), property, scenario, limits } = parse_macro_input!(input as QueryInput);
    let name_s = name.to_string();
    let (s, o, f) = typed_of(&scenario);
    let kind = if is_find { quote!(::umpire::QueryKind::Find) } else { quote!(::umpire::QueryKind::Verify) };
    quote! {
        #[allow(non_upper_case_globals)]
        pub static #name: ::std::sync::LazyLock<::umpire::Query<#s, #o, #f>> =
            ::std::sync::LazyLock::new(|| ::umpire::Query {
                name: #name_s,
                kind: #kind,
                // Coerces through the blanket `PropertyOn` impl; whether the property's machine is
                // the scenario's or one it refines is decided when the query runs.
                property: &*#property,
                scenario: &#scenario,
                limits: &#limits,
            });
    }
    .into()
}

// ---------------------------------------------------------------------------------------------
// set!
// ---------------------------------------------------------------------------------------------

struct SetInput {
    name: Ident,
    purpose: Ident,
    bind: Vec<KeyIdent>,
    repeat: Option<Ident>,
    queries: Option<(kw::queries, Vec<Ident>)>,
    machine: Option<(kw::machine, Ident)>,
    cover: Option<(kw::cover, Vec<Ident>)>,
    budget: Option<(kw::budget, Ident)>,
}

impl Parse for SetInput {
    fn parse(input: ParseStream) -> syn::Result<Self> {
        let _ = input;
        todo!()
    }
}

/// A set's shape follows its purpose: `queries:` for functional and canary, `machine:`/`cover:`/
/// `budget:` for exploratory, never both. Rejected on the token that does not belong.
fn check_set(s: &SetInput) -> syn::Result<()> {
    match s.purpose.to_string().as_str() {
        "functional" | "canary" => {
            if let Some((kw, _)) = &s.machine {
                return Err(syn::Error::new_spanned(kw, format!("a {} set lists `queries:`; `machine:` belongs to an exploratory set", s.purpose)));
            }
            if s.queries.as_ref().map_or(true, |(_, q)| q.is_empty()) {
                return Err(syn::Error::new_spanned(&s.purpose, format!("a {} set needs a non-empty `queries:` list", s.purpose)));
            }
            Ok(())
        }
        "exploratory" => {
            if let Some((kw, _)) = &s.queries {
                return Err(syn::Error::new_spanned(kw, "an exploratory set covers a `machine:` rather than listing `queries:`"));
            }
            for (missing, present) in [("machine", s.machine.is_some()), ("cover", s.cover.is_some()), ("budget", s.budget.is_some())] {
                if !present {
                    return Err(syn::Error::new_spanned(&s.purpose, format!("an exploratory set needs a `{missing}:` line")));
                }
            }
            Ok(())
        }
        other => Err(syn::Error::new_spanned(&s.purpose, format!("unknown purpose `{other}`; expected functional, canary or exploratory"))),
    }
}

#[proc_macro]
pub fn set(input: TokenStream) -> TokenStream {
    let s = parse_macro_input!(input as SetInput);
    if let Err(e) = check_set(&s) {
        return e.to_compile_error().into();
    }
    let SetInput { name, purpose, bind, repeat, queries, machine, cover, budget } = s;
    let name_s = name.to_string();
    let purpose = format_ident!("{}", upper_camel(&purpose.to_string()));
    let bind = bind.iter().map(|KeyIdent { key, value }| {
        let party = format_ident!("{}", upper_camel(&key.to_string()));
        let how = format_ident!("{}", upper_camel(&value.to_string()));
        quote!((::umpire::Party::#party, ::umpire::Bind::#how))
    });
    let repeat = match repeat {
        Some(r) => { let r = format_ident!("{}", upper_camel(&r.to_string())); quote!(Some(::umpire::Repeat::#r)) }
        None => quote!(None),
    };
    let queries = queries.map(|(_, q)| q).unwrap_or_default().into_iter()
        .map(|q| quote_spanned!(q.span()=> &*#q as &dyn ::umpire::AnyQuery));
    let machine = match machine {
        Some((_, m)) => quote_spanned!(m.span()=> Some(&*#m as &dyn ::umpire::AnyMachine)),
        None => quote!(None),
    };
    let cover = cover.map(|(_, c)| c).unwrap_or_default().into_iter()
        .map(|c| { let c = format_ident!("{}", upper_camel(&c.to_string())); quote!(::umpire::Cover::#c) });
    let budget = match budget { Some((_, b)) => quote!(Some(&#b)), None => quote!(None) };
    quote! {
        #[allow(non_upper_case_globals)]
        pub static #name: ::std::sync::LazyLock<::umpire::Set> = ::std::sync::LazyLock::new(|| ::umpire::Set {
            name: #name_s,
            purpose: ::umpire::Purpose::#purpose,
            bind: vec![#(#bind),*],
            repeat: #repeat,
            queries: vec![#(#queries),*],
            machine: #machine,
            cover: vec![#(#cover),*],
            budget: #budget,
        });
    }
    .into()
}

// ---------------------------------------------------------------------------------------------
// compose!
// ---------------------------------------------------------------------------------------------

struct ComposeInput {
    name: Ident,
    entities: Vec<Path>,
    state: Type,
    members: Vec<KeyIdent>,
    /// `workerStop: operation.workerStop || worker.workerStop`, parsed as a `syn::Expr::Binary`
    /// with `||` and two field accesses. No custom parser needed.
    sync: Vec<(Ident, Expr)>,
    starts: Vec<KeyIdent>,
    ends: Vec<(Ident, Vec<Ident>)>,
}

impl Parse for ComposeInput {
    fn parse(input: ParseStream) -> syn::Result<Self> {
        let _ = input;
        todo!()
    }
}

/// A `sync:` line may only name the two declared members, and the roles in `starts:`/`ends:`
/// must be members. All within the block, so all `syn::Error`.
fn check_compose(c: &ComposeInput) -> syn::Result<()> {
    let roles: Vec<String> = c.members.iter().map(|m| m.key.to_string()).collect();
    if roles.len() != 2 {
        return Err(syn::Error::new_spanned(&c.name, "compose! takes exactly two members (nest compositions for more)"));
    }
    for (name, expr) in &c.sync {
        let Expr::Binary(syn::ExprBinary { left, op: syn::BinOp::Or(_), right, .. }) = expr else {
            return Err(syn::Error::new_spanned(expr, format!("`{name}:` must be `member.action || member.action`")));
        };
        for side in [left, right] {
            let Expr::Field(syn::ExprField { base, .. }) = &**side else {
                return Err(syn::Error::new_spanned(side, "expected `member.action`"));
            };
            let Expr::Path(p) = &**base else { return Err(syn::Error::new_spanned(base, "expected a member name")) };
            let role = p.path.get_ident().map(Ident::to_string).unwrap_or_default();
            if !roles.contains(&role) {
                return Err(syn::Error::new_spanned(base, format!("`{role}` is not a member of this composition (members: {})", roles.join(", "))));
            }
        }
    }
    Ok(())
}

/// Expands to a `Typed` twin and a `LazyLock<Machine<St, Either<O1, O2>, Either<F1, F2>>>` built
/// from `Compose { left: Member { get: |s| &s.operation, set: |mut s, v| { s.operation = v; s }, .. }, .. }.machine()`.
/// The `get`/`set` closures name the state's fields, so a member whose role is not a field of
/// `state:` is E0609 on the role.
#[proc_macro]
pub fn compose(input: TokenStream) -> TokenStream {
    let c = parse_macro_input!(input as ComposeInput);
    if let Err(e) = check_compose(&c) {
        return e.to_compile_error().into();
    }
    todo!()
}

// ---------------------------------------------------------------------------------------------

fn upper_camel(s: &str) -> String {
    let mut c = s.chars();
    match c.next() { Some(f) => f.to_uppercase().chain(c).collect(), None => String::new() }
}

fn lower_camel(s: &str) -> String {
    let mut c = s.chars();
    match c.next() { Some(f) => f.to_lowercase().chain(c).collect(), None => String::new() }
}

#[allow(dead_code)]
fn call_site() -> Span {
    Span::call_site()
}
