"""Revision-bound vocabulary and semantic identifier audit for fn-111 task 1 (R1-R3).

Reads only; writes vocabulary-audit.json beside this file. Exit status is nonzero
when any check fails.
"""
import collections
import hashlib
import json
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[3]
HERE = pathlib.Path(__file__).resolve().parent
OUT = pathlib.Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else HERE
OUT.mkdir(parents=True, exist_ok=True)
BASE = '29917069e089dc0739ec091b18e99161245b9bd5'
CONSOLIDATION = '83d143293c'
GUIDES = ['SPEC', 'ARCHITECTURE', 'CLI', 'TUTORIAL', 'README']
ANCHOR = 'SPEC.md#productvocabulary-ubiquitous-language'
FLOW_SPEC = ROOT / '.flow/specs/fn-111-gomad-consolidate-vocabulary-and-update.md'

errors = []


def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT, text=True)


def sha256(text):
    return 'sha256:' + hashlib.sha256(text.encode()).hexdigest()


def check(name, passed, **detail):
    if not passed:
        errors.append({'check': name, **detail})
    return passed


def identifiers(text):
    return re.findall(r'\[([A-Z][A-Z0-9]*(?:\.[A-Z0-9]+)*)\]', text)


def section(text, start, end):
    return text.split(start, 1)[1].split(end, 1)[0]


docs = {name: (ROOT / f'tools/gomad3/{name}.md').read_text() for name in GUIDES}
spec, readme = docs['SPEC'], docs['README']
old_spec = git('show', f'{BASE}:tools/gomad3/SPEC.md')
old_glossary = git('show', f'{BASE}:tools/gomad3/GLOSSARY.md')

# Source revisions and file hashes.
consolidation_parent = git('rev-parse', f'{CONSOLIDATION}^').strip()
check('baseline is the consolidation parent', consolidation_parent == BASE, parent=consolidation_parent)
retained = {name: (HERE / f'baseline-{name}.md').read_text() for name in ['SPEC', 'GLOSSARY']}
check('retained baseline SPEC matches revision', retained['SPEC'] == old_spec)
check('retained baseline GLOSSARY matches revision', retained['GLOSSARY'] == old_glossary)
glossary_in_head = subprocess.run(['git', 'cat-file', '-e', 'HEAD:tools/gomad3/GLOSSARY.md'], cwd=ROOT, capture_output=True)
check('glossary deleted from HEAD', glossary_in_head.returncode != 0)
check('glossary absent from working tree', not (ROOT / 'tools/gomad3/GLOSSARY.md').exists())

# R1: the 25 original Language entries, their retained assessments, and their current disposition.
language = section(old_glossary, '## Language', '\n## ')
terms = re.findall(r'^\*\*([^*]+)\*\*', language, re.M)
check('original Language entry count', len(terms) == 25 and len(set(terms)) == 25, count=len(terms))

decisions = section(FLOW_SPEC.read_text(), '## Terminology Decisions', '\n## ')
rows = [[cell.strip() for cell in line.strip().strip('|').split('|')] for line in decisions.splitlines() if line.startswith('|')][2:]
check('every original entry is assessed in order', [row[0] for row in rows] == terms, assessed=[row[0] for row in rows])
check('every assessment has a recommendation and rationale', all(len(row) == 3 and all(row) for row in rows))

vocabulary = section(spec, '## [PRODUCT.VOCABULARY]', '## [PLATFORM]')
definitions, aliases = {}, {}
duplicate_heads = []
for line in vocabulary.splitlines():
    head = re.match(r'^\*\*([^*]+)\*\*(?: \(also called \*\*([^*]+)\*\*\))?: (.+)$', line)
    if not head:
        check('vocabulary line is a heading, the preamble, or a definition',
              not line.startswith('**'), line=line[:80])
        continue
    term, alias, body = head.groups()
    if term in definitions or term in aliases:
        duplicate_heads.append(term)
    definitions[term] = body
    if alias:
        if alias in definitions or alias in aliases:
            duplicate_heads.append(alias)
        aliases[alias] = term
check('each vocabulary term has one owner', not duplicate_heads, duplicates=duplicate_heads)

dispositions = []
for term in terms:
    if term == 'Parity Case':
        dispositions.append({'term': term, 'disposition': 'historical', 'owner': 'README.md Simulation contract'})
    elif term in definitions:
        dispositions.append({'term': term, 'disposition': 'definition', 'owner': term})
    elif term in aliases:
        dispositions.append({'term': term, 'disposition': 'alias', 'owner': aliases[term]})
    else:
        dispositions.append({'term': term, 'disposition': 'missing'})
current = [item for item in dispositions if item['disposition'] in ('definition', 'alias')]
check('all 24 current concepts survive as definitions or aliases', len(current) == 24,
      missing=[item['term'] for item in dispositions if item['disposition'] == 'missing'])

contract = section(readme, '## Simulation contract', '\n## ')
parity = next((paragraph for paragraph in contract.split('\n\n') if 'Parity Case' in paragraph), '')
check('README names Parity Case in its historical narrative',
      bool(parity) and 'historical' in parity and 'removed by fn-81' in parity)
check('Parity Case is absent from the current vocabulary',
      'Parity Case' not in spec and 'Parity Case' not in definitions and 'Parity Case' not in aliases)

# R2: single vocabulary owner and identifier preservation.
original_ids, current_ids = identifiers(old_spec), identifiers(spec)
repeated = lambda values: sorted(key for key, count in collections.Counter(values).items() if count > 1)
check('original identifier count', len(original_ids) == 123, count=len(original_ids))
check('original identifiers are unique', not repeated(original_ids), repeated=repeated(original_ids))
check('current identifiers are unique', not repeated(current_ids), repeated=repeated(current_ids))
check('identifiers preserved in original order', original_ids == current_ids,
      removed=[item for item in original_ids if item not in current_ids],
      added=[item for item in current_ids if item not in original_ids])
table_ids = lambda text: re.findall(r'^\| `\[([A-Z0-9.]+)\]` \|', text, re.M)
check('command-table identifiers preserved in order', table_ids(old_spec) == table_ids(spec) and bool(table_ids(spec)))
check('vocabulary heading anchor is stable', '## [PRODUCT.VOCABULARY] Ubiquitous Language' in spec)
links = {name: ANCHOR in body or (name == 'SPEC') for name, body in docs.items()}
check('current guides link the SPEC vocabulary', all(links.values()), links=links)
stale = {name: [term for term in ['GLOSSARY.md', 'Combined ChoiceExploration'] if term in body] for name, body in docs.items()}
check('no guide references the deleted glossary or malformed term', not any(stale.values()), stale=stale)

# R3: boundary wording, bound to the implementation it describes.
def has(term, *phrases):
    body = definitions.get(term, '')
    return all(phrase in body for phrase in phrases)

def lacks(term, *phrases):
    body = definitions.get(term, '')
    return bool(body) and not any(phrase.lower() in body.lower() for phrase in phrases)

source = lambda path: (ROOT / path).read_text()
tape, sim_spec = source('tools/gomad3/choice/tape.go'), source('tools/gomad3sim/spec.go')
boundaries = {
    'Target is the selection, Prepared Target the immutable executable':
        has('Target', 'user-selected') and lacks('Target', 'immutable') and has('Prepared Target', 'immutable executable', 'bound execution inputs'),
    'canonical Prepared Target is never written prepared Target': 'prepared Target' not in spec,
    'Campaign is an effort, not its plan':
        has('Campaign', 'bounded exploration effort', 'A Campaign is not its plan') and has('Portable Plan', 'selected work'),
    'Choice Trace is observed evidence, not a control plan':
        has('Choice Trace', 'observations', 'not itself a replay control plan'),
    'Decision Tape holds branching Choices from a complete Choice Trace':
        has('Decision Tape', 'branching', 'complete Choice Trace', 'bound to execution identities'),
    'implementation: tape omits observations and nonbranching records':
        'record.Flags&FlagObservation != 0 || record.Alternatives < 2' in tape,
    'exact replay uses a complete tape; exploration forces a finite prefix':
        has('Choice Replay Plan', 'Exact replay uses a complete Decision Tape', 'finite prefix', 'from the Seed')
        and has('Exact Replay', 'complete consumption', 'recorded platform'),
    'implementation: exact and prefix plans validate separately':
        'func ValidateReplayPlan(' in tape and 'func ValidatePrefixReplayPlan(' in tape,
    'Backend is the mechanism': has('Backend', 'execution mechanism', 'in-process', 'process execution') and lacks('Backend', 'fidelity', 'isolation'),
    'Fidelity is the claim, constrained by Backend':
        has('Fidelity', 'recorded separately from the selected Backend', 'Both Backends support Model Fidelity', 'only the process Backend supports Hard Isolation')
        and lacks('Fidelity', 'independently'),
    'Model Fidelity claims no hard isolation': has('Model Fidelity', 'does not claim fresh arbitrary package globals or hard cleanup'),
    'Hard Isolation is limited to the process Backend': has('Hard Isolation', 'process-Backend guarantee'),
    'implementation: hard isolation requires the process backend':
        'hard isolation fidelity requires the process backend' in sim_spec,
    'World is separate from host I/O and application state':
        has('World', 'application-declared', 'owns neither host input/output nor application state'),
}
for name, passed in boundaries.items():
    check(name, passed)

paths = ['tools/gomad3/SPEC.md', 'tools/gomad3/README.md', 'tools/gomad3/GLOSSARY.md']
whitespace = subprocess.run(['git', 'diff', '--check', BASE, '--', *paths], cwd=ROOT, capture_output=True, text=True)
check('scoped baseline-to-current whitespace', whitespace.returncode == 0, output=whitespace.stdout + whitespace.stderr)

report = {
    'baseline_revision': BASE,
    'consolidation_revision': git('rev-parse', CONSOLIDATION).strip(),
    'head_revision': git('rev-parse', 'HEAD').strip(),
    'baseline_sha256': {'SPEC': sha256(old_spec), 'GLOSSARY': sha256(old_glossary)},
    'baseline_blob': {'SPEC': git('rev-parse', f'{BASE}:tools/gomad3/SPEC.md').strip(),
                      'GLOSSARY': git('rev-parse', f'{BASE}:tools/gomad3/GLOSSARY.md').strip()},
    'working_tree_sha256': {'SPEC': sha256(spec), 'README': sha256(readme)},
    'head_sha256': {name: sha256(git('show', f'HEAD:tools/gomad3/{name}.md')) for name in ['SPEC', 'README']},
    'original_language_entries': len(terms),
    'assessments': len(rows),
    'dispositions': dispositions,
    'current_concepts': len(current),
    'vocabulary_definitions': len(definitions),
    'vocabulary_aliases': aliases,
    'original_identifier_count': len(original_ids),
    'current_identifier_count': len(current_ids),
    'command_table_identifier_count': len(table_ids(spec)),
    'identifiers_preserved_in_order': original_ids == current_ids,
    'semantic_identifiers': current_ids,
    'boundaries': boundaries,
    'whitespace_argv': ['git', 'diff', '--check', BASE, '--', *paths],
    'whitespace_status': whitespace.returncode,
    'errors': errors,
}
(OUT / 'vocabulary-audit.json').write_text(json.dumps(report, indent=2) + '\n')
print(json.dumps({key: report[key] for key in [
    'baseline_revision', 'head_revision', 'working_tree_sha256', 'original_language_entries', 'assessments',
    'current_concepts', 'vocabulary_definitions', 'original_identifier_count', 'current_identifier_count',
    'command_table_identifier_count', 'identifiers_preserved_in_order', 'whitespace_status', 'errors']}, indent=2))
sys.exit(bool(errors))
