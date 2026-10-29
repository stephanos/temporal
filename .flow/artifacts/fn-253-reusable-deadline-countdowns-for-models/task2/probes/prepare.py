from pathlib import Path
import json
import re
import subprocess

BASE = '02d4f55660e4da54612d703f16bd92170c742539'
out = Path(__file__).resolve().parent
root = out.parents[2]
source = subprocess.check_output(['git', 'show', f'{BASE}:model/temporal/feature/activity/model.rs'], cwd=root, text=True)
counts = {}

def sub(text, pattern, replacement, expected, label):
    text, count = re.subn(pattern, replacement, text, flags=re.S)
    assert count == expected, (label, count, expected)
    counts[label] = count
    return text

def method(text, name, body):
    marker = f'        fn {name}('
    assert text.count(marker) == 1, name
    start = text.index(marker)
    opening = text.index('{', start)
    depth, end = 1, opening + 1
    while depth:
        depth += (text[end] == '{') - (text[end] == '}')
        end += 1
    counts['method:' + name] = 1
    return text[:opening + 1] + '\n' + body + '\n        }' + text[end:]

budgets = [('schedule_to_start', 'schedule_to_start'), ('start_to_close', 'start_to_close'),
           ('schedule_to_close', 'schedule_to_close'), ('heartbeat', 'heartbeat_timeout')]
helpers = '''
        fn arm(ticks: Option<Ticks>) -> Option<Remaining> {
            if ticks.is_some() { Some(Remaining { ticks: ticks.unwrap_or(Ticks(1)).0 }) } else { None }
        }
        fn tick(left: Option<Remaining>) -> Option<Remaining> {
            if left.is_some() && left.unwrap_or(Remaining { ticks: 0 }).ticks > 0 {
                Some(Remaining { ticks: left.unwrap_or(Remaining { ticks: 0 }).ticks - 1 })
            } else { left }
        }
        fn armed(left: Option<Remaining>) -> bool { left.is_some() }
        fn expired(left: Option<Remaining>) -> bool {
            left.is_some() && left.unwrap_or(Remaining { ticks: 0 }).ticks == 0
        }
        fn within(left: Option<Remaining>, configured: Option<Ticks>) -> bool {
            left.is_none() || (configured.is_some()
                && left.unwrap_or(Remaining { ticks: 0 }).ticks > 0
                && left.unwrap_or(Remaining { ticks: 0 }).ticks <= configured.unwrap_or(Ticks(1)).0)
        }
        fn descends(after: Option<Remaining>, before: Option<Remaining>) -> bool {
            after == Self::tick(before)
        }
'''
assert source.count('    impl Model {') == 1
text = source.replace('    impl Model {', '    impl Model {' + helpers)
for field, configured in budgets:
    literal = r'Some\(Remaining\s*\{\s*ticks:\s*world\.' + configured + r'\.unwrap_or\(Ticks\(1\)\)\.0,?\s*\}\)'
    # Replace every constructed configured budget, retaining all original control-flow guards.
    expected = {'schedule_to_start': 3, 'start_to_close': 1, 'schedule_to_close': 1, 'heartbeat': 2}[field]
    text = sub(text, literal, f'Self::arm(world.{configured})', expected, 'arm:' + field)
text = method(text, 'decrement', '\n'.join(f'            self.{field}_remaining = Self::tick(self.{field}_remaining);' for field, _ in budgets))
for field, _ in budgets:
    # Existing deadline predicates appear once each, except start-to-close selects the winning cause twice.
    expr = r'self\.' + field + r'_remaining\.is_some\(\)\s*&&\s*self\s*\.\s*' + field + r'_remaining\s*\.\s*unwrap_or\(Remaining\s*\{\s*ticks:\s*0\s*\}\)\s*\.\s*ticks\s*==\s*0'
    text = sub(text, expr, f'Self::expired(self.{field}_remaining)', 2 if field == 'start_to_close' else 1, 'expired:' + field)
    # The Tick enabling guard uses presence, preserving absent versus exhausted.
    # This occurrence is intentionally scoped to deadline(), not invariants.
start, end = text.index('        fn deadline('), text.index('        fn retry_or_finish(') if '        fn retry_or_finish(' in text else text.index('        fn decrement(')
part = text[start:end]
for field, _ in budgets:
    part = sub(part, r'self\.' + field + r'_remaining\.is_some\(\)', f'Self::armed(self.{field}_remaining)', 1, 'armed:' + field)
text = text[:start] + part + text[end:]
world_checks = '\n'.join(f'            check!(world.{configured}.is_none() || Ticks::contains(world.{configured}.unwrap_or(Ticks(1)).0));' for _, configured in budgets)
text = method(text, 'state_bounds', '''            check!(world.maximum_attempts >= 1 && world.maximum_attempts <= 3);
''' + world_checks)
text = sub(text, r'fn state_bounds\(', 'fn policy_in_range(', 1, 'policy-rename')
marker = '        #[invariant]\n        fn deadline_arming'
assert text.count(marker) == 1
state_laws = '''        #[invariant]
        fn attempt_within_policy(&self, world: &World) {
            check!(self.attempt >= 1 && self.attempt <= world.maximum_attempts);
        }

        #[invariant]
        fn countdown_bounds(&self, world: &World) {
'''
state_laws += '\n'.join(f'            check!(Self::within(self.{field}_remaining, world.{configured}));' for field, configured in budgets)
state_laws += '\n        }\n\n'
text = text.replace(marker, state_laws + marker)
retry = '''            if before.phase == Phase::Started && self.phase == Phase::Scheduled {
                check!(!before.cancel_requested && before.attempt < world.maximum_attempts);
                check!(self.phase == Phase::Scheduled && self.attempt == before.attempt + 1);
                check!(!self.cancel_requested);
                check!(self.schedule_to_start_remaining == Self::arm(world.schedule_to_start));
                check!(self.start_to_close_remaining.is_none() && self.heartbeat_remaining.is_none());
            } else {
'''
retry += '\n'.join(f'                check!(self.closed() || before.{field}_remaining.is_none() || Self::descends(self.{field}_remaining, before.{field}_remaining));' for field in ['schedule_to_start', 'start_to_close', 'heartbeat'])
retry += '\n            }\n            check!(self.closed() || before.schedule_to_close_remaining.is_none() || Self::descends(self.schedule_to_close_remaining, before.schedule_to_close_remaining));'
text = method(text, 'countdowns_decrement', retry)
# Priority remains the original implication structure, but asks whether ticking each previous timer expires it.
start = text.index('        fn tick_priority(')
end = text.index('        #[assumption]', start)
part = text[start:end]
for field, _ in budgets:
    pattern = r'before\.' + field + r'_remaining\.is_some\(\)\s*&&\s*before\s*\.\s*' + field + r'_remaining\s*\.\s*unwrap_or\(Remaining\s*\{\s*ticks:\s*0\s*\}\)\s*\.\s*ticks\s*<=\s*1'
    part = sub(part, pattern, f'Self::expired(Self::tick(before.{field}_remaining))', {'schedule_to_close': 4, 'start_to_close': 3, 'heartbeat': 3, 'schedule_to_start': 2}[field], 'priority:' + field)
text = text[:start] + part + text[end:]
(out / 'activity-baseline.rs').write_text(source)
(out / 'activity-local.rs').write_text(text)
shared = text.replace(helpers, '')
assert shared != text
for name in ['arm', 'tick', 'armed', 'expired', 'within', 'descends']:
    shared = shared.replace('Self::' + name + '(', 'countdown::' + name + '(')
shared = sub(shared, r'Action, Finite, Outcome,', 'countdown, Action, Finite, Outcome,', 1, 'library-import')
shared = sub(shared, r'check!\((countdown::within\(self\.\w+_remaining, world\.\w+\))\);', r'\1;', 4, 'library-within-laws')
shared = sub(shared, r'check!\(self.closed\(\) \|\| before\.(\w+)_remaining.is_none\(\) \|\| countdown::descends\(self\.\1_remaining, before\.\1_remaining\)\);',
    lambda m: f'if !self.closed() && before.{m[1]}_remaining.is_some() {{ countdown::descends(self.{m[1]}_remaining, before.{m[1]}_remaining); }}',
    4, 'library-descends-laws')
# Concrete wrappers stay Model-owned. The disposable DSL adapter's traits bridge them for native mechanics only.
trait_impls = '''
    impl countdown::Configured for Ticks {
        type Remaining = Remaining;
        fn ticks(self) -> u8 { self.0 }
        fn maximum() -> u8 { 3 }
    }
    impl countdown::Remaining for Remaining {
        fn ticks(self) -> u8 { self.ticks }
        fn from_ticks(ticks: u8) -> Self { Self { ticks } }
    }
'''
shared = shared.replace('    impl Model {', trait_impls + '\n    impl Model {', 1)
(out / 'activity-library.rs').write_text(shared)
owned = text.replace(helpers, '')
assert owned != text
for name in ['arm', 'tick', 'armed', 'expired', 'within', 'descends']:
    owned = owned.replace('Self::' + name + '(', 'countdown::' + name + '(')
owned = sub(owned, r'Action, Finite, Outcome,', 'countdown_proposed as countdown, countdown_types::{Ticks, Remaining}, Action, Finite, Outcome,', 1, 'shared-import')
owned = sub(owned, r'#\[derive\(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite\)\]\s*pub struct Ticks\(#\[inputs\(1\.\.=3\)\] pub u8\);', '', 1, 'remove-ticks')
owned = sub(owned, r'#\[derive\(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite\)\]\s*pub struct Remaining\s*\{\s*/// Number of ticks remaining\.\s*#\[inputs\(0\.\.=3\)\]\s*pub ticks: u8,\s*\}', '', 1, 'remove-remaining')
owned = sub(owned, r'Option<Ticks>', 'Option<Ticks<3>>', 4, 'shared-configured-types')
owned = sub(owned, r'Option<Remaining>', 'Option<Remaining<3>>', 4, 'shared-remaining-types')
owned = sub(owned, r'Ticks::contains', 'Ticks::<3>::contains', 4, 'shared-contains')
owned = sub(owned, r'check!\((countdown::within\(self\.\w+_remaining, world\.\w+\))\);', r'\1;', 4, 'shared-within-laws')
owned = sub(owned, r'check!\(self.closed\(\) \|\| before\.(\w+)_remaining.is_none\(\) \|\| countdown::descends\(self\.\1_remaining, before\.\1_remaining\)\);',
    lambda m: f'if !self.closed() && before.{m[1]}_remaining.is_some() {{ countdown::descends(self.{m[1]}_remaining, before.{m[1]}_remaining); }}',
    4, 'shared-descends-laws')
(out / 'activity-shared.rs').write_text(owned)
# include! cannot introduce inner module documentation.
for name in ['baseline', 'local', 'library', 'shared']:
    original = (out / f'activity-{name}.rs').read_text()
    (out / f'native-{name}.rs').write_text(original.replace('//!', '//'))
(out / 'rewrite-counts.json').write_text(json.dumps(counts, indent=2) + '\n')

# Each machine isolates one syntax form; no earlier refusal can hide its target.
header = '''use umpire_dsl::machine;
#[machine]
pub mod probe {
    use umpire_dsl::{Action, Finite, State, World, check};
    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite)]
    pub struct Ticks(#[inputs(1..=3)] pub u8);
    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite)]
    pub struct Remaining { #[inputs(0..=3)] pub ticks: u8 }
    #[derive(World)]
    pub struct World { pub timeout: Option<Ticks> }
    #[derive(State)]
    pub struct Model { pub left: Option<Remaining> }
    #[derive(Action, Copy)]
    pub enum Clock { Tick }
'''
def machine(name, extra='', initial='None', effect='', helper='', claim=''):
    body = header + extra + '''
    impl Model {
        #[starts]
        fn initial(world: &World) -> impl IntoIterator<Item = Self> {
            [Self { left: INITIAL }]
        }
        #[on(Clock)]
        fn clock(&mut self, action: Clock, world: &World) {
            match action { Clock::Tick => { EFFECT } }
        }
        HELPER
        CLAIM
    }
}
'''
    body = body.replace('INITIAL', initial).replace('EFFECT', effect).replace('HELPER', helper).replace('CLAIM', claim)
    (out / (name + '.rs')).write_text(body)

machine('control', effect='if self.armed() { self.left = None; }', helper='fn armed(&self) -> bool { self.left.is_some() }')
arm_helper = helpers[helpers.index('        fn arm'):helpers.index('        fn tick')]
machine('local-arm-initial', initial='Self::arm(world.timeout)', helper=arm_helper)
machine('local-arm', effect='self.left = Self::arm(world.timeout);', helper=arm_helper)
machine('local-tick', effect='self.left = Self::tick(self.left);', helper=helpers[helpers.index('        fn tick'):helpers.index('        fn armed')])
machine('receiver-args', effect='if self.is_expired(self.left) { self.left = None; }', helper='fn is_expired(&self, left: Option<Remaining>) -> bool { left.is_some() }')
machine('library-assignment', extra='use umpire_dsl::countdown;', effect='self.left = countdown::tick(self.left);')
machine('library-condition', extra='use umpire_dsl::countdown;', effect='if countdown::armed(self.left) { self.left = None; }')
machine('library-within-law', extra='use umpire_dsl::countdown;', claim='#[invariant] fn bounds(&self, world: &World) { countdown::within(self.left, world.timeout); }')
machine('library-descends-law', extra='use umpire_dsl::countdown;', claim='#[postcondition] fn decreases(&self, before: &Self) { countdown::descends(self.left, before.left); }')
machine('unused-generic', extra='pub struct Boxed<const N: u8>(pub u8);')
machine('local-generic', extra='#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite)] pub struct Boxed<const N: u8>(#[inputs(0..=3)] pub u8);')
machine('local-alias', extra='type Left = Remaining;')
machine('local-trait', extra=trait_impls + 'use umpire_dsl::countdown;')
machine('project-library', extra='use crate::temporal_countdown;', effect='if temporal_countdown::armed(self.left) { self.left = None; }')
# Imported shared wrappers test signature and state typing separately from function calls.
for generic, label in [('Ticks', 'shared-ticks'), ('Remaining', 'shared-remaining')]:
    content = header
    if generic == 'Ticks':
        content = content.replace('pub struct World { pub timeout: Option<Ticks> }', 'pub struct World { pub timeout: Option<SharedTicks<3>> }')
        content += 'use umpire_dsl::countdown_types::Ticks as SharedTicks;\n'
    else:
        content = content.replace('pub struct Model { pub left: Option<Remaining> }', 'pub struct Model { pub left: Option<SharedRemaining<3>> }')
        content += 'use umpire_dsl::countdown_types::Remaining as SharedRemaining;\n'
    content += '''impl Model {
        #[starts] fn initial(world: &World) -> impl IntoIterator<Item = Self> { [Self { left: None }] }
        #[on(Clock)] fn clock(&mut self, action: Clock) { match action { Clock::Tick => {} } }
    }
}
'''
    (out / (label + '.rs')).write_text(content)
(out / 'parse-error.rs').write_text('#[machine]\npub mod broken {\n')
manifest = {name + '.rs': {'outcome': 'admitted', 'message': ''} for name in ['activity-baseline', 'control', 'local-arm-initial', 'unused-generic']}
messages = {
    'activity-local': 'unsupported expression ' + chr(96) + 'Self :: arm (world . start_to_close)',
    'activity-library': 'trait or generic impl in machine module is unsupported',
    'activity-shared': 'unsupported type ' + chr(96) + 'Option<Ticks<3>>' + chr(96),
    'local-arm': 'unsupported expression ' + chr(96) + 'Self :: arm (world . timeout)',
    'local-tick': 'unsupported expression ' + chr(96) + 'Self :: tick (self . left)',
    'receiver-args': 'only pure helper calls with an optional trailing World are admitted',
    'library-assignment': 'unsupported expression ' + chr(96) + 'countdown :: tick (self . left)',
    'library-condition': 'unsupported expression ' + chr(96) + 'countdown :: armed (self . left)',
    'library-within-law': 'claim body supports check! and conditional checks only',
    'library-descends-law': 'claim body supports check! and conditional checks only',
    'local-generic': 'bounded newtypes must be nongeneric',
    'local-alias': 'unsupported declaration in machine module',
    'local-trait': 'trait or generic impl in machine module is unsupported',
    'project-library': 'unsupported expression ' + chr(96) + 'temporal_countdown :: armed (self . left)',
    'shared-ticks': 'unsupported type ' + chr(96) + 'SharedTicks<3>',
    'shared-remaining': 'unsupported type ' + chr(96) + 'SharedRemaining<3>',
}
manifest.update({name + '.rs': {'outcome': 'model_refusal', 'message': message} for name, message in messages.items()})
manifest['parse-error.rs'] = {'outcome': 'parse_error', 'message': 'cannot parse string into token stream'}
(out / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')

def method_lines(content, name):
    marker = f'        fn {name}('
    if marker not in content:
        return None
    assert content.count(marker) == 1, name
    start = content.index(marker)
    opening = content.index('{', start)
    depth, end = 1, opening + 1
    while depth:
        depth += (content[end] == '{') - (content[end] == '}')
        end += 1
    return content[start:end].count('\n') + 1

line_counts = {'base': BASE, 'variants': {}}
names = ['initial', 'worker', 'deadline', 'retry', 'decrement', 'state_bounds',
         'policy_in_range', 'attempt_within_policy', 'countdown_bounds',
         'countdowns_decrement', 'tick_priority']
for variant in ['baseline', 'local', 'library', 'shared']:
    content = (out / f'activity-{variant}.rs').read_text()
    line_counts['variants'][variant] = {
        'total': len(content.splitlines()),
        'methods': {name: method_lines(content, name) for name in names},
    }
(out / 'line-counts.json').write_text(json.dumps(line_counts, indent=2) + '\n')
