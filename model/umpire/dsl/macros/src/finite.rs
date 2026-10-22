use proc_macro2::TokenStream;
use quote::quote;
use umpire_front::declare::{self, Finite};

pub(crate) fn expand(input: &syn::DeriveInput) -> manyhow::Result<TokenStream> {
    let name = &input.ident;
    let (implementation, types, constraints) = input.generics.split_for_impl();
    match declare::finite(input)? {
        Finite::Enum(variants) => {
            let labels = variants.iter().map(|variant| format!("{name}::{variant}"));
            let count = variants.len();
            Ok(quote! {
                #[automatically_derived]
                impl #implementation ::umpire_dsl::Finite for #name #types #constraints {
                    fn domain() -> impl ::core::iter::Iterator<Item = Self> {
                        ::core::iter::IntoIterator::into_iter([#(Self::#variants),*] as [Self; #count])
                    }
                }
                #[doc(hidden)]
                impl #implementation ::umpire_dsl::IntoValue for #name #types #constraints {
                    fn value(&self) -> ::umpire_dsl::Value {
                        match *self { #(Self::#variants => ::umpire_dsl::Value::Enum(#labels)),* }
                    }
                }
            })
        }
        Finite::Newtype { field, ty, low, high } => {
            let construct = field
                .as_ref()
                .map_or_else(|| quote!(Self(raw)), |field| quote!(Self { #field: raw }));
            Ok(quote! {
                #[automatically_derived]
                impl #implementation ::umpire_dsl::Finite for #name #types #constraints {
                    fn domain() -> impl ::core::iter::Iterator<Item = Self> {
                        ::core::ops::RangeInclusive::<#ty>::new(#low, #high).map(|raw| #construct)
                    }
                    fn __umpire_input_domain() -> ::umpire_dsl::InputDomain<Self>
                    where
                        Self: 'static,
                    {
                        ::umpire_dsl::InputDomain::range(::core::ops::RangeInclusive::<#ty>::new(#low, #high)).map(|raw| #construct)
                    }
                }
                impl #implementation #name #types #constraints {
                    /// Builds a value only when `raw` lies in the declared range.
                    pub fn try_new(raw: #ty) -> ::core::option::Option<Self> {
                        Self::contains(raw).then_some(#construct)
                    }
                    /// Whether `raw` lies in the declared range.
                    pub fn contains(raw: #ty) -> bool {
                        ::core::ops::RangeInclusive::<#ty>::new(#low, #high).contains(&raw)
                    }
                }
            })
        }
    }
}
