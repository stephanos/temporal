fn example() {
    let _pattern = {
        use ::umpire_dsl::path_build::{
            Factory as _, ActorFactoryValue as _, AppendPath as _,
        };
        let __first = ::umpire_dsl::path_build::Probe::new(at(0));
        let __factory = (&__first).factory();
        __factory
            .build(|__path| {
                let __value = ::umpire_dsl::path_build::Probe::new(done);
                (&__value)
                    .append(
                        ((((((__path)
                            .structure(
                                ::umpire_dsl::Pattern::Either(
                                    vec![
                                        (::umpire_dsl::Pattern::action(User::__umpire_path_Send()
                                        .amount(::umpire_dsl::InputDomain::values(vec![0, 2]))
                                        .finish().map(::core::convert::Into::into), None, None))
                                        .with_outcome(::std::rc::Rc::new(| __outcome |
                                        matches!(__outcome, Outcome::Accepted)),
                                        stringify!(Outcome::Accepted) .into()),
                                        (::umpire_dsl::Pattern::action(User::__umpire_path_Send()
                                        .amount(::umpire_dsl::InputDomain::values(vec![0, 2]))
                                        .finish().map(::core::convert::Into::into), None, None))
                                        .with_outcome(::std::rc::Rc::new(| __outcome |
                                        matches!(__outcome, Outcome::Denied)),
                                        stringify!(Outcome::Denied) .into())
                                    ],
                                ),
                            ))
                            .structure(
                                ::umpire_dsl::Pattern::action(
                                    Deadline::__umpire_path_actions()
                                        .map(::core::convert::Into::into),
                                    None,
                                    None,
                                ),
                            ))
                            .structure(
                                ::umpire_dsl::Pattern::Either(
                                    vec![
                                        { use ::umpire_dsl::path_build::PatternValue as _; let
                                        __value = ::umpire_dsl::path_build::Probe::new(User::Poll);
                                        (& __value).node() }, { use
                                        ::umpire_dsl::path_build::PatternValue as _; let __value =
                                        ::umpire_dsl::path_build::Probe::new(User::Cancel); (&
                                        __value).node() }
                                    ],
                                ),
                            ))
                            .structure(
                                ::umpire_dsl::Pattern::Repeat(
                                    Box::new({
                                        use ::umpire_dsl::path_build::PatternValue as _;
                                        let __value = ::umpire_dsl::path_build::Probe::new(
                                            User::Poll,
                                        );
                                        (&__value).node()
                                    }),
                                    0,
                                    2,
                                ),
                            ))
                            .structure(
                                ::umpire_dsl::Pattern::Unordered(
                                    vec![
                                        { use ::umpire_dsl::path_build::PatternValue as _; let
                                        __value = ::umpire_dsl::path_build::Probe::new(User::Poll);
                                        (& __value).node() }, { use
                                        ::umpire_dsl::path_build::PatternValue as _; let __value =
                                        ::umpire_dsl::path_build::Probe::new(User::Cancel); (&
                                        __value).node() }
                                    ],
                                ),
                            ))
                            .structure(
                                ::umpire_dsl::Pattern::Gap(
                                    ::umpire_dsl::Gap::Any,
                                    Some(
                                        ::std::rc::Rc::new(move |__action| {
                                            false || __action == &(User::Cancel).into()
                                        }),
                                    ),
                                ),
                            ),
                    )
            })
    };
}
