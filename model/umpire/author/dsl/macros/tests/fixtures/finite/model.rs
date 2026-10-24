pub(crate) fn run() -> umpire_testing::RunResult {
    let mut output = String::new();
    for input in [
        syn::parse_quote! {
            pub enum Speed {
                Slow,
                Fast,
            }
        },
        syn::parse_quote! {
            pub struct Level(#[inputs(-1..=3)] i8);
        },
    ] {
        let expanded = crate::finite::expand(&input)
            .map_err(|error| manyhow::ToTokensError::into_token_stream(error).to_string())?;
        output.push_str(&prettyplease::unparse(&syn::parse2(expanded)?));
    }
    Ok(umpire_testing::Files::from([("expansion.rs".into(), output)]))
}
