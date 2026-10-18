pub(crate) fn run() -> umpire_testing::RunResult {
    let output = crate::state::expand(&syn::parse_quote! {
        pub struct Kettle {
            #[starts(COLDEST_TEMPERATURE..=40)]
            temperature: i32,
            heating_on: bool,
            beeping: bool,
        }
    }).map_err(|error| manyhow::ToTokensError::into_token_stream(error).to_string())?;
    let file = syn::parse2(output)?;
    Ok(umpire_testing::Files::from([("expansion.rs".into(), prettyplease::unparse(&file))]))
}
