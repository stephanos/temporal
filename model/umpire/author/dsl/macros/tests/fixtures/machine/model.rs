pub(crate) fn run() -> umpire_testing::RunResult {
    let file = syn::parse_file(include_str!("../../../../tests/fixtures/machine/model.rs"))?;
    let mut input = file.items.into_iter().find_map(|item| match item {
        syn::Item::Mod(input) if input.attrs.iter().any(|attr| attr.path().is_ident("machine")) => Some(input),
        _ => None,
    }).ok_or("machine fixture is missing its module")?;
    input.attrs.remove(0);
    let output = crate::machine::expand(&proc_macro2::TokenStream::new(), &input)
        .map_err(|error| manyhow::ToTokensError::into_token_stream(error).to_string())?;
    let file = syn::parse2(output)?;
    Ok(umpire_testing::Files::from([("expansion.rs".into(), prettyplease::unparse(&file))]))
}
