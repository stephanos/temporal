use crate::{Diagnostic, Locate, Source};
use std::collections::BTreeSet;

/// The names a `#[machine]` module spells, as the macro receives it.
#[must_use]
pub fn source(module: &syn::ItemMod) -> Source {
    #[derive(Default)]
    struct Names {
        idents: BTreeSet<String>,
        source: Source,
    }
    impl Names {
        fn first(&mut self, name: String, span: proc_macro2::Span) {
            let location = span.start();
            self.source.first.entry(name).or_insert((location.line, location.column + 1));
        }
    }
    impl<'ast> syn::visit::Visit<'ast> for Names {
        fn visit_ident(&mut self, ident: &'ast syn::Ident) {
            self.idents.insert(ident.to_string());
            self.first(ident.to_string(), ident.span());
        }
        fn visit_lit_str(&mut self, literal: &'ast syn::LitStr) {
            self.first(literal.value(), literal.span());
        }
        fn visit_item_enum(&mut self, value: &'ast syn::ItemEnum) {
            for variant in &value.variants {
                self.source.variants.push((value.ident.to_string(), variant.ident.to_string()));
            }
            syn::visit::visit_item_enum(self, value);
        }
        fn visit_impl_item_fn(&mut self, value: &'ast syn::ImplItemFn) {
            self.source.methods.push(value.sig.ident.to_string());
            syn::visit::visit_impl_item_fn(self, value);
        }
    }
    let mut names = Names::default();
    syn::visit::Visit::visit_item_mod(&mut names, module);
    names.source.idents = names.idents.into_iter().collect();
    names.source.control = control(quote::ToTokens::to_token_stream(module));
    names.source
}

/// The first string literal carrying a control character other than a line break or tab.
fn control(tokens: proc_macro2::TokenStream) -> Option<Diagnostic> {
    tokens.into_iter().find_map(|token| match token {
        proc_macro2::TokenTree::Group(group) => control(group.stream()),
        proc_macro2::TokenTree::Literal(literal) => {
            syn::parse_str::<syn::LitStr>(&literal.to_string())
                .ok()
                .filter(|text| {
                    text.value().chars().any(|character| {
                        character.is_control() && !['\n', '\r', '\t'].contains(&character)
                    })
                })
                .map(|_| Diagnostic::span(literal.span(), "unsupported string control character"))
        }
        _ => None,
    })
}
