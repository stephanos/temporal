use umpire_dsl::{Outcome, Spell};

#[allow(clippy::unnecessary_wraps, reason = "fixture producers share a fallible runner signature")]
pub(crate) fn run() -> umpire_testing::RunResult {
    let mut output = String::new();
    let values: &[(&str, &dyn Spell)] = &[
        ("false", &false),
        ("true", &true),
        ("i8", &i8::MIN),
        ("i16", &i16::MIN),
        ("i32", &i32::MIN),
        ("i64", &i64::MIN),
        ("i128", &i128::MIN),
        ("isize", &-42_isize),
        ("u8", &u8::MAX),
        ("u16", &u16::MAX),
        ("u32", &u32::MAX),
        ("u64", &u64::MAX),
        ("u128", &u128::MAX),
        ("usize", &42_usize),
        ("verified", &Outcome::Verified),
        ("refuted", &Outcome::Refuted),
        ("unknown", &Outcome::Unknown),
        ("none", &None::<bool>),
        ("some", &Some((true, -3_i8))),
        ("nested", &Some(None::<bool>)),
        ("unit", &()),
        ("singleton", &(1_u8,)),
        ("tuple", &(false, -1_i32, Some(true))),
        ("wide_tuple", &(0_u8, 1_u8, 2_u8, 3_u8, 4_u8, 5_u8, 6_u8, 7_u8, 8_u8, 9_u8, 10_u8, 11_u8)),
        (
            "str",
            &"quote: \" slash: \\ newline: \n carriage: \r tab: \t nul: \0 control: \u{1b} unicode: é",
        ),
        ("string", &String::from("owned")),
        ("char", &'\''),
        ("slice", &&[true, false][..]),
        ("vec", &vec![Some(1_u8), None]),
        ("array", &[1_u16, 2, 3]),
        ("empty", &[0_u8; 0]),
        ("reference", &&42_i32),
    ];
    for (name, value) in values {
        output.push_str(name);
        output.push_str(": ");
        value.spell(&mut output);
        output.push('\n');
    }
    Ok(umpire_testing::Files::from([("values.txt".into(), output)]))
}
