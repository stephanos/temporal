//! Derives constructor tokens for the internal admitted Model tree.
use proc_macro::TokenStream;
use quote::{format_ident, quote};
use syn::{Data, DeriveInput, Fields, Path, parse_macro_input};

/// Print a struct or enum at its explicit `print_path` constructor path.
#[proc_macro_derive(Print, attributes(print_path))]
pub fn derive_print(input: TokenStream) -> TokenStream {
    let input = parse_macro_input!(input as DeriveInput);
    expand(&input).unwrap_or_else(syn::Error::into_compile_error).into()
}

fn expand(input: &DeriveInput) -> syn::Result<proc_macro2::TokenStream> {
    let path: Path = input
        .attrs
        .iter()
        .find(|attr| attr.path().is_ident("print_path"))
        .ok_or_else(|| syn::Error::new_spanned(input, "Print requires a print_path"))?
        .parse_args()?;
    let arm = |fields: &Fields,
               pattern_path: proc_macro2::TokenStream,
               output_path: proc_macro2::TokenStream| {
        let members: Vec<_> = fields.iter().filter_map(|field| field.ident.as_ref()).collect();
        let bindings: Vec<_> =
            (0..fields.len()).map(|index| format_ident!("field{index}")).collect();
        let pattern = match fields {
            Fields::Named(_) => quote!({ #(#members: #bindings),* }),
            Fields::Unnamed(_) => quote!((#(#bindings),*)),
            Fields::Unit => quote!(),
        };
        let slots: Vec<_> = bindings.iter().map(|field| format_ident!("printed_{field}")).collect();
        let hashes = slots.iter().map(|slot| quote!(# #slot));
        let output = match fields {
            Fields::Named(_) => quote!(#output_path { #(#members: #hashes),* }),
            Fields::Unnamed(_) => quote!(#output_path(#(#hashes),*)),
            Fields::Unit => output_path,
        };
        quote! {
            #pattern_path #pattern => {
                #(let #slots = crate::print::Print::print(#bindings);)*
                ::quote::quote!(#output)
            }
        }
    };
    let arms = match &input.data {
        Data::Struct(data) => vec![arm(&data.fields, quote!(Self), quote!(#path))],
        Data::Enum(data) => data
            .variants
            .iter()
            .map(|variant| {
                let name = &variant.ident;
                arm(&variant.fields, quote!(Self::#name), quote!(#path::#name))
            })
            .collect(),
        Data::Union(_) => {
            return Err(syn::Error::new_spanned(input, "Print supports structs and enums"));
        }
    };
    let name = &input.ident;
    Ok(quote! {
        impl crate::print::Print for #name {
            fn print(&self) -> ::proc_macro2::TokenStream {
                match self { #(#arms),* }
            }
        }
    })
}
