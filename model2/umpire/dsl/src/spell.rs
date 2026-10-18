use crate::Outcome;

/// The canonical Rust spelling of a Model value in every Umpire output.
pub trait Spell {
    /// Appends the value, spelling records as `Name { field: value }` and enums as `Type::Variant`.
    fn spell(&self, out: &mut String);
}

impl Spell for bool {
    fn spell(&self, out: &mut String) {
        out.push_str(if *self { "true" } else { "false" });
    }
}

macro_rules! integers {
    ($($integer:ty),+ $(,)?) => {$(
        impl Spell for $integer {
            fn spell(&self, out: &mut String) {
                out.push_str(&self.to_string());
            }
        }
    )+};
}

integers!(i8, i16, i32, i64, i128, isize, u8, u16, u32, u64, u128, usize);

impl Spell for Outcome {
    fn spell(&self, out: &mut String) {
        out.push_str(match self {
            Self::Verified => "Outcome::Verified",
            Self::Refuted => "Outcome::Refuted",
            Self::Unknown => "Outcome::Unknown",
        });
    }
}

impl<T: Spell> Spell for Option<T> {
    fn spell(&self, out: &mut String) {
        match self {
            None => out.push_str("Option::None"),
            Some(value) => {
                out.push_str("Option::Some { 0: ");
                value.spell(out);
                out.push_str(" }");
            }
        }
    }
}

impl Spell for () {
    fn spell(&self, out: &mut String) {
        out.push_str("()");
    }
}

impl<T: Spell> Spell for (T,) {
    fn spell(&self, out: &mut String) {
        out.push('(');
        self.0.spell(out);
        out.push_str(",)");
    }
}

macro_rules! tuples {
    ($first:ident : $first_index:tt, $($value:ident : $index:tt),+ $(,)?) => {
        impl<$first: Spell, $($value: Spell),+> Spell for ($first, $($value),+) {
            fn spell(&self, out: &mut String) {
                out.push('(');
                self.$first_index.spell(out);
                $(out.push_str(", "); self.$index.spell(out);)+
                out.push(')');
            }
        }
    };
}

tuples!(A: 0, B: 1);
tuples!(A: 0, B: 1, C: 2);
tuples!(A: 0, B: 1, C: 2, D: 3);
tuples!(A: 0, B: 1, C: 2, D: 3, E: 4);
tuples!(A: 0, B: 1, C: 2, D: 3, E: 4, F: 5);
tuples!(A: 0, B: 1, C: 2, D: 3, E: 4, F: 5, G: 6);
tuples!(A: 0, B: 1, C: 2, D: 3, E: 4, F: 5, G: 6, H: 7);
tuples!(A: 0, B: 1, C: 2, D: 3, E: 4, F: 5, G: 6, H: 7, I: 8);
tuples!(A: 0, B: 1, C: 2, D: 3, E: 4, F: 5, G: 6, H: 7, I: 8, J: 9);
tuples!(A: 0, B: 1, C: 2, D: 3, E: 4, F: 5, G: 6, H: 7, I: 8, J: 9, K: 10);
tuples!(A: 0, B: 1, C: 2, D: 3, E: 4, F: 5, G: 6, H: 7, I: 8, J: 9, K: 10, L: 11);

impl<T: Spell + ?Sized> Spell for &T {
    fn spell(&self, out: &mut String) {
        T::spell(self, out);
    }
}

impl Spell for str {
    fn spell(&self, out: &mut String) {
        out.push('"');
        for character in self.chars() {
            escape(character, '"', out);
        }
        out.push('"');
    }
}

impl Spell for String {
    fn spell(&self, out: &mut String) {
        self.as_str().spell(out);
    }
}

impl Spell for char {
    fn spell(&self, out: &mut String) {
        out.push('\'');
        escape(*self, '\'', out);
        out.push('\'');
    }
}

fn escape(character: char, quote: char, out: &mut String) {
    match character {
        '\0' => out.push_str("\\0"),
        '\n' => out.push_str("\\n"),
        '\r' => out.push_str("\\r"),
        '\t' => out.push_str("\\t"),
        '\\' => out.push_str("\\\\"),
        value if value == quote => {
            out.push('\\');
            out.push(value);
        }
        value if value.is_control() => out.extend(value.escape_default()),
        value => out.push(value),
    }
}

impl<T: Spell> Spell for [T] {
    fn spell(&self, out: &mut String) {
        out.push('[');
        for (index, value) in self.iter().enumerate() {
            if index > 0 {
                out.push_str(", ");
            }
            value.spell(out);
        }
        out.push(']');
    }
}

impl<T: Spell> Spell for Vec<T> {
    fn spell(&self, out: &mut String) {
        self.as_slice().spell(out);
    }
}

impl<T: Spell, const N: usize> Spell for [T; N] {
    fn spell(&self, out: &mut String) {
        self.as_slice().spell(out);
    }
}
