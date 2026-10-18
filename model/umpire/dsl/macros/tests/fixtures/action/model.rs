pub(crate) fn run() -> umpire_testing::RunResult {
    let output = crate::action::expand(&syn::parse_quote! {
        #[internal]
        pub enum Heater {
            Heat,
            Failover,
        }
    }).map_err(|error| manyhow::ToTokensError::into_token_stream(error).to_string())?;
    let file = syn::parse2(output)?;
    Ok(umpire_testing::Files::from([("expansion.rs".into(), prettyplease::unparse(&file))]))
}
