import collections
import hashlib
import json
import pathlib
import re
import subprocess
import sys
import urllib.parse

ROOT = pathlib.Path(__file__).resolve().parents[3]
OUT = pathlib.Path(sys.argv[2]).resolve() if len(sys.argv) > 2 else ROOT / '.flow/artifacts/fn-111-gomad-consolidate-vocabulary-and-update'
BASE = '29917069e089dc0739ec091b18e99161245b9bd5'
GUIDES = ['SPEC', 'ARCHITECTURE', 'CLI', 'TUTORIAL', 'README']
BIN = pathlib.Path(sys.argv[1]).resolve()
OUT.mkdir(parents=True, exist_ok=True)

def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT, text=True)

def unfenced(body):
    lines, marker = [], None
    for line in body.splitlines():
        fence = re.match(r'^\s{0,3}(`{3,}|~{3,})', line)
        if fence:
            token = fence.group(1)
            if marker is None:
                marker = token
            elif token[0] == marker[0] and len(token) >= len(marker):
                marker = None
            continue
        if marker is None:
            lines.append(line)
    return '\n'.join(lines), marker

def anchors(path):
    body, _ = unfenced(path.read_text())
    seen, result = collections.Counter(), set()
    for title in re.findall(r'^#{1,6}\s+(.+?)\s*#*\s*$', body, re.M):
        title = re.sub(r'\[([^]]+)\]\([^)]*\)', r'\1', title)
        slug = re.sub(r'[^\w\- ]', '', title.lower()).replace(' ', '-')
        index = seen[slug]
        seen[slug] += 1
        result.add(slug + (f'-{index}' if index else ''))
    result.update(re.findall(r'<a\s+(?:id|name)=["\']([^"\']+)', body))
    return result

docs = {name: (ROOT / f'tools/gomad3/{name}.md').read_text() for name in GUIDES}
old_spec = git('show', f'{BASE}:tools/gomad3/SPEC.md')
old_glossary = git('show', f'{BASE}:tools/gomad3/GLOSSARY.md')
if len(sys.argv) <= 2:
    (OUT / 'baseline-SPEC.md').write_text(old_spec)
    (OUT / 'baseline-GLOSSARY.md').write_text(old_glossary)
ids = lambda text: re.findall(r'\[([A-Z][A-Z0-9]*(?:\.[A-Z0-9]+)*)\]', text)
original, current = ids(old_spec), ids(docs['SPEC'])
language = old_glossary.split('## Language', 1)[1].split('\n## ', 1)[0]
terms = re.findall(r'^\*\*([^*]+)\*\*', language, re.M)
vocabulary = docs['SPEC'].split('## [PRODUCT.VOCABULARY]', 1)[1].split('## [PLATFORM]', 1)[0]
defined = set(re.findall(r'\*\*([^*]+)\*\*', vocabulary))
missing_terms = [term for term in terms if term != 'Parity Case' and term not in defined]
navigation, errors = [], []
for name, body in docs.items():
    path = ROOT / f'tools/gomad3/{name}.md'
    prose, unclosed = unfenced(body)
    if unclosed:
        errors.append({'file': str(path.relative_to(ROOT)), 'error': 'unclosed code fence'})
    for label, destination in re.findall(r'\[([^]]+)\]\(([^)\n]+)\)', prose):
        if re.match(r'^[a-zA-Z][a-zA-Z0-9+.-]*:', destination):
            continue
        if '<' in destination or '>' in destination:
            continue
        destination = destination.split(' "', 1)[0]
        target, _, fragment = destination.partition('#')
        target_path = (path.parent / urllib.parse.unquote(target)).resolve() if target else path
        item = {'file': str(path.relative_to(ROOT)), 'destination': destination, 'exists': target_path.exists()}
        if not item['exists']:
            errors.append(item)
        elif fragment and target_path.suffix == '.md' and urllib.parse.unquote(fragment) not in anchors(target_path):
            item['fragment_resolves'] = False
            errors.append(item)
        navigation.append(item)
    for stale in ['GLOSSARY.md', 'Combined ChoiceExploration']:
        if stale in body:
            errors.append({'file': str(path.relative_to(ROOT)), 'error': 'active stale term', 'term': stale})

index = docs['CLI'].split('## Command index', 1)[1]
commands = {}
for tool in ['gomad', 'gomadtool']:
    block = index.split(f'### `{tool}`', 1)[1].split('\n### ', 1)[0]
    commands[tool] = re.findall(r'^\| `([^`]+)` \|', block, re.M)
help_entries, registered_flags = [], set()
for tool, names in commands.items():
    for name in names:
        variants = ['discover', 'review', 'generate', 'check', 'qualify'] if name == 'compatibility-pack' else [None]
        for action in variants:
            argv = [str(BIN / tool), name] + ([action] if action else []) + ['-h']
            result = subprocess.run(argv, capture_output=True, text=True, timeout=15)
            output = result.stdout + result.stderr
            key = f'{tool}-{name}' + (f'-{action}' if action else '')
            (OUT / f'{key}.help.txt').write_text(output)
            found = re.findall(r'^\s+-([a-z][a-z0-9-]*)(?:\s|$)', output, re.M)
            registered_flags.update(found)
            help_entries.append({'argv': argv, 'status': result.returncode, 'flags': found, 'output_file': f'{key}.help.txt'})
            positional_usage = name == 'checked-run' and output.startswith('usage: gomadtool checked-run <seconds>') and result.returncode == 125
            if 'unknown ' + tool + ' command' in output or not found and not positional_usage:
                errors.append({'command': key, 'error': 'command help unavailable', 'status': result.returncode})
documented_flags = sorted(set(re.findall(r'--([a-z][a-z0-9-]*)', docs['CLI'])))
target_parser = (ROOT / 'tools/gomad3/cmd/gomad/internal/cli/cli.go').read_text()
assert 'arguments[1] != "--provenance"' in target_parser
registered_flags.add('provenance')
unknown_flags = sorted(set(documented_flags) - registered_flags)
paths = [f'tools/gomad3/{name}.md' for name in GUIDES]
whitespace = subprocess.run(['git', 'diff', '--check', BASE, '--', *paths], cwd=ROOT, capture_output=True, text=True)
if original != current:
    errors.append({'error': 'semantic identifier order/set changed'})
if missing_terms or len(terms) != 25:
    errors.append({'error': 'original glossary term loss', 'missing': missing_terms, 'term_count': len(terms)})
if (ROOT / 'tools/gomad3/GLOSSARY.md').exists():
    errors.append({'error': 'duplicate glossary owner remains'})
if unknown_flags:
    errors.append({'error': 'documented flag absent from actual CLI help', 'flags': unknown_flags})
if whitespace.returncode:
    errors.append({'error': 'whitespace check', 'output': whitespace.stdout + whitespace.stderr})
report = {'baseline_revision': BASE, 'head_revision': git('rev-parse', 'HEAD').strip(),
          'guide_sha256': {name: 'sha256:' + hashlib.sha256(text.encode()).hexdigest() for name, text in docs.items()},
          'original_semantic_identifiers': original, 'current_semantic_identifiers': current,
          'original_identifier_count': len(original), 'current_identifier_count': len(current),
          'identifiers_preserved_in_order': original == current,
          'original_glossary_terms': terms, 'current_terms_missing': missing_terms,
          'navigation': navigation, 'commands': commands, 'actual_help': help_entries,
          'documented_flags': documented_flags, 'unknown_documented_flags': unknown_flags,
          'whitespace_argv': ['git', 'diff', '--check', BASE, '--', *paths],
          'whitespace_status': whitespace.returncode, 'errors': errors}
(OUT / 'documentation-audit.json').write_text(json.dumps(report, indent=2) + '\n')
print(json.dumps({'identifier_count': len(current), 'original_glossary_terms': len(terms),
                  'commands': sum(map(len, commands.values())), 'documented_flags': len(documented_flags),
                  'links_checked': len(navigation), 'errors': errors}, indent=2))
sys.exit(bool(errors))
