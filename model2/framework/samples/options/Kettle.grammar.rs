// Option A (custom): one `model!` proc macro with its own grammar, matching Kettle.scala line for line.
// Hypothetical framework "umpire"; does not compile.
//
// The macro owns the whole block, so: no `self.` (var names are rewritten to fields), `check`,
// `after`, `without`, `whenever`, `eventually` are keywords, `before` is bound for step claims,
// actors are modules so `user::press_button` reads like Scala's `user.pressButton`. Lifting is
// free: the macro has the tokens and emits the lifted tree beside the code.
//
// The price, and why this is the most expensive option to build and maintain: a custom grammar
// means a hand-written parser over `syn` tokens, rust-analyzer cannot complete or type-check
// inside the block, and every error message is only as good as the macro's own diagnostics.

use umpire::prelude::*;

// A kettle: a heating element the user switches on and off, which warms the water by one degree a
// step and fails over, beeping, once the water boils.
umpire::model! {
    // the boiling point of water, in Celsius
    const boiling_temperature = 100;
    // the coldest water the kettle starts with, in Celsius
    const coldest_temperature = -40;

    actor user {
        action press_button;
        action depress_button;
    }

    internal heater {
        step heat;
        step failover;
    }

    machine Model {
        var temperature = one_of(coldest_temperature..=40);
        var heating_on = false;
        var beeping = false;

        // one step of the state machine: any enabled action
        transitions {
            on user::press_button:
                if !heating_on { effects::press_button() }
            on heater::failover:
                if heating_on && temperature >= boiling_temperature { effects::failover() }
            on heater::heat:
                if heating_on && temperature < boiling_temperature { effects::heat() }
            on user::depress_button:
                if heating_on { effects::depress_button() }
        }

        effects {
            // turn on the heating element
            fn press_button() {
                heating_on = true;
                beeping = false;
            }
            // turn off the heating element and beep
            fn failover() {
                heating_on = false;
                beeping = true;
            }
            // heat up the water by 1C; `heating_on` holds and `beeping` doesn't, since heat only
            // fires while heating
            fn heat() {
                temperature += 1;
            }
            // turn off the heating element
            fn depress_button() {
                heating_on = false;
                beeping = false;
            }
        }

        invariants {
            beeps_only_when_off:
                if beeping {
                    check !heating_on;
                }
        }

        postconditions {
            warms_only_when_on:
                if !before.heating_on {
                    check temperature <= before.temperature;
                }
            fails_over_at_boiling:
                after heater::failover {
                    check temperature == boiling_temperature;
                    check beeping;
                }
        }

        // The user switches the kettle on but never off: one who switches it off keeps it from
        // boiling, whatever the heater does.
        scope never_switched_off = without user::depress_button;

        progress {
            // Every claim above also holds for a kettle that never heats; this one says it boils.
            // Switched on and left on, the water boils and the kettle beeps.
            boils:
                (whenever heating_on {
                    eventually beeping
                }) in never_switched_off
        }
    }
}

// Expansion sketch: a `struct Model` with Hash/Eq/Clone and the start set; an exhaustive
// `enum Action` and `fn step(&mut self, Action) -> Outcome` whose arms are the `on` blocks with
// `self.` inserted and a missing effect call meaning refusal; one fn per claim; scopes as action
// sets; progress as (whenever, eventually, scope); plus the lifted tree at compile time.
