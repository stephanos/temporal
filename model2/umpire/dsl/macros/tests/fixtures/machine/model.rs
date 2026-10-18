pub(crate) fn run() -> umpire_testing::RunResult {
    let file = syn::parse_file(include_str!("../../../../tests/fixtures/machine/model.rs"))?;
    let mut input = file.items.into_iter().find_map(|item| match item {
        syn::Item::Impl(input) if input.attrs.iter().any(|attr| attr.path().is_ident("machine")) => Some(input),
        _ => None,
    }).ok_or("machine fixture is missing its impl")?;
    let attr = input.attrs.remove(0);
    let syn::Meta::List(args) = attr.meta else { return Err("machine arguments missing".into()); };
    let output = crate::machine::expand(args.tokens, &input)
        .map_err(|error| manyhow::ToTokensError::into_token_stream(error).to_string())?;
    let file = syn::parse2(output)?;
    Ok(umpire_testing::Files::from([("expansion.rs".into(), prettyplease::unparse(&file))]))
}
