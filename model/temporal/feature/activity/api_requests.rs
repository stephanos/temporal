// Supplemental request contexts share source types without duplicating descriptor catalogs.
macro_rules! supplemental_request {
    (@covers $covers:literal) => {{
        const _: () = {
            let bytes = $covers.as_bytes();
            let mut index = 0;
            let mut nonblank = false;
            while index < bytes.len() {
                if bytes[index].is_ascii_alphanumeric() {
                    nonblank = true;
                }
                index += 1;
            }
            assert!(nonblank, "supplemental request covers must name a source symbol");
        };
        $covers
    }};
    (uuid $ty:ty, $message:literal, $path:literal,
        get = $get:expr, set = $set:expr,
        expected = $expected:expr, covers = $covers:literal $(,)?) => {{
        let expectation: umpire_annotations::Expectation = $expected;
        umpire_annotations::Catalog::<$ty>::request($message)
            .field(umpire_annotations::Field::<$ty, String>::new(
                $path, umpire_annotations::Kind::Uuid, $get, $set,
            ).tag(&umpire_annotations::Consequence::Decided)
                .expects(&umpire_annotations::Expected {
                    expectation,
                    covers: supplemental_request!(@covers $covers).into(),
                }))
            .build()
    }};
    (required $ty:ty, $message:literal, $path:literal, $field:literal,
        get = $get:expr, set = $set:expr,
        expected = $expected:expr, covers = $covers:literal $(,)?) => {{
        let expectation: umpire_annotations::Expectation = $expected;
        umpire_annotations::Catalog::<$ty>::request($message)
            .field(umpire_annotations::Field::<$ty, Option<Vec<u8>>>::projected(
                $path,
                umpire_annotations::Kind::Bytes(umpire_annotations::BytesRules::default()),
                $get, $set,
            ).tag(&umpire_annotations::Consequence::Decided))
            .rule(umpire_annotations::Rule {
                presence: umpire_annotations::Presence::Required(vec![$field.into()]),
                expected: Some(umpire_annotations::Expected {
                    expectation,
                    covers: supplemental_request!(@covers $covers).into(),
                }),
            })
            .build()
    }};
}
