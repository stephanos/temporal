//! Constructor tokens consumed by the DSL macro.
use proc_macro2::{Literal, TokenStream};
use quote::quote;
use std::collections::BTreeMap;

/// Constructor tokens for a tree value.
pub trait Print {
    /// Print the value as a Rust constructor expression.
    fn print(&self) -> TokenStream;
}

impl Print for String {
    fn print(&self) -> TokenStream {
        quote!(::std::string::String::from(#self))
    }
}
impl Print for bool {
    fn print(&self) -> TokenStream {
        quote!(#self)
    }
}
impl Print for usize {
    fn print(&self) -> TokenStream {
        let value = Literal::usize_suffixed(*self);
        quote!(#value)
    }
}
impl Print for u64 {
    fn print(&self) -> TokenStream {
        let value = Literal::u64_suffixed(*self);
        quote!(#value)
    }
}
impl Print for i128 {
    fn print(&self) -> TokenStream {
        // A negative literal may split into `-` and a positive one, which i128::MIN overflows.
        match self.checked_neg() {
            None => quote!(::core::primitive::i128::MIN),
            Some(positive) if *self < 0 => {
                let value = Literal::i128_suffixed(positive);
                quote!(-#value)
            }
            Some(_) => {
                let value = Literal::i128_suffixed(*self);
                quote!(#value)
            }
        }
    }
}
impl<T: Print> Print for Vec<T> {
    fn print(&self) -> TokenStream {
        let values = self.iter().map(Print::print);
        quote!(::std::vec![#(#values),*])
    }
}
impl<T: Print> Print for Box<T> {
    fn print(&self) -> TokenStream {
        let value = self.as_ref().print();
        quote!(::std::boxed::Box::new(#value))
    }
}
impl<T: Print> Print for Option<T> {
    fn print(&self) -> TokenStream {
        self.as_ref().map_or_else(
            || quote!(::core::option::Option::None),
            |value| {
                let value = value.print();
                quote!(::core::option::Option::Some(#value))
            },
        )
    }
}
impl<T: Print, E: Print> Print for Result<T, E> {
    fn print(&self) -> TokenStream {
        match self {
            Ok(value) => {
                let value = value.print();
                quote!(::core::result::Result::Ok(#value))
            }
            Err(error) => {
                let error = error.print();
                quote!(::core::result::Result::Err(#error))
            }
        }
    }
}
impl<A: Print, B: Print> Print for (A, B) {
    fn print(&self) -> TokenStream {
        let (first, second) = (self.0.print(), self.1.print());
        quote!((#first, #second))
    }
}
impl<V: Print> Print for BTreeMap<String, V> {
    fn print(&self) -> TokenStream {
        let entries = self.iter().map(|(key, value)| (key.clone(), value).print());
        quote!(::std::collections::BTreeMap::from([#(#entries),*]))
    }
}
impl<T: Print> Print for &T {
    fn print(&self) -> TokenStream {
        (*self).print()
    }
}
