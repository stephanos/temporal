use proc_macro_crate::{FoundCrate, crate_name};
use proc_macro2::{Group, TokenStream, TokenTree};
use quote::{format_ident, quote};

pub(crate) fn resolve(mut tokens: TokenStream, roots: &[(&str, &str)]) -> TokenStream {
    for (package, module) in roots {
        let canonical = package.replace('-', "_");
        let root = match crate_name(package) {
            Ok(FoundCrate::Name(name)) if name != canonical => {
                let name = format_ident!("{name}");
                Some(quote!(::#name))
            }
            Ok(_) => None,
            Err(_) => match crate_name("umpire") {
                Ok(FoundCrate::Name(name)) => {
                    let name = format_ident!("{name}");
                    let module = format_ident!("{module}");
                    Some(quote!(::#name::#module))
                }
                Ok(FoundCrate::Itself) => {
                    let module = format_ident!("{module}");
                    Some(quote!(crate::#module))
                }
                Err(_) => None,
            },
        };
        if let Some(root) = root {
            tokens = replace(tokens, &canonical, &root);
        }
    }
    tokens
}

fn replace(tokens: TokenStream, canonical: &str, root: &TokenStream) -> TokenStream {
    rewrite(tokens, canonical, root).0
}

fn rewrite(tokens: TokenStream, canonical: &str, root: &TokenStream) -> (TokenStream, bool) {
    let original = tokens.clone();
    let tokens: Vec<_> = tokens.into_iter().collect();
    let mut changed = false;
    let mut output = TokenStream::new();
    let mut index = 0;
    while index < tokens.len() {
        let matched = matches!(&tokens[index..], [TokenTree::Punct(first), TokenTree::Punct(second), TokenTree::Ident(name), ..]
            if first.as_char() == ':' && second.as_char() == ':' && name == canonical);
        if matched && absolute(&tokens, index) {
            output.extend(root.clone());
            changed = true;
            index += 3;
            continue;
        }
        match &tokens[index] {
            TokenTree::Group(group) => {
                let (stream, replaced) = rewrite(group.stream(), canonical, root);
                if replaced {
                    let mut nested = Group::new(group.delimiter(), stream);
                    nested.set_span(group.span());
                    output.extend([TokenTree::Group(nested)]);
                    changed = true;
                } else {
                    output.extend([TokenTree::Group(group.clone())]);
                }
            }
            token => output.extend([token.clone()]),
        }
        index += 1;
    }
    if changed { (output, true) } else { (original, false) }
}

fn absolute(tokens: &[TokenTree], index: usize) -> bool {
    match index.checked_sub(1).map(|previous| &tokens[previous]) {
        Some(TokenTree::Ident(name)) => {
            !matches!(name.to_string().as_str(), "self" | "Self" | "super" | "crate")
                && syn::parse2::<syn::Ident>(quote!(#name)).is_err()
        }
        Some(TokenTree::Group(_) | TokenTree::Literal(_)) => false,
        Some(TokenTree::Punct(mark)) if mark.as_char() == '>' => {
            if matches!(index.checked_sub(2).map(|before| &tokens[before]), Some(TokenTree::Punct(mark)) if matches!(mark.as_char(), '-' | '='))
            {
                return true;
            }
            let mut depth = 1;
            for before in (0..index - 1).rev() {
                if let TokenTree::Punct(mark) = &tokens[before] {
                    match mark.as_char() {
                        '>' => depth += 1,
                        '<' => depth -= 1,
                        _ => {}
                    }
                    if depth == 0 {
                        return matches!(before.checked_sub(1).map(|head| &tokens[head]), Some(TokenTree::Ident(name)) if name == "for" || name == "impl");
                    }
                }
            }
            false
        }
        _ => true,
    }
}

#[cfg(test)]
mod tests {
    use quote::quote;

    #[test]
    fn resolves_absolute_roots_without_rewriting_qualified_caller_paths()
    -> Result<(), proc_macro2::LexError> {
        let root = quote!(::author_kit::dsl);
        for (input, expected) in [
            ("::umpire_dsl::Value", "::author_kit::dsl::Value"),
            ("impl ::umpire_dsl::Trait for Model {}", "impl ::author_kit::dsl::Trait for Model {}"),
            (
                "impl<T> ::umpire_dsl::Trait for Model<T> {}",
                "impl<T> ::author_kit::dsl::Trait for Model<T> {}",
            ),
            (
                "impl<F: Fn() -> T> ::umpire_dsl::Trait for Model<F> {}",
                "impl<F: Fn() -> T> ::author_kit::dsl::Trait for Model<F> {}",
            ),
            ("Type<fn() -> T>::umpire_dsl::Value", "Type<fn() -> T>::umpire_dsl::Value"),
            (
                "vec![::umpire_dsl::Value, stringify!(*outcome == Outcome::Rejected(expected))]",
                "vec![::author_kit::dsl::Value, stringify!(*outcome == Outcome::Rejected(expected))]",
            ),
            ("impl Trait for ::umpire_dsl::Model {}", "impl Trait for ::author_kit::dsl::Model {}"),
            ("for<'a> ::umpire_dsl::Trait", "for<'a> ::author_kit::dsl::Trait"),
            ("fn value() -> ::umpire_dsl::Value {}", "fn value() -> ::author_kit::dsl::Value {}"),
            (
                "match value { _ => ::umpire_dsl::Value }",
                "match value { _ => ::author_kit::dsl::Value }",
            ),
            ("vec![::umpire_dsl::Value]", "vec![::author_kit::dsl::Value]"),
            (
                "#[derive(::umpire_dsl::State)] struct Model;",
                "#[derive(::author_kit::dsl::State)] struct Model;",
            ),
            ("foreign::umpire_dsl::Value", "foreign::umpire_dsl::Value"),
            ("<T as Trait>::umpire_dsl::Value", "<T as Trait>::umpire_dsl::Value"),
            ("Type<T>::umpire_dsl::Value", "Type<T>::umpire_dsl::Value"),
        ] {
            let input: proc_macro2::TokenStream = input.parse()?;
            let expected: proc_macro2::TokenStream = expected.parse()?;
            assert_eq!(
                super::replace(input, "umpire_dsl", &root).to_string(),
                expected.to_string()
            );
        }
        let canonical = quote!(::umpire_dsl::Value);
        assert_eq!(
            super::resolve(canonical.clone(), &[("umpire-dsl", "dsl")]).to_string(),
            canonical.to_string()
        );
        Ok(())
    }
}
