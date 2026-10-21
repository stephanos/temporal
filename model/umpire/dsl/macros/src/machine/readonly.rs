use proc_macro2::TokenStream;
use quote::quote;

pub(super) fn root(depth: usize) -> TokenStream {
    if depth == 0 {
        return quote!(self);
    }
    let mut root = quote!(super);
    for _ in 1..depth {
        root = quote!(super::#root);
    }
    root
}

pub(super) fn legacy_probe(state: &TokenStream) -> TokenStream {
    quote! {
        struct __Probe<T: ?Sized>(::core::marker::PhantomData<T>);
        struct __MutableModelMethod;
        trait __Readonly {
            type __Result;
            fn __probe(self) -> Self::__Result;
        }
        impl __Readonly for &__Probe<#state> {
            type __Result = __MutableModelMethod;
            fn __probe(self) -> Self::__Result { __MutableModelMethod }
        }
        impl<T: ?Sized> __Readonly for &&__Probe<T> {
            type __Result = ();
            fn __probe(self) {}
        }
    }
}

pub(super) fn probe(state: &syn::Ident, generics: &syn::Generics) -> TokenStream {
    let (_, arguments, _) = generics.split_for_impl();
    let state: syn::Type = syn::parse_quote!(#state #arguments);
    let generics = super::declared::resolved_bounds(generics, &state);
    let (implementation, _, constraints) = generics.split_for_impl();
    quote! {
        struct __UmpireReadonlyProbe<T: ?Sized>(::core::marker::PhantomData<T>);
        struct __MutableModelMethod;
        trait __UmpireReadonly {
            type __Result;
            fn __probe(self) -> Self::__Result;
        }
        impl #implementation __UmpireReadonly for &__UmpireReadonlyProbe<#state> #constraints {
            type __Result = __MutableModelMethod;
            fn __probe(self) -> Self::__Result { __MutableModelMethod }
        }
        impl<T: ?Sized> __UmpireReadonly for &&__UmpireReadonlyProbe<T> {
            type __Result = ();
            fn __probe(self) {}
        }
    }
}
