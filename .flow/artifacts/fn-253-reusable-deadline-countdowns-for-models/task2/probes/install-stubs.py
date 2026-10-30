from pathlib import Path
import shutil

out = Path(__file__).resolve().parent
root = out.parents[2]
assert root.name == 'fn2532-spike' and root.parent.name == '.worktrees', root
dsl = root / 'model/umpire/author/dsl/src'
lib = dsl / 'lib.rs'
text = lib.read_text()
marker = '\npub mod countdown;\npub mod countdown_types;\n'
if marker in text:
    assert text.count(marker) == 1
else:
    assert 'pub mod countdown;' not in text and 'pub mod countdown_types;' not in text
    lib.write_text(text + marker)
shutil.copyfile(out / 'trait_countdown.rs', dsl / 'countdown.rs')
shutil.copyfile(out / 'shared-types.rs', dsl / 'countdown_types.rs')
shutil.copyfile(out / 'countdown-proposed.rs', dsl / 'countdown_proposed.rs')
text = lib.read_text()
if '\npub mod countdown_proposed;\n' not in text:
    lib.write_text(text + '\npub mod countdown_proposed;\n')
# A real prospective Temporal crate/module owner, rather than an unattached .rs sketch.
temporal = root / 'model/temporal/realization/src'
temporal_lib = temporal / 'lib.rs'
text = temporal_lib.read_text()
declaration = '\npub mod countdown_spike;\n'
assert text.count(declaration) <= 1
if declaration not in text:
    temporal_lib.write_text(text + declaration)
shutil.copyfile(out / 'temporal-countdown.rs', temporal / 'countdown_spike.rs')
