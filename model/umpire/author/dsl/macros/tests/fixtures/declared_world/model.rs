pub(crate) fn run() -> umpire_testing::RunResult {
    let file = syn::parse_file(include_str!("../../../../../suite/tests/declared_world.rs"))?;
    let mut module = file.items.into_iter().find_map(|item| match item {
        syn::Item::Mod(module) => Some(module),
        _ => None,
    }).ok_or("declared World fixture is missing its module")?;
    module.attrs.clear();
    let output = crate::machine::expand(&proc_macro2::TokenStream::new(), &module)
        .map_err(|error| manyhow::ToTokensError::into_token_stream(error).to_string())?;
    let file = syn::parse2(output)?;
    Ok(umpire_testing::Files::from([("expansion.rs".into(), prettyplease::unparse(&file))]))
}
