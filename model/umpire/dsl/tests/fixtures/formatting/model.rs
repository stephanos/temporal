use std::{collections::BTreeMap, fmt::{self, Write}};

use umpire_dsl::{Action, State};

#[derive(Clone, Debug, Default, PartialEq, Eq, Hash)]
struct DebugOnly(u8);

#[derive(State, Default)]
struct Nested {
    value: DebugOnly,
}

#[derive(State)]
struct Model {
    values: Vec<Option<DebugOnly>>,
    map: BTreeMap<String, (DebugOnly, bool)>,
    nested: Nested,
}

#[derive(Action)]
enum Actor { Advance }

#[derive(Action)]
enum Empty {}

struct RefusingWriter;

impl Write for RefusingWriter {
    fn write_str(&mut self, _: &str) -> fmt::Result { Err(fmt::Error) }
}

pub(crate) fn run() -> umpire_testing::RunResult {
    fn empty_formats<T: fmt::Display + fmt::Debug>() {}
    empty_formats::<Empty>();
    assert_eq!(Empty::actions(), []);
    let model = Model {
        values: vec![Some(DebugOnly(1)), None],
        map: BTreeMap::from([("key".into(), (DebugOnly(2), true))]),
        nested: Nested { value: DebugOnly(3) },
    };
    assert_eq!(write!(RefusingWriter, "{model}"), Err(fmt::Error));
    assert_eq!(write!(RefusingWriter, "{}", Actor::Advance), Err(fmt::Error));
    assert_eq!(format!("{:?}", Actor::Advance), "Advance");
    let mut output = String::new();
    writeln!(output, "{model}")?;
    writeln!(output, "{}", Actor::Advance)?;
    Ok(umpire_testing::Files::from([("values.txt".into(), output)]))
}
