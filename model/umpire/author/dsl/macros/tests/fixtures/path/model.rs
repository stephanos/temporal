pub(crate) fn run() -> umpire_testing::RunResult {
    let expanded = crate::path::expand(quote::quote! {
        at(0),
        User::Send { amount: (0 | 2), .. } => (Outcome::Accepted | Outcome::Denied),
        Deadline::*,
        (User::Poll | User::Cancel),
        [User::Poll; 0..=2],
        unordered![User::Poll, User::Cancel],
        ..!(User::Cancel),
        done
    })?;
    let file = syn::parse2(quote::quote! {
        fn example() { let _pattern = #expanded; }
    })?;
    Ok(umpire_testing::Files::from([("expansion.rs".into(), prettyplease::unparse(&file))]))
}
