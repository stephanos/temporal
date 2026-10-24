//! The admitted Model, printed as data the compiled Model embeds for the lifted back ends, and
//! the typed accessors the interpreted checker reads and writes the state through.
use proc_macro2::{Span, TokenStream};
use quote::quote;
use umpire_front::{Embedded, Lifted, Type};
use umpire_ir::print::Print;

/// `impl Lifts for World` in the expanded module, off the wasm guest, and for an admitted
/// module also `impl Interprets for World`.
pub(crate) fn embed(
    declared: &umpire_front::declare::Machine,
    input: &syn::ItemMod,
) -> Vec<syn::Item> {
    let span = input.ident.span();
    // rustc's path for the file; the runner resolves it, since the macro reads no files.
    let file = span.local_file().map_or_else(|| span.file(), |file| file.display().to_string());
    let admission = crate::roots::admission(input);
    let admitted = umpire_front::admit_checks(declared, &admission);
    let interprets =
        admitted.as_ref().ok().map(|(model, checks)| interprets(declared, model, checks));
    let embedded = Embedded {
        file,
        model: admitted.map(|(model, _)| model),
        source: umpire_front::source(input),
    }
    .print();
    let mut items = vec![syn::parse_quote! {
        #[cfg(not(target_arch = "wasm32"))]
        #[doc(hidden)]
        impl ::umpire_dsl::__lifted::Lifts for World {
            fn __umpire_lifted() -> ::umpire_dsl::__lifted::Embedded { #embedded }
        }
    }];
    items.extend(interprets);
    items
}

fn interprets(
    declared: &umpire_front::declare::Machine,
    model: &Lifted,
    checks: &[TokenStream],
) -> syn::Item {
    let state = &declared.state;
    let names: Vec<_> = model.fields.iter().map(|field| ident(&field.name)).collect();
    let slots: Vec<_> = (0..model.fields.len()).collect();
    let gets = model
        .fields
        .iter()
        .zip(&names)
        .map(|(field, name)| encode(&field.ty, &quote!(state.#name)));
    let sets = model.fields.iter().map(|field| decode(&field.ty, &quote!(value)));
    let values = model
        .fields
        .iter()
        .enumerate()
        .map(|(slot, field)| decode(&field.ty, &quote!(&values[#slot])));
    let invariants = 0..model.invariants.len();
    let postconditions = 0..model.postconditions.len();
    let progress = 0..model.progress.len();
    let outcome = decode(&model.outcome_type, &quote!(value));
    let outcome_value = encode(&model.outcome_type, &quote!((*outcome)));
    let fact = (model.fact_type != Type::Unit).then(|| {
        let value = decode(&model.fact_type, &quote!(value));
        quote! {
            fn __umpire_fact(
                value: &::umpire_dsl::__lifted::value::Value,
            ) -> <Self as ::umpire_dsl::Machine>::Fact {
                use ::umpire_dsl::__lifted::value::Value as __UmpireValue;
                #value
            }
        }
    });
    syn::parse_quote! {
        #[cfg(not(target_arch = "wasm32"))]
        #[doc(hidden)]
        #[allow(unreachable_patterns, clippy::all, clippy::pedantic)]
        impl ::umpire_dsl::__lifted::Interprets for World {
            fn __umpire_get(state: &#state, slot: usize) -> ::umpire_dsl::__lifted::value::Value {
                use ::umpire_dsl::__lifted::value::Value as __UmpireValue;
                match slot {
                    #(#slots => #gets,)*
                    _ => ::core::panic!("the state has no field at slot {slot}"),
                }
            }
            fn __umpire_set(state: &mut #state, slot: usize, value: &::umpire_dsl::__lifted::value::Value) {
                use ::umpire_dsl::__lifted::value::Value as __UmpireValue;
                match slot {
                    #(#slots => state.#names = #sets,)*
                    _ => ::core::panic!("the state has no field at slot {slot}"),
                }
            }
            fn __umpire_state(values: &[::umpire_dsl::__lifted::value::Value]) -> #state {
                use ::umpire_dsl::__lifted::value::Value as __UmpireValue;
                #state { #(#names: #values),* }
            }
            fn __umpire_outcome(
                value: &::umpire_dsl::__lifted::value::Value,
            ) -> <Self as ::umpire_dsl::Machine>::Outcome {
                use ::umpire_dsl::__lifted::value::Value as __UmpireValue;
                #outcome
            }
            #fact
            fn __umpire_outcome_value(
                outcome: &<Self as ::umpire_dsl::Machine>::Outcome,
            ) -> ::umpire_dsl::__lifted::value::Value {
                use ::umpire_dsl::__lifted::value::Value as __UmpireValue;
                #outcome_value
            }
            fn __umpire_callbacks<W: ::umpire_dsl::__lifted::Evaluates<Self>>(
            ) -> ::umpire_dsl::__lifted::Callbacks<W, Self> {
                ::umpire_dsl::__lifted::Callbacks {
                    invariants: ::std::vec![#((
                        |world: &W, state: &#state| {
                            ::umpire_dsl::__lifted::Evaluates::<Self>::__umpire_invariant(world, #invariants, state)
                        },
                        |world: &W, state: &#state| {
                            ::umpire_dsl::claim::evaluate(|| {
                                ::umpire_dsl::__lifted::Evaluates::<Self>::__umpire_invariant(world, #invariants, state)
                            })
                            .verdict
                        },
                    )),*],
                    postconditions: ::std::vec![#(
                        |world: &W, before: &#state, after: &#state, _: &<Self as ::umpire_dsl::Machine>::Action, outcome: &<Self as ::umpire_dsl::Machine>::Outcome, _: &[<Self as ::umpire_dsl::Machine>::Fact]| {
                            ::umpire_dsl::__lifted::Evaluates::<Self>::__umpire_postcondition(world, #postconditions, before, after, outcome)
                        }
                    ),*],
                    progress: ::std::vec![#(
                        |world: &W, state: &#state| {
                            ::umpire_dsl::__lifted::Evaluates::<Self>::__umpire_progress(world, #progress, state)
                        }
                    ),*],
                }
            }
            fn __umpire_checks() -> &'static [&'static str] {
                // `check!`'s own capture and spelling of its argument.
                macro_rules! __umpire_text {
                    ($expression:expr $(,)?) => { ::core::stringify!($expression) };
                }
                &[#(__umpire_text!(#checks)),*]
            }
        }
    }
}

fn ident(name: &str) -> syn::Ident {
    name.strip_prefix("r#").map_or_else(
        || syn::Ident::new(name, Span::call_site()),
        |raw| syn::Ident::new_raw(raw, Span::call_site()),
    )
}

/// The Rust type of an admitted integer range, so decoding never relies on inference.
fn integer(min: i128, max: i128) -> Option<TokenStream> {
    Some(match (min, max) {
        (-128, 127) => quote!(i8),
        (-32_768, 32_767) => quote!(i16),
        (-2_147_483_648, 2_147_483_647) => quote!(i32),
        (0, 255) => quote!(u8),
        (0, 65_535) => quote!(u16),
        _ => return None,
    })
}

/// The pattern or path of one qualified enum variant; the stock outcome and rejection by
/// absolute path, since the module may not import them.
fn variant(ty: &Type, variant: &str) -> TokenStream {
    let rejection = |name: &str| {
        let name = syn::Ident::new(name, Span::call_site());
        quote!(::umpire_dsl::Rejection::#name)
    };
    if *ty == Type::outcome() {
        let rejected =
            variant.strip_prefix("Outcome::Rejected(").and_then(|rest| rest.strip_suffix(')'));
        if let Some(name) = rejected {
            let rejection = rejection(name);
            return quote!(::umpire_dsl::Outcome::Rejected(#rejection));
        }
        return quote!(::umpire_dsl::Outcome::Accepted);
    }
    if *ty == Type::rejection() {
        return rejection(variant.trim_start_matches("Rejection::"));
    }
    syn::parse_str::<syn::Path>(variant)
        .map_or_else(|error| error.to_compile_error(), |path| quote!(#path))
}

/// `place`, a value of type `ty`, as an interpreter value.
fn encode(ty: &Type, place: &TokenStream) -> TokenStream {
    match ty {
        Type::Bool => quote!(__UmpireValue::Bool(#place)),
        Type::Int { .. } => quote!(__UmpireValue::Int(::core::primitive::i128::from(#place))),
        Type::Bounded(bounded) => {
            let member = bounded.field.as_ref().map_or_else(
                || quote!(0),
                |name| {
                    let name = ident(name);
                    quote!(#name)
                },
            );
            quote!(__UmpireValue::Int(::core::primitive::i128::from((#place).#member)))
        }
        Type::Optional(bounded) => {
            let inner = encode(&Type::Bounded(bounded.clone()), &quote!((*__umpire_payload)));
            quote!(__UmpireValue::Int(match &(#place) {
                ::core::option::Option::None => 0,
                ::core::option::Option::Some(__umpire_payload) => match #inner {
                    __UmpireValue::Int(raw) => raw + 1,
                    _ => ::core::unreachable!(),
                },
            }))
        }
        Type::Enum { variants, .. } => {
            let arms = variants.iter().enumerate().map(|(index, name)| {
                let pattern = variant(ty, name);
                quote!(#pattern => #index,)
            });
            quote!(__UmpireValue::Enum(match &#place {
                #(#arms)*
                _ => ::core::unreachable!("a variant outside the admitted Model"),
            }))
        }
        Type::Text => quote!(__UmpireValue::Text(::std::string::ToString::to_string(&#place))),
        Type::Unit => quote!({
            let _ = &#place;
            __UmpireValue::Unit
        }),
        Type::Tuple(types) => {
            let elements = types.iter().enumerate().map(|(index, ty)| {
                let index = syn::Index::from(index);
                encode(ty, &quote!(#place.#index))
            });
            quote!(__UmpireValue::Tuple(::std::vec![#(#elements),*]))
        }
        Type::Seq(inner) => {
            let element = encode(inner, &quote!((*__umpire_item)));
            quote!(__UmpireValue::Seq(#place.iter().map(|__umpire_item| #element).collect()))
        }
    }
}

/// The value of type `ty` that `value`, a `&Value`, holds.
fn decode(ty: &Type, value: &TokenStream) -> TokenStream {
    let mismatch = quote!(::core::panic!("an interpreted value does not match its admitted type"));
    match ty {
        Type::Bool => quote!(match #value { __UmpireValue::Bool(value) => *value, _ => #mismatch }),
        Type::Int { min, max } => {
            let target = integer(*min, *max).map_or_else(
                || quote!(::core::convert::TryFrom::try_from),
                |ty| quote!(<#ty as ::core::convert::TryFrom<i128>>::try_from),
            );
            quote!(match #value {
                __UmpireValue::Int(value) => #target(*value).unwrap_or_else(|_| #mismatch),
                _ => #mismatch,
            })
        }
        Type::Bounded(bounded) => {
            let name = ident(&bounded.name);
            let backing = ident(&bounded.backing);
            let raw = quote!(<#backing as ::core::convert::TryFrom<i128>>::try_from(*raw).unwrap_or_else(|_| #mismatch));
            let construct = bounded.field.as_ref().map_or_else(
                || quote!(#name(#raw)),
                |field| {
                    let field = ident(field);
                    quote!(#name { #field: #raw })
                },
            );
            quote!(match #value { __UmpireValue::Int(raw) => #construct, _ => #mismatch })
        }
        Type::Optional(bounded) => {
            let inner =
                decode(&Type::Bounded(bounded.clone()), &quote!(&__UmpireValue::Int(*code - 1)));
            quote!(match #value {
                __UmpireValue::Int(0) => ::core::option::Option::None,
                __UmpireValue::Int(code) => ::core::option::Option::Some(#inner),
                _ => #mismatch,
            })
        }
        Type::Enum { variants, .. } => {
            let arms = variants.iter().enumerate().map(|(index, name)| {
                let path = variant(ty, name);
                quote!(#index => #path,)
            });
            quote!(match #value {
                __UmpireValue::Enum(index) => match *index { #(#arms)* _ => #mismatch },
                _ => #mismatch,
            })
        }
        Type::Text => {
            quote!(match #value { __UmpireValue::Text(value) => ::core::clone::Clone::clone(value), _ => #mismatch })
        }
        Type::Unit => quote!(match #value { __UmpireValue::Unit => (), _ => #mismatch }),
        Type::Tuple(types) => {
            let elements = types
                .iter()
                .enumerate()
                .map(|(index, ty)| decode(ty, &quote!(&__umpire_elements[#index])));
            quote!(match #value {
                __UmpireValue::Tuple(__umpire_elements) => (#(#elements,)*),
                _ => #mismatch,
            })
        }
        Type::Seq(inner) => {
            let element = decode(inner, &quote!(__umpire_item));
            quote!(match #value {
                __UmpireValue::Seq(__umpire_items) => __umpire_items.iter().map(|__umpire_item| #element).collect(),
                _ => #mismatch,
            })
        }
    }
}
