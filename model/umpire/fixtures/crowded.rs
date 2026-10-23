//! More claims of each kind than a fixed table of callbacks would hold: the macro registers
//! exactly one callback per claim, so the interpreter has no claim cap.
use umpire_dsl::machine;

macro_rules! crowded {
    ([$($invariant:ident)*] [$($postcondition:ident)*] [$($progress:ident)*]) => {
        /// Seventy invariants, postconditions and progress claims over one counter.
        #[machine]
        pub mod crowded {
            use umpire_dsl::{Action, State, check, path};

            /// A counter up to two.
            #[derive(State)]
            pub struct Model {
                /// The count.
                pub value: u8,
            }

            /// Counting.
            #[derive(Action, Copy)]
            pub enum User {
                /// Adds one below two.
                Count,
            }

            impl Model {
                #[on(User)]
                fn user(&mut self, action: User) {
                    match action {
                        User::Count if self.value < 2 => self.value += 1,
                        User::Count => {}
                    }
                }

                $(
                    #[invariant]
                    fn $invariant(&self) {
                        check!(self.value < 2);
                    }
                )*

                $(
                    #[postcondition]
                    fn $postcondition(&self, before: &Self) {
                        check!(self.value > before.value);
                    }
                )*

                $(
                    #[progress]
                    fn $progress(&self) -> LeadsTo {
                        path![true, .., self.value == 2]
                    }
                )*
            }
        }
    };
}

crowded!([i0 i1 i2 i3 i4 i5 i6 i7 i8 i9 i10 i11 i12 i13 i14 i15 i16 i17 i18 i19 i20 i21 i22 i23 i24 i25 i26 i27 i28 i29 i30 i31 i32 i33 i34 i35 i36 i37 i38 i39 i40 i41 i42 i43 i44 i45 i46 i47 i48 i49 i50 i51 i52 i53 i54 i55 i56 i57 i58 i59 i60 i61 i62 i63 i64 i65 i66 i67 i68 i69 ] [p0 p1 p2 p3 p4 p5 p6 p7 p8 p9 p10 p11 p12 p13 p14 p15 p16 p17 p18 p19 p20 p21 p22 p23 p24 p25 p26 p27 p28 p29 p30 p31 p32 p33 p34 p35 p36 p37 p38 p39 p40 p41 p42 p43 p44 p45 p46 p47 p48 p49 p50 p51 p52 p53 p54 p55 p56 p57 p58 p59 p60 p61 p62 p63 p64 p65 p66 p67 p68 p69 ] [g0 g1 g2 g3 g4 g5 g6 g7 g8 g9 g10 g11 g12 g13 g14 g15 g16 g17 g18 g19 g20 g21 g22 g23 g24 g25 g26 g27 g28 g29 g30 g31 g32 g33 g34 g35 g36 g37 g38 g39 g40 g41 g42 g43 g44 g45 g46 g47 g48 g49 g50 g51 g52 g53 g54 g55 g56 g57 g58 g59 g60 g61 g62 g63 g64 g65 g66 g67 g68 g69 ]);
