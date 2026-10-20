fn example() {
    let _pattern = {
        use ::umpire_dsl::path_build::{
            Factory as _, ActorFactoryValue as _, AppendPath as _,
        };
        let __first = ::umpire_dsl::path_build::Probe::new(at(0));
        let __factory = (&__first).factory();
        __factory
            .build(|__path| {
                let __built = ((((((__path)
                    .structure_with(|__pattern| {
                        (::umpire_dsl::Pattern::action(
                            User::__umpire_path_Send()
                                .__umpire_input_amount(
                                    Some(::umpire_dsl::InputDomain::values(vec![0, 2])),
                                )
                                .finish()
                                .map(::core::convert::Into::into),
                            None,
                            None,
                        ))
                            .with_outcomes(
                                vec![
                                    (::std::rc::Rc::new(| __outcome : & _ | matches!(__outcome,
                                    Outcome::Accepted)) as ::umpire_dsl::OutcomePredicate < _ >,
                                    stringify!(Outcome::Accepted) .into()),
                                    (::std::rc::Rc::new(| __outcome : & _ | matches!(__outcome,
                                    Outcome::Denied)) as ::umpire_dsl::OutcomePredicate < _ >,
                                    stringify!(Outcome::Denied) .into())
                                ],
                            )
                    }))
                    .structure_with(|__pattern| ::umpire_dsl::Pattern::action(
                        Deadline::__umpire_path_actions()
                            .map(::core::convert::Into::into),
                        None,
                        None,
                    )))
                    .structure_with(|__pattern| ::umpire_dsl::Pattern::Either(
                        vec![
                            { use ::umpire_dsl::path_build::PatternValue as _; let
                            __value = ::umpire_dsl::path_build::Probe::new(User::Poll);
                            (& __value).node() }, { use
                            ::umpire_dsl::path_build::PatternValue as _; let __value =
                            ::umpire_dsl::path_build::Probe::new(User::Cancel); (&
                            __value).node() }
                        ],
                    )))
                    .structure_with(|__pattern| ::umpire_dsl::Pattern::Repeat(
                        Box::new({
                            use ::umpire_dsl::path_build::PatternValue as _;
                            let __value = ::umpire_dsl::path_build::Probe::new(
                                User::Poll,
                            );
                            (&__value).node()
                        }),
                        0,
                        2,
                    )))
                    .structure_with(|__pattern| ::umpire_dsl::Pattern::Unordered(
                        vec![
                            { use ::umpire_dsl::path_build::PatternValue as _; let
                            __value = ::umpire_dsl::path_build::Probe::new(User::Poll);
                            (& __value).node() }, { use
                            ::umpire_dsl::path_build::PatternValue as _; let __value =
                            ::umpire_dsl::path_build::Probe::new(User::Cancel); (&
                            __value).node() }
                        ],
                    )))
                    .structure_with(|__pattern| ::umpire_dsl::Pattern::Gap(
                        ::umpire_dsl::Gap::Any,
                        Some(
                            ::std::rc::Rc::new(move |__action| {
                                false || __action == &(User::Cancel).into()
                            }),
                        ),
                    ));
                let __value = ::umpire_dsl::path_build::Probe::new(done);
                (&__value).append(__built)
            })
    };
}
