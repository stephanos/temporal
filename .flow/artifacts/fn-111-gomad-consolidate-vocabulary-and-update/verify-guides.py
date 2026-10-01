"""Guide reconciliation audit for fn-111 task 2 (R4-R8).

Usage: verify-guides.py BIN_DIR FLOWCTL

BIN_DIR holds `gomad` and `gomadtool` built from the audited tree; FLOWCTL is
the flowctl executable. The audit writes guide-audit.json beside this file and
exits nonzero on any error.

Each command example in the five guides is checked against its own command:
flag registration and placement from the binary's -h output, then the value
rules in validate_value and the cross-flag rules in check_invocation, each of
which restates one parser or Runner validation named beside it. A flag whose
value has no rule there (a path, a tag, a probe name) is checked for presence
only. Uppercase metavariables such as N or DIR are accepted in prose spans and
rejected in fenced examples for numeric, size, and duration flags.

The claim matrix pairs a guide sentence with the source line implementing it.
The corpus section compares guide and milestone statements with the manifests.
The clock probe runs on the pinned toolchain. Every file the audit reads is
hashed and its difference from HEAD recorded, because the tree is uncommitted.
"""

import collections
import hashlib
import json
import pathlib
import re
import shlex
import shutil
import subprocess
import sys
import tempfile
import urllib.parse

ROOT = pathlib.Path(__file__).resolve().parents[3]
OUT = pathlib.Path(__file__).resolve().parent
BASE = '29917069e089dc0739ec091b18e99161245b9bd5'
GOMAD3 = ROOT / 'tools/gomad3'
GUIDES = ['SPEC', 'ARCHITECTURE', 'CLI', 'TUTORIAL', 'README']
MILESTONES = '.plans/GOMAD_MILESTONES.md'
PATHS = [f'tools/gomad3/{name}.md' for name in GUIDES] + [MILESTONES]
BIN = pathlib.Path(sys.argv[1]).resolve()
FLOWCTL = pathlib.Path(sys.argv[2]).resolve()
inputs = set()
TARGET_KINDS = {'go-run', 'go-test', 'exec'}
TARGET_COMMANDS = {'explore', 'plan', 'qualify', 'analyze'}
PACK_ACTIONS = ['discover', 'review', 'generate', 'check', 'qualify']
errors = []


def git(*args, check=True):
    result = subprocess.run(['git', *args], cwd=ROOT, capture_output=True, text=True)
    if check and result.returncode:
        raise SystemExit(f'git {args}: {result.stderr}')
    return result


def read(path):
    inputs.add(str(path))
    return (ROOT / path).read_text()


def squash(text):
    return re.sub(r'\s+', ' ', text)


# --- fences -----------------------------------------------------------------

def scan_fences(path):
    """Return prose lines, fenced blocks, and fence defects for one file."""
    prose, blocks, defects, current = [], [], [], None
    for number, line in enumerate(read(path).splitlines(), 1):
        fence = re.match(r'^(\s{0,3})(`{3,}|~{3,})(.*)$', line)
        if fence and current is None:
            info = fence.group(3).strip()
            if not info:
                defects.append({'file': path, 'line': number, 'error': 'fence has no type'})
            current = {'file': path, 'line': number, 'marker': fence.group(2), 'type': info, 'lines': []}
            continue
        if fence and current is not None and fence.group(2)[0] == current['marker'][0] and not fence.group(3).strip():
            if len(fence.group(2)) != len(current['marker']):
                defects.append({'file': path, 'line': number, 'error': 'closing fence length differs from opening'})
            if len(fence.group(2)) >= len(current['marker']):
                blocks.append(current)
                current = None
                continue
        (prose if current is None else current['lines']).append(line)
    if current is not None:
        defects.append({'file': path, 'line': current['line'], 'error': 'unclosed code fence'})
    return prose, blocks, defects


scanned = {path: scan_fences(path) for path in PATHS}
fence_report = {}
for path, (_, blocks, defects) in scanned.items():
    errors.extend(defects)
    fence_report[path] = dict(collections.Counter(block['type'] for block in blocks))


# --- links and fragments ----------------------------------------------------

def anchors(path):
    prose, _, _ = scanned.get(path) or scan_fences(path)
    seen, result = collections.Counter(), set()
    for title in re.findall(r'^#{1,6}\s+(.+?)\s*#*\s*$', '\n'.join(prose), re.M):
        title = re.sub(r'\[([^]]+)\]\([^)]*\)', r'\1', title)
        slug = re.sub(r'[^\w\- ]', '', title.lower()).replace(' ', '-')
        index = seen[slug]
        seen[slug] += 1
        result.add(slug + (f'-{index}' if index else ''))
    return result


navigation = []
for path in PATHS:
    text = '\n'.join(scanned[path][0])
    for destination in re.findall(r'\[[^]]+\]\(([^)\s]+)\)', squash(text)):
        if re.match(r'^[a-zA-Z][a-zA-Z0-9+.-]*:', destination):
            continue
        target, _, fragment = destination.partition('#')
        resolved = ((ROOT / path).parent / urllib.parse.unquote(target)).resolve() if target else ROOT / path
        item = {'file': path, 'destination': destination, 'exists': resolved.exists()}
        if not item['exists']:
            errors.append({**item, 'error': 'link target missing'})
        elif fragment and resolved.suffix == '.md':
            item['fragment_resolves'] = urllib.parse.unquote(fragment) in anchors(str(resolved.relative_to(ROOT)))
            if not item['fragment_resolves']:
                errors.append({**item, 'error': 'fragment missing'})
        navigation.append(item)

stale = []
for name in GUIDES:
    body = read(f'tools/gomad3/{name}.md')
    for term in ['GLOSSARY.md', 'Combined ChoiceExploration', 'World Model Model', 'prepared Target']:
        if term in body:
            stale.append({'file': f'tools/gomad3/{name}.md', 'term': term})
errors.extend({**item, 'error': 'stale or malformed vocabulary'} for item in stale)
if (GOMAD3 / 'GLOSSARY.md').exists():
    errors.append({'error': 'deleted glossary still present'})


# --- actual command help ----------------------------------------------------

def command_help(tool, name, action=None):
    argv = [str(BIN / tool), name] + ([action] if action else []) + ['-h']
    result = subprocess.run(argv, capture_output=True, text=True, timeout=30)
    output = result.stdout + result.stderr
    flags = {}
    lines = output.splitlines()
    for index, line in enumerate(lines):
        match = re.match(r'^\s+-([a-z_][a-z0-9_-]*)(?:\s+(\S+))?\s*$', line)
        if match:
            description = lines[index + 1].strip() if index + 1 < len(lines) else ''
            flags[match.group(1)] = {'value': match.group(2) is not None, 'type': match.group(2), 'description': description}
    return {'argv': [tool, name] + ([action] if action else []) + ['-h'], 'status': result.returncode,
            'sha256': hashlib.sha256(output.encode()).hexdigest(), 'flags': flags, 'output': output}


index = read('tools/gomad3/CLI.md').split('## Command index', 1)[1]
commands = {tool: re.findall(r'^\| `([^`]+)` \|', index.split(f'### `{tool}`', 1)[1].split('\n### ', 1)[0], re.M)
            for tool in ['gomad', 'gomadtool']}
helps = {}
for tool, names in commands.items():
    for name in names:
        for action in (PACK_ACTIONS if name == 'compatibility-pack' else [None]):
            key = f'{tool} {name}' + (f' {action}' if action else '')
            helps[key] = command_help(tool, name, action)
            positional = name == 'checked-run' and helps[key]['output'].startswith('usage: gomadtool checked-run <seconds>')
            if f'unknown {tool} command' in helps[key]['output'] or not helps[key]['flags'] and not positional:
                errors.append({'command': key, 'error': 'command help unavailable'})
            retained = OUT / (key.replace(' ', '-') + '.help.txt')
            if retained.exists() and read(retained.relative_to(ROOT)) != helps[key]['output']:
                errors.append({'command': key, 'error': 'retained help text differs from the actual binary'})

usage = read('tools/gomad3/cmd/gomad/internal/cli/cli.go') + read('tools/gomad3/cmd/gomadtool/main.go')
for tool, names in commands.items():
    for name in names:
        if not re.search(r'case "' + re.escape(name) + '"', usage):
            errors.append({'command': f'{tool} {name}', 'error': 'indexed command has no dispatch case'})
dispatched = set(re.findall(r'case "([a-z][a-z-]*)"(?:, "[^"]*")*:\n\t\treturn run', usage))
undocumented = sorted(dispatched - set(commands['gomad']) - set(commands['gomadtool']))
if undocumented:
    errors.append({'error': 'dispatched public command missing from the command index', 'commands': undocumented})
test_modes = set(re.findall(r'^\t"(test[a-z-]*)": \{', read('tools/gomad3/internal/gomadtool/conformance/registry.go'), re.M))


# --- command examples -------------------------------------------------------

BYTE_SIZE = re.compile(r'^[1-9][0-9]*(KiB|MiB|GiB)?$')
UNITS = {'KiB': 1 << 10, 'MiB': 1 << 20, 'GiB': 1 << 30, None: 1}
DURATION = re.compile(r'^(\d+)(ns|us|ms|s|m|h)$')
SECONDS = {'ns': 1e-9, 'us': 1e-6, 'ms': 1e-3, 's': 1, 'm': 60, 'h': 3600}


def constants(path, type_name):
    """Return the string constants a Go source file declares for one type."""
    return set(re.findall(r'^\t\w+\s+' + type_name + r' = "([^"]+)"', read(path), re.M))


# The closed vocabulary each parser accepts, read from the constants and comparisons it switches on.
ENUMERATED = {
    'strategy': constants('tools/gomad3/runner/runner.go', 'Strategy'),
    'coverage': constants('tools/gomad3/runner/runner.go', 'CoverageMode'),
    'keep-successes': constants('tools/gomad3/runner/runner.go', 'KeepSuccesses'),
    'on-failure': constants('tools/gomad3/runner/runner.go', 'FailurePolicy'),
    'capability-mode': constants('tools/gomad3/target/target.go', 'CapabilityMode'),
    'clock-tick': set(re.findall(r'^\tClockTick(?:Strict|Forward)\s+= "([^"]+)"', read('tools/gomad3/record/validation.go'), re.M)),
    'format': set(re.findall(r'\*format != "([a-z]+)"', read('tools/gomad3/cmd/gomad/internal/cli/analyze.go'))),
}
EXPECTED_ENUMERATIONS = {'strategy': 3, 'coverage': 4, 'keep-successes': 3, 'on-failure': 3, 'capability-mode': 3, 'clock-tick': 2, 'format': 2}
for flag, size in EXPECTED_ENUMERATIONS.items():
    if len(ENUMERATED[flag]) != size:
        errors.append({'error': 'parser enumeration could not be read from source', 'flag': flag, 'values': sorted(ENUMERATED[flag])})
for path in ['compare_support.go', 'qualify_set.go']:
    if set(re.findall(r'\*format != "([a-z]+)"', read('tools/gomad3/cmd/gomad/internal/cli/' + path))) != ENUMERATED['format']:
        errors.append({'error': 'format vocabulary differs between commands', 'file': path})
BYTE_FLAGS = {'choice-bytes', 'io-transcript-bytes', 'success-bytes', 'output-limit', 'max-exploration-bytes',
              'max-exploration-result-bytes', 'world-transition-limit', 'min-free-bytes', 'max-bytes'}
SIMULATION_BOUNDS = ['max-executions', 'max-forced-decisions', 'max-exploration-bytes', 'max-exploration-result-bytes',
                     'max-runtime-decisions', 'max-scenario-decisions', 'max-network-decisions', 'max-storage-decisions',
                     'max-fault-decisions', 'max-crash-decisions']
COMBINED_ONLY = set(SIMULATION_BOUNDS) - {'max-executions', 'max-exploration-bytes'}
wire = read('tools/gomad3/choice/internal/wire/wire_generated.go')
MINIMUM_CHOICE_BYTES = sum(int(re.search(name + r'\s+= (\d+)', wire).group(1)) for name in ['HeaderBytes', 'RecordBytes'])


def byte_value(text):
    match = BYTE_SIZE.match(text)
    return int(re.match(r'\d+', text).group()) * UNITS[match.group(1)] if match else None


def seed_count(text):
    """Count the seeds runner.ParseSeeds selects, or None when it rejects the text."""
    total = 0
    for term in text.split(','):
        match = re.match(r'^(\d+)(?:-(\d+))?$', term)
        if not match or match.group(2) is not None and int(match.group(2)) < int(match.group(1)):
            return None
        total += 1 if match.group(2) is None else int(match.group(2)) - int(match.group(1)) + 1
    return total


def validate_value(name, flag, value, entry, fenced):
    """Return why this flag value is rejected by the implementation, or None."""
    kind = entry['type']
    if not entry['value']:
        return None if value in ('', 'true', 'false') else 'boolean flag given a value'
    if value == '':
        return 'empty value'
    if flag in ENUMERATED:
        if any(literal not in entry['description'] for literal in ENUMERATED[flag]):
            return 'usage text omits a value the parser accepts'
        return None if value in ENUMERATED[flag] else 'value outside the parser enumeration'
    if flag == 'mode':  # conformance.Resolve
        return None if value in test_modes else 'unknown conformance mode'
    numeric = kind in ('uint', 'int', 'duration') or flag in BYTE_FLAGS
    if numeric and re.match(r'^[A-Z][A-Z_]*$', value):
        return 'metavariable in a fenced example' if fenced else None
    if flag in BYTE_FLAGS:  # cli.byteSize.Set: positive, no leading zero, KiB/MiB/GiB
        size = byte_value(value)
        if size is None:
            return 'malformed byte size'
        if flag == 'choice-bytes' and not MINIMUM_CHOICE_BYTES <= size <= 64 << 20:  # cli.resolveChoiceTrace
            return 'choice trace capacity outside the implemented bound'
        if flag == 'io-transcript-bytes' and (size < 64 << 20 or size > 1 << 30 or size % (1 << 20)):  # deterministicio.ValidateTranscriptLimit
            return 'I/O transcript capacity outside the implemented bound'
        return None
    if kind in ('uint', 'int'):
        if not value.isdigit():
            return 'not an unsigned integer'
        number = int(value)
        if flag == 'repeat' and not 2 <= number <= 32:  # cli maximumQualificationRepeats
            return 'repeat outside 2 through 32'
        # zero is rejected by resolveExploreStrategy, resolveExploreSeeds, runMinimizeWith, and runner.validateConfig
        if number == 0 and (flag.startswith('max-') or flag in {'attempt-budget', 'count', 'failure-budget', 'success-limit', 'parallel'}):
            return 'zero is rejected by the parser'
        return None
    if kind == 'duration':
        match = DURATION.match(value)
        if not match or int(match.group(1)) == 0:  # runner.validateConfig requires positive deadlines
            return 'not a positive Go duration'
        if name == 'analyze' and int(match.group(1)) * SECONDS[match.group(2)] > 1800:  # resolveCapabilityAnalysisTimeout
            return 'analysis timeout above 30 minutes'
        return None
    if flag == 'seeds':  # runner.ParseSeeds
        return None if seed_count(value) else 'seed selection the parser rejects'
    if flag == 'shard':  # cli.parseShardAssignment and Shard.Validate
        match = re.match(r'^(\d+)/(\d+)$', value)
        if match:
            return None if int(match.group(1)) < int(match.group(2)) else 'shard index not below its count'
        return None if value in ('INDEX/COUNT', '$shard') else 'malformed shard'
    if flag in {'approve-boundary-diff', 'approve-review'}:  # record.ParseSHA256
        return None if re.match(r'^(sha256:([0-9a-f]{64}|[A-Z_]+)|SHA256|<[a-z0-9-]+>|LT[a-z0-9-]+GT)$', value) else 'not a sha256: digest'
    if flag in {'io-ro-mount', 'env'}:  # readonlymount.ParseMappings, runner.parseEnvironment
        return None if re.match(r'^[^=]+=.+$', value) else 'needs NAME=VALUE'
    return None


def check_invocation(tool, tokens, origin, fenced):
    """Validate one documented invocation against its command's actual flags."""
    if not tokens:
        return None
    name, rest = tokens[0], tokens[1:]
    if name not in commands[tool]:
        return {'origin': origin, 'error': f'unknown {tool} command', 'command': name}
    key = f'{tool} {name}'
    if name == 'compatibility-pack':
        if not rest or rest[0] not in PACK_ACTIONS:
            return {'origin': origin, 'error': 'compatibility-pack needs an action', 'tokens': tokens}
        key, rest = f'{key} {rest[0]}', rest[1:]
    if name == 'checked-run':
        if len(rest) < 6 or rest[4] != '--' or not rest[0].isdigit() or not rest[1].isdigit():
            return {'origin': origin, 'error': 'checked-run needs SECONDS STATUS LABEL DIR -- COMMAND', 'tokens': tokens}
        return {'origin': origin, 'command': key, 'flags': [], 'ok': True}
    flags, used, position, positional, target = helps[key]['flags'], {}, 0, 0, None
    while position < len(rest):
        token = rest[position]
        position += 1
        if name in TARGET_COMMANDS and token in TARGET_KINDS:
            target = token
            break
        if token == '--' or not token.startswith('-'):
            positional += 1
            continue
        if positional:
            return {'origin': origin, 'error': 'flag after a positional argument is not parsed', 'flag': token, 'command': key}
        flag, _, value = token.lstrip('-').partition('=')
        if flag not in flags:
            return {'origin': origin, 'error': 'flag not registered for this command', 'flag': flag, 'command': key}
        if flags[flag]['value'] and '=' not in token:
            if position >= len(rest):
                return {'origin': origin, 'error': 'flag needs a value', 'flag': flag, 'command': key}
            value = rest[position]
            position += 1
        used.setdefault(flag, []).append(value)
        rejected = validate_value(name, flag, value, flags[flag], fenced)
        if rejected:
            return {'origin': origin, 'error': rejected, 'flag': flag, 'value': value, 'command': key}
    if name in TARGET_COMMANDS:
        if positional:
            return {'origin': origin, 'error': 'positional argument before the target kind', 'tokens': tokens}
        if target is None and fenced:
            return {'origin': origin, 'error': 'example has no target kind', 'tokens': tokens}
        if target == 'exec' and (name == 'analyze' or rest[position:position + 1] != ['--provenance'] or rest[position + 2:position + 3] != ['--']):
            return {'origin': origin, 'error': 'exec needs --provenance FILE -- BINARY and is not analyzable', 'tokens': tokens}
        if target in {'go-run', 'go-test'} and len(rest) > position + 1 and rest[position + 1] != '--':
            return {'origin': origin, 'error': 'target arguments need the -- separator', 'tokens': tokens}

    def last(flag, default=None):
        return used[flag][-1] if flag in used else default

    strategy, coverage = last('strategy', 'seed'), last('coverage', 'semantic' if 'guide' in used else 'none')
    exploring = strategy != 'seed'
    selected = seed_count(last('seeds', '1')) if not re.match(r'^[A-Z]', last('seeds', '1')) else 1
    explore = name in {'explore', 'plan'}
    rules = {  # each restates one validation in cli.go, qualify.go, or runner.validateConfig
        'choice-bytes requires choices': 'choice-bytes' in used and 'choices' not in used and not exploring,
        'count excludes seeds': 'count' in used and 'seeds' in used,
        'corpus requires guide and guide requires corpus': ('corpus' in used) != ('guide' in used),
        'guide requires semantic or choice coverage': 'guide' in used and coverage == 'none',
        'choice coverage requires choices': explore and coverage in {'choice', 'semantic+choice'} and 'choices' not in used and not exploring,
        'require-probe requires semantic coverage': explore and 'require-probe' in used and coverage not in {'semantic', 'semantic+choice'},
        'novel retention requires coverage': explore and last('keep-successes') == 'novel' and coverage == 'none',
        'retention needs both bounds': explore and last('keep-successes', 'none') != 'none' and not {'success-limit', 'success-bytes'} <= set(used),
        'retention bounds need a retention policy': explore and last('keep-successes', 'none') == 'none' and bool({'success-limit', 'success-bytes'} & set(used)),
        'replay-successes needs both bounds': 'replay-successes' in used and not {'success-limit', 'success-bytes'} <= set(used),
        'qualify retention bounds need replay-successes': name == 'qualify' and bool({'success-limit', 'success-bytes'} & set(used)) and 'replay-successes' not in used,
        'failure budget only with on-failure=budget': 'failure-budget' in used and last('on-failure') != 'budget',
        'seed strategy rejects exploration bounds': explore and not exploring and bool(({'max-executions', 'max-choice-depth', 'max-exploration-bytes'} | COMBINED_ONLY) & set(used)),
        'exploration needs exactly one base seed': explore and exploring and selected != 1,
        'exploration rejects count and guide': explore and exploring and bool({'count', 'guide'} & set(used)),
        'choice exploration needs its three bounds': strategy == 'choice-exploration' and not {'max-executions', 'max-choice-depth', 'max-exploration-bytes'} <= set(used),
        'choice exploration rejects combined bounds': strategy == 'choice-exploration' and bool(COMBINED_ONLY & set(used)),
        'combined exploration needs all ten bounds': strategy == 'simulation-exploration' and not set(SIMULATION_BOUNDS) <= set(used),
        'combined exploration rejects max-choice-depth': strategy == 'simulation-exploration' and 'max-choice-depth' in used,
        'plan needs output': name == 'plan' and fenced and 'output' not in used,
        'output only with plan': name == 'explore' and 'output' in used,
        'plan is an unguided seed campaign': name == 'plan' and (exploring or 'guide' in used or last('on-failure', 'all') != 'all'),
        'execute-shard needs shard': name == 'execute-shard' and 'shard' not in used,
        'merge needs output': name == 'merge' and fenced and 'output' not in used,
        'qualify-set needs manifest and working-dir': name == 'qualify-set' and fenced and not {'manifest', 'working-dir'} <= set(used),
        'merge-set needs manifest and a shard report': name == 'merge-set' and fenced and ('manifest' not in used or positional == 0),
        'compare-support needs baseline and candidate': name == 'compare-support' and fenced and not {'baseline', 'candidate'} <= set(used),
        'one positional artifact or campaign': name in {'inspect', 'replay', 'minimize', 'recover', 'resume', 'execute-shard'} and fenced and positional != 1,
        'doctor takes no positional': name == 'doctor' and positional != 0,
    }
    broken = [rule for rule, violated in rules.items() if violated]
    if broken:
        return {'origin': origin, 'error': 'example violates a parser rule', 'rules': broken, 'tokens': tokens}
    used = sorted(used)
    return {'origin': origin, 'command': key, 'flags': used, 'target': target, 'ok': True}


PREFIXES = [('gomadtool', r'go -C tools/gomad3 run \./cmd/gomadtool '), ('gomadtool', r'gomadtool '),
            ('gomad', r'tools/gomad3/\.bin/gomad '), ('gomad', r'gomad ')]
makefiles = {'.': read('Makefile'), 'tools/gomad3': read('tools/gomad3/Makefile')}
examples, make_examples = [], []


def split_tokens(text):
    try:
        return shlex.split(text.replace('<', 'LT').replace('>', 'GT').replace('|', 'OR'))
    except ValueError:
        return None


def check_make(text, origin):
    tokens = split_tokens(text)
    directory, targets, position = '.', [], tokens.index('make') + 1
    while position < len(tokens):
        token = tokens[position]
        position += 1
        if token == '-C':
            directory = tokens[position]
            position += 1
        elif '=' not in token and not token.startswith('-'):
            targets.append(token)
    for target in targets:
        found = directory in makefiles and re.search(r'^' + re.escape(target) + r':', makefiles[directory], re.M)
        make_examples.append({'origin': origin, 'directory': directory, 'target': target, 'ok': bool(found)})
        if not found:
            errors.append({'origin': origin, 'error': 'make target not defined', 'target': target, 'directory': directory})


def check_text(text, origin, fenced):
    text = text.strip().rstrip(';').strip()
    if re.match(r'^([A-Z0-9_]+=\S+ )*make ', text):
        check_make(text, origin)
        return
    for tool, prefix in PREFIXES:
        match = re.match(r'^' + prefix + r'(.*)$', text)
        if not match:
            continue
        if match.group(1).split()[0] in {'CLI', 'COMMAND'}:
            return
        tokens = split_tokens(match.group(1))
        result = check_invocation(tool, tokens, origin, fenced) if tokens is not None else {'origin': origin, 'error': 'unparseable example'}
        examples.append(result)
        if 'error' in result:
            errors.append(result)
        return
    if fenced:
        return
    first = text.split()[0] if text.split() else ''
    for tool in ['gomad', 'gomadtool']:
        if first in commands[tool] and re.search(r'(^| )--?[a-z]', text) and first not in {'test'}:
            result = check_invocation(tool, split_tokens(text), origin, False)
            examples.append(result)
            if 'error' in result:
                errors.append(result)
            return


for name in GUIDES:
    path = f'tools/gomad3/{name}.md'
    prose, blocks, _ = scanned[path]
    for block in blocks:
        if block['type'] != 'sh':
            continue
        logical = re.sub(r'\\\n\s*', ' ', '\n'.join(block['lines']))
        for offset, line in enumerate(logical.splitlines()):
            check_text(line, f'{path}:{block["line"]}+{offset}', True)
    for span in re.findall(r'`([^`]+)`', squash('\n'.join(prose))):
        check_text(span, f'{path}:inline `{span[:60]}`', False)

documented = set()
for name in GUIDES:
    documented.update(re.findall(r'(?<![\w-])--([a-z][a-z0-9-]*)', read(f'tools/gomad3/{name}.md')))
registered = {flag for entry in helps.values() for flag in entry['flags']} | {'provenance'}
unknown_flags = sorted(documented - registered)
if unknown_flags:
    errors.append({'error': 'documented flag is registered by no command', 'flags': unknown_flags})

# Prose that attributes a flag to named commands, checked per command.
ATTRIBUTED = {
    'clock-tick': ['explore', 'plan', 'qualify'], 'io-transcript-bytes': ['explore', 'qualify'],
    'choices': ['explore', 'qualify', 'inspect'], 'working-dir': ['explore', 'plan', 'qualify', 'analyze', 'qualify-set'],
    'io-ro-mount': ['explore', 'qualify'], 'env': ['explore', 'qualify'], 'build-tag': ['explore', 'qualify', 'analyze'],
    'json': ['explore', 'execute-shard', 'resume', 'doctor', 'inspect', 'recover', 'minimize', 'merge', 'qualify'],
    'format': ['analyze', 'qualify-set', 'merge-set', 'compare-support'], 'partial': ['merge'],
    'prune-qualified-artifacts': ['qualify-set'], 'min-free-bytes': ['qualify-set'], 'shard': ['qualify-set', 'execute-shard'],
    'verify-only': ['replay'], 'approve-boundary-diff': ['compare-support'], 'toolchain-root': ['doctor', 'explore', 'replay', 'resume'],
}
attribution = []
for flag, names in ATTRIBUTED.items():
    for name in names:
        ok = flag in helps[f'gomad {name}']['flags']
        attribution.append({'flag': flag, 'command': name, 'ok': ok})
        if not ok:
            errors.append({'error': 'guide attributes a flag to a command that lacks it', 'flag': flag, 'command': name})
for name in ['replay', 'analyze', 'compare-support', 'qualify-set', 'merge-set']:
    if 'json' in helps[f'gomad {name}']['flags']:
        errors.append({'error': 'command gained --json; the automation section must list it', 'command': name})


# --- claim matrix: guide statement against implementing source ---------------

CLI_GO = 'tools/gomad3/cmd/gomad/internal/cli/cli.go'
CLI_DIR = 'tools/gomad3/cmd/gomad/internal/cli/'
RUNTIME = 'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go'
PATCH = 'tools/gomad3/toolchain/runtime/go1.27.1.patch'
MATRIX = [
    ('R5', 'explore default seed', 'CLI', r'`explore` uses seed 1', CLI_GO, r'flags\.String\("seeds", "1"'),
    ('R5', 'explore default parallelism', 'CLI', r'smaller of the host CPU count and 8', CLI_GO, r'"parallel", min\(runtime\.NumCPU\(\), 8\)'),
    ('R5', 'per-execution deadline', 'CLI', r'30 seconds per execution and 10 minutes overall', CLI_GO, r'"execution-timeout", 30\*time\.Second'),
    ('R5', 'overall deadline', 'CLI', r'30 seconds per execution and 10 minutes overall', CLI_GO, r'"overall-timeout", 10\*time\.Minute'),
    ('R5', 'default failure policy', 'CLI', r'default failure policy stops after the first retained failure', CLI_GO, r'"on-failure", string\(runner\.PolicyFirst\)'),
    ('R5', 'count excludes seeds', 'CLI', r'cannot be combined with `--seeds`', CLI_GO, r'--count and --seeds are mutually exclusive'),
    ('R5', 'choice trace default', 'CLI', r'Choice recording defaults to 8 MiB', CLI_GO, r'choiceLimit := byteSize\(8 << 20\)'),
    ('R5', 'choice trace maximum', 'CLI', r'at most `--choice-bytes=64MiB`', CLI_GO, r'--choice-bytes must be between %d bytes and 64MiB'),
    ('R5', 'choice bytes need choices', 'README', r'is valid only with `--choices`', CLI_GO, r'--choice-bytes requires --choices'),
    ('R5', 'output retention default', 'CLI', r'Output retention defaults to 8 MiB per stream', CLI_GO, r'outputLimit := byteSize\(8 << 20\)'),
    ('R5', 'transcript default', 'CLI', r'transcript capacity defaults to 64 MiB', 'tools/gomad3/deterministicio/session.go', r'DefaultTranscriptBytes = 64 << 20'),
    ('R5', 'transcript maximum', 'CLI', r'from 64 MiB through 1 GiB in whole MiB increments', 'tools/gomad3/deterministicio/session.go', r'MaximumTranscriptBytes = 1 << 30'),
    ('R5', 'transcript granularity', 'CLI', r'whole MiB increments', 'tools/gomad3/deterministicio/session.go', r'limit%TranscriptBytesGranularity != 0'),
    ('R5', 'required probes need semantic coverage', 'CLI', r'with `--coverage=semantic` or `--coverage=semantic\+choice`', CLI_GO, r'case runner\.CoverageSemantic, runner\.CoverageSemanticChoice:'),
    ('R5', 'choice coverage needs choices', 'README', r'Choice coverage requires `--choices`', CLI_GO, r'--coverage=%s requires --choices'),
    ('R5', 'guided coverage', 'README', r'`--coverage=none` is rejected', CLI_GO, r'--guide requires semantic or choice coverage'),
    ('R5', 'choice exploration base seed', 'CLI', r'requires one base seed and explicit positive bounds', CLI_GO, r'--strategy=choice-exploration requires exactly one base seed'),
    ('R5', 'choice exploration rejects count', 'CLI', r'does not combine with `--count` or guided exploration', CLI_GO, r'--strategy=choice-exploration does not accept --count'),
    ('R5', 'choice exploration rejects guide', 'CLI', r'does not combine with `--count` or guided exploration', CLI_GO, r'--strategy=choice-exploration does not support --guide'),
    ('R5', 'choice exploration implies choices', 'CLI', r'It implies choice recording', CLI_GO, r'explicit positive --max-exploration-bytes"\)\s+\}\s+return strategy, true, nil'),
    ('R5', 'combined exploration bounds', 'CLI', r'Every dimension is explicit', CLI_GO, r'--strategy=simulation-exploration requires an explicit positive %s'),
    ('R5', 'novel retention needs coverage and bounds', 'README', r'requires semantic or choice coverage', 'tools/gomad3/runner/runner.go', r'novel success retention requires coverage and explicit count and byte limits'),
    ('R5', 'plan fixes the failure policy', 'CLI', r'fixes the failure policy to complete all planned work', CLI_GO, r'"--__plan", "--on-failure=all"'),
    ('R5', 'plan accepts unguided seed campaigns', 'CLI', r'accepts unguided seed Campaigns', 'tools/gomad3/runner/portable_plan.go', r'portable campaign plans require an unguided seed campaign with on-failure=all'),
    ('R5', 'exec provenance form', 'CLI', r'exec --provenance \./target\.provenance\.json -- \./target', CLI_GO, r'arguments\[1\] != "--provenance"'),
    ('R5', 'analyze target kinds', 'CLI', r'`analyze` accepts only `go-run` and `go-test`', CLI_DIR + 'analyze.go', r'gomad analyze requires a go-run or go-test target'),
    ('R5', 'analyze closure bound', 'CLI', r'30-second wall limit in closure mode', CLI_DIR + 'analyze.go', r'capabilityAnalysisTimeout = 30 \* time\.Second'),
    ('R5', 'analyze linked and guarded bound', 'CLI', r'two minutes in linked or guarded mode', CLI_DIR + 'analyze.go', r'mode == target\.CapabilityModeLinked \|\| mode == target\.CapabilityModeGuarded'),
    ('R5', 'analyze maximum bound', 'CLI', r'at most 30 minutes', CLI_DIR + 'analyze.go', r'maximumCapabilityAnalysisTimeout = 30 \* time\.Minute'),
    ('R5', 'capability modes', 'CLI', r'`--capability-mode=guarded` is the third mode', CLI_GO, r'case target\.CapabilityModeClosure, target\.CapabilityModeLinked, target\.CapabilityModeGuarded:'),
    ('R5', 'qualify defaults', 'CLI', r'default is seed 1 with two repetitions', CLI_DIR + 'qualify.go', r'flags\.Uint64\("seed", 1,.*\n.*flags\.Uint64\("repeat", 2,'),
    ('R5', 'qualify repeat bound', 'CLI', r'`--repeat` accepts 2 through 32', CLI_DIR + 'qualify.go', r'maximumQualificationRepeats = 32'),
    ('R5', 'qualify replay bounds', 'CLI', r'`--replay-successes` requires explicit count and byte bounds', CLI_DIR + 'qualify.go', r'--replay-successes requires explicit --success-limit and --success-bytes bounds'),
    ('R5', 'qualify coverage', 'CLI', r'`--choices` adds choice coverage', CLI_DIR + 'qualify.go', r'coverage = runner\.CoverageSemanticChoice'),
    ('R5', 'minimize default budget', 'TUTORIAL', r'gomad minimize --attempt-budget=64', CLI_GO, r'"attempt-budget", 64,'),
    ('R5', 'doctor unavailable status', 'CLI', r'unavailable installation returns status 1', CLI_GO, r'if !report\.Available \{\s+return 1'),
    ('R5', 'replay preflight status', 'CLI', r'Status 2 means the input or compatibility contract was invalid', CLI_GO, r'errors\.As\(err, &preflightError\) \{\s+fmt\.Fprintln\(stderr, err\)\s+return 2'),
    ('R5', 'watchdog replay stays diagnostic status 1', 'CLI', r'watchdog observation uses diagnostic replay and returns status 1', CLI_GO, r'result=watchdog_observation choice-replay=%s\\n", result\.ChoiceReplayStatus\)\s+return 1, err'),
    ('R5', 'retained success replay status 0', 'CLI', r'a matching retained success returns 0', CLI_GO, r'result=success choice-replay=%s\\n", result\.ChoiceReplayStatus\)\s+return 0, err'),
    ('R5', 'watchdog is not a target failure', 'CLI', r'distinguishes a target failure, watchdog observation, replay divergence, mixed failure, and success', CLI_DIR + 'explore_output.go', r'summary\.Watchdogs == summary\.Failures \{\s+return "watchdog_observation"'),
    ('R5', 'compare-support incomparable status', 'CLI', r'incomparable reports return 2', CLI_DIR + 'compare_support.go', r'supportcomparison\.Incomparable \{\s+return 2'),
    ('R5', 'compare-support review status', 'CLI', r'Regressions or changes that require review return 1', CLI_DIR + 'compare_support.go', r'result\.ReviewRequired \{\s+return 1'),
    ('R5', 'merge-set statuses', 'CLI', r'0 when every expectation matched, 1 when the merged report retains a mismatch, 2 for invalid shards, and 3 when the report cannot be written', CLI_DIR + 'qualify_set.go', r'0 when every merged\s+// expectation matched, 1 when the merged report retains a mismatch, 2 when the\s+// shards or manifest are invalid, and 3 when the report cannot be written'),
    ('R5', 'qualify statuses', 'README', r'Input was invalid or the unsupported boundary was retained', 'tools/gomad3/qualification/events.go', r'case "unsupported_target", "invalid_input":\s+return 2'),
    ('R5', 'recover statuses', 'README', r'Invalid or non-recoverable input returns status 2', CLI_DIR + 'recover.go', r'runner\.IsInvalidRecoveryError\(err\) \{\s+return 2'),
    ('R5', 'qualify-set free-space default', 'README', r'`--min-free-bytes` \(2 GiB by default\)', 'tools/gomad3/qualification/set/freespace.go', r'DefaultMinimumFreeBytes = 2 << 30'),
    ('R5', 'explore event schema', 'README', r'`gomad3\.explore-event/v3`', CLI_DIR + 'explore_output.go', r'exploreEventSchema = "gomad3\.explore-event/v3"'),
    ('R5', 'inspect report schema', 'README', r'`gomad3\.inspect/v5`', 'tools/gomad3/runner/inspect.go', r'reportSchema = "gomad3\.inspect/v5"'),
    ('R5', 'conformance modes', 'CLI', r'--mode=test-builder', 'tools/gomad3/internal/gomadtool/conformance/registry.go', r'"test-builder": \{'),
    ('R5', 'full conformance gate tiers', 'README', r'then the builder, live-capability, runtime, and upstream tiers in that order', 'tools/gomad3/internal/gomadtool/conformance/registry.go', r'Tiers:\s+\[\]string\{"test-builder", "test-live-capability", "test-runtime", "test-upstream"\}'),
    ('R5', 'full make gate order', 'README', r'the harness, toolchain, interception, host, overlay, and World tests', 'tools/gomad3/Makefile', r'^test: test-harness test-toolchain intercept-test test-host overlay-test world-test test-builder test-live-capability test-runtime test-upstream$'),
    ('R5', 'dossier gates', 'CLI', r'validation, patch/compiler, Runner, World, probe, builder, runtime, disabled-upstream, host-clock, and cached-build gates', 'tools/gomad3/cmd/gomadtool/upgrade.go', r'"manifest-validation".*\n.*"toolchain-and-compiler".*\n.*"host-world-and-probes".*\n.*"builder".*\n.*"runtime".*\n.*"disabled-upstream".*\n.*"host-clock-escape".*\n.*"cached-toolchain-build"'),
    ('R5', 'checked-run output bound', 'README', r'retains at most 1 MiB from each child output stream', 'tools/gomad3/cmd/gomadtool/main.go', r'OutputLimit: 1 << 20'),
    ('R4', 'activation instant', 'ARCHITECTURE', r'midnight UTC on 2000-01-01', RUNTIME, r'gomadInitialTime = 946684800000000000'),
    ('R4', 'forward draw range', 'ARCHITECTURE', r'cumulative 1–1024 nanosecond increment at each `time\.Now` read', RUNTIME, r'return int64\(1 \+ value&gomadClockTickMask\)'),
    ('R4', 'forward draw mask', 'README', r'by 1 to 1024 nanoseconds', RUNTIME, r'gomadClockTickMask = 1023'),
    ('R4', 'forward stream is separate from scheduling choices', 'ARCHITECTURE', r'using a separate seed-derived stream', RUNTIME, r'gomadClockTickState = seed \^ 0x6c62272e07bb0142'),
    ('R4', 'offset is cumulative and outside faketime', 'ARCHITECTURE', r'retain the idle-driven clock', RUNTIME, r'now := faketime\s+if gomadClockForward \{\s+gomadClockTickOffset \+= gomadClockTickDraw\(\)\s+now \+= gomadClockTickOffset'),
    ('R4', 'only the time.Now entry point ticks', 'ARCHITECTURE', r'For a reading that still carries its monotonic value, `time\.Since` and `time\.Until` read the idle-driven clock', PATCH, r'func time_runtimeNow\(\) \(sec int64, nsec int32, mono int64\) \{[^@]*\+\tif gomadEnabled \{\n\+\t\treturn gomadTimeNow\(\)'),
    ('R4', 'wall-only readings compare against ticked time.Now', 'ARCHITECTURE', r'A reading stripped of its monotonic value, as by `Round\(0\)`, serialization, or parsing, is compared against a fresh ticked `time\.Now` instead', RUNTIME, r'func gomadTimeNow\(\) \(sec int64, nsec int32, mono int64\)'),
    ('R4', 'synctest bubble keeps its clock', 'ARCHITECTURE', r'a `testing/synctest` bubble keeps its own clock', PATCH, r'return sec, nsec, 0\n \t\}\n\+\tif gomadEnabled \{'),
    ('R4', 'unknown tick policy stops before user code', 'README', r'any other value stops the process before user initialization', RUNTIME, r'runtime: invalid GOMAD3_CLOCK_TICK'),
    ('R4', 'tick policy is recorded identity', 'README', r'`GOMAD3_CLOCK_TICK=forward`', 'tools/gomad3/record/validation.go', r'ClockTickEnvironment = "GOMAD3_CLOCK_TICK"'),
    ('R4', 'strict is recorded as absence', 'README', r'`strict` is recorded as the entry\'s absence', 'tools/gomad3/runner/runner.go', r'case "", record\.ClockTickStrict:\n\tcase record\.ClockTickForward:\n\t\tenvironment = append'),
    ('R4', 'forward conformance check', 'CLI', r'cumulative seeded offset of 1 to 1024 nanoseconds per `time\.Now` read', 'tools/gomad3/internal/gomadtool/conformance/runtime_clock_tick.go', r'delta < 1 \|\| delta > 1024'),
    ('R4', 'single P and no async preemption', 'ARCHITECTURE', r'forces the initial `GOMAXPROCS` to one, disables asynchronous preemption and the system monitor', RUNTIME, r'debug\.asyncpreemptoff = 1\s+haveSysmon = false'),
    ('R4', 'decision tape holds branching decisions only', 'ARCHITECTURE', r'Decision Tape containing only branching decisions', 'tools/gomad3/choice/tape.go', r'exact choice tape contains a non-branching decision'),
    ('R4', 'exploration forces a finite prefix', 'ARCHITECTURE', r'Choice Exploration uses forced prefixes from one base Seed', 'tools/gomad3/choice/tape.go', r'choice prefix contains a non-branching decision'),
    ('R4', 'choice trace maximum', 'TUTORIAL', r'Trace storage defaults to 8 MiB and is limited to 64 MiB', 'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go', r'mappingBytes > 64<<20'),
    ('R4', 'trace overflow is a Runner failure', 'TUTORIAL', r'If the Choice Trace overflows, Gomad reports a Runner failure', 'tools/gomad3/runner/runner.go', r'record\.ArtifactRunnerFailure && outcome\.Reason == "choice_trace_overflow"'),
    ('R4', 'exploration strategies', 'ARCHITECTURE', r'Combined Exploration keeps runtime, scenario, network, storage, fault, and crash decisions in separate dimensions', 'tools/gomad3/runner/runner.go', r'StrategySimulationExploration Strategy = "simulation-exploration"'),
    ('R4', 'hard isolation needs the process backend', 'ARCHITECTURE', r'the process backend supports Model Fidelity or Hard Isolation', 'tools/gomad3sim/spec.go', r'case FidelityHardIsolation:\s+if spec\.Backend != BackendProcess'),
    ('R4', 'both backends exist', 'ARCHITECTURE', r'The in-process backend supports Model Fidelity', 'tools/gomad3sim/spec.go', r'case BackendInProcess, BackendProcess:'),
    ('R6', 'backend and fidelity are separate fields', 'TUTORIAL', r'Only the process Backend supplies \*\*Hard Isolation\*\*', 'tools/gomad3sim/spec.go', r'Backend\s+Backend\s+`json:"backend"`\s+Fidelity\s+Fidelity\s+`json:"fidelity"`'),
    ('R4', 'Runner drains target output', 'ARCHITECTURE', r'Runner drains stdout and stderr concurrently, hashes every byte, and retains a bounded head and tail', 'tools/gomad3/runner/internal/execution/process_unix.go', r'io\.Copy\(writer, stdoutRead\)'),
    ('R6', 'supervisor passes output through', 'TUTORIAL', r'The Runner drains stdout and stderr, continues reading beyond the storage bound, and hashes the complete streams', 'tools/gomad3/runner/internal/execution/supervisor_unix.go', r'target\.Stdout = stdout\s+target\.Stderr = stderr'),
    ('R4', 'bounded head and tail with a full hash', 'ARCHITECTURE', r'retains a bounded head and tail', 'tools/gomad3/internal/hostexec/output.go', r'tailLimit := int\(limit / 4\)\s+headLimit := int\(limit\) - tailLimit'),
    ('R4', 'guided selection reserves a quarter', 'ARCHITECTURE', r'at least `ceil\(count/4\)` requested seeds unguided', 'tools/gomad3/runner/seeds.go', r'unguided := base\.count / 4'),
    ('R4', 'corpus limits', 'ARCHITECTURE', r'fixed limits of 1,024 entries and 1 GiB', 'tools/gomad3/runner/internal/corpus/model.go', r'maximumEntries = 1024\s+maximumBytes\s+= 1 << 30'),
    ('R4', 'launch ownership files', 'ARCHITECTURE', r'output collection live in `process_unix\.go`', 'tools/gomad3/runner/internal/execution/launch_plan_unix.go', r'package execution'),
    ('R4', 'prepared-target retention', 'README', r'most recently used binaries stay within 2 GiB', 'tools/gomad3/target/prepared_cache.go', r'maximumPreparedTargetBytes = uint64\(2 << 30\)'),
    ('R4', 'build cache trim', 'README', r'trimmed to 4 GiB', 'tools/gomad3/target/internal/build/trim.go', r'MaximumCacheBytes = 4 << 30'),
    ('R4', 'mapped file bound', 'README', r'total mapped bytes \(64 MiB\) fail closed', 'tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/fs.go', r'maximumMappedBytes = 64 << 20'),
    ('R6', 'coordinator and supervisor are private modes', 'TUTORIAL', r'the CLI starts an isolated coordinator for the campaign', CLI_GO, r'case "__coordinator", "__target_bootstrap", "__supervisor":'),
    ('R6', 'replay does not require choices', 'TUTORIAL', r'An Artifact recorded without Choices can still repeat the Seed', CLI_GO, r'choice-replay=%s'),
    ('R6', 'roadmap link preserved', 'TUTORIAL', r'\[\.plans/GOMAD_NEXT\.md\]\(\.\./\.\./\.plans/GOMAD_NEXT\.md\)', '.plans/GOMAD_NEXT.md', r'^# '),
    ('R7', 'README links the canonical vocabulary', 'README', r'\[Product specification\]\(SPEC\.md#productvocabulary-ubiquitous-language\)', 'tools/gomad3/SPEC.md', r'^## \[PRODUCT\.VOCABULARY\] Ubiquitous Language$'),
    ('R7', 'Parity Case is historical', 'README', r'Parity Case, the name for one of those mapped cases, is a historical term', 'tools/gomad3/SPEC.md', r'^# '),
]
squashed = {name: squash(read(f'tools/gomad3/{name}.md')) for name in GUIDES}
matrix = []
for requirement, claim, guide, statement, source, pattern in MATRIX:
    source_text = read(source) if (ROOT / source).exists() else None
    row = {'requirement': requirement, 'claim': claim, 'guide': guide, 'statement': statement, 'source': source, 'source_pattern': pattern,
           'statement_present': bool(re.search(statement, squashed[guide])),
           'source_matches': source_text is not None and bool(re.search(pattern, source_text, re.M))}
    matrix.append(row)
    if not (row['statement_present'] and row['source_matches']):
        errors.append({**row, 'error': 'claim matrix row failed'})
if 'Parity Case' in read('tools/gomad3/SPEC.md'):
    errors.append({'error': 'historical Parity Case appears in the current vocabulary'})

output_go = read(CLI_DIR + 'explore_output.go')
classification_rows = []
for literal in ['invalid_input', 'unsupported_target', 'semantic_coverage_failure', 'capacity', 'cancelled', 'overall_timeout',
                'runner_failure', 'success', 'target_failure', 'watchdog_observation', 'replay_divergence', 'mixed_failure']:
    ok = f'`{literal}`' in squashed['README'] and f'"{literal}"' in output_go
    classification_rows.append({'classification': literal, 'ok': ok})
    if not ok:
        errors.append({'error': 'explore classification missing from README or source', 'classification': literal})
for name, body in squashed.items():
    for schema in sorted(set(re.findall(r'gomad3\.[a-z0-9-]+/v\d+', body))):
        found = git('grep', '-q', '-F', schema, '--', 'tools/gomad3', 'tools/gomad3sim', ':!*.md', check=False).returncode == 0
        if not found and schema != 'gomad3.simulation-parity/v1':
            errors.append({'error': 'documented schema version absent from source', 'schema': schema, 'guide': name})


# --- platforms ---------------------------------------------------------------

version_platforms = json.loads(read('tools/gomad3/toolchain/version/version.json'))['supported_platforms']
boundary_platforms = json.loads(read('tools/gomad3/deterministicio/boundary/manifest.json'))['platforms']
platforms = {'version_json': version_platforms, 'boundary_manifest': boundary_platforms, 'guides': {}}
if version_platforms != boundary_platforms or version_platforms != ['darwin/arm64', 'linux/amd64']:
    errors.append({'error': 'platform descriptors disagree', **platforms})
for name in ['ARCHITECTURE', 'CLI', 'TUTORIAL', 'README']:
    named = sorted(set(re.findall(r'\b(?:darwin|linux|windows|freebsd)/[a-z0-9]+\b', squashed[name])))
    platforms['guides'][name] = named
    if named != version_platforms:
        errors.append({'error': 'guide names a different platform set', 'guide': name, 'platforms': named})


# --- corpus inventory --------------------------------------------------------

def suites(path):
    return json.loads(read(path))['suites']


def expectation(suite, platform):
    return ((suite.get('platform_expectations') or {}).get(platform) or suite['expectation'])['classification']


core, temporal, tests = (suites(path) for path in ['tools/gomad3/qualification/core.json',
                                                   'tools/gomad3integration/qualification/temporal.json',
                                                   'tools/gomad3integration/qualification/tests.json'])
tier2 = [suite for suite in temporal if suite['tier'] == 2]
tier3 = [suite for suite in temporal if suite['tier'] == 3]
untraced = sorted(suite['test'] for suite in tests if suite.get('choice_bytes') == 0)
inventory = {
    'core_workloads': len(core), 'temporal_workloads': len(temporal), 'temporal_tier2': len(tier2), 'temporal_tier3': len(tier3),
    'temporal_tier3_tests_suites': sum(suite['package'] == './tests' for suite in tier3),
    'temporal_tier2_gomad_tag': sum('gomad' in suite.get('build_tags', []) for suite in tier2),
    'temporal_darwin_qualified': sum(expectation(suite, 'darwin/arm64') == 'qualified' for suite in temporal),
    'temporal_linux_tier2': dict(collections.Counter(expectation(suite, 'linux/amd64') for suite in tier2)),
    'temporal_linux_tier3': dict(collections.Counter(expectation(suite, 'linux/amd64') for suite in tier3)),
    'tests_workloads': len(tests), 'tests_without_choice_trace': untraced,
    'tests_raised_transcript': sorted(suite['test'] for suite in tests if suite.get('io_transcript_bytes')),
    'tests_not_qualified': {suite['test']: {platform: expectation(suite, platform) for platform in version_platforms}
                            for suite in tests if any(expectation(suite, platform) != 'qualified' for platform in version_platforms)},
    'tests_skipped_subtests': sum(len(suite.get('skip') or []) for suite in tests),
    'tests_forward_clock': sorted(suite['test'] for suite in tests if suite.get('clock_tick') == 'forward'),
}
WORDS = {5: 'five', 7: 'seven', 8: 'eight', 9: 'nine', 10: 'ten', 12: 'twelve', 13: 'thirteen', 15: 'fifteen', 28: 'twenty-eight'}
milestones = squash(read(MILESTONES))
expected_text = [
    ('README', f"{WORDS.get(len(core))} assertion-based workloads"),
    ('README', f"holds {WORDS.get(len(tier2))} tier 2 package workloads and {WORDS.get(len(tier3))} tier 3 functional workloads"),
    ('README', f"{WORDS.get(inventory['temporal_tier3_tests_suites'])} `./tests` suites and one `./tests/gomadfunctional` probe"),
    ('README', f"all {WORDS.get(inventory['temporal_darwin_qualified'])} workloads are expected to qualify"),
    ('README', f"{WORDS.get(inventory['temporal_tier2_gomad_tag'])} of the package workloads build with the `gomad` tag"),
    ('README', f"{WORDS.get(inventory['temporal_linux_tier2'].get('qualified'))} package workloads qualify and {WORDS.get(inventory['temporal_linux_tier2'].get('unsupported_target'))} retain exact unsupported analyses"),
    ('README', f"the {WORDS.get(inventory['temporal_linux_tier3'].get('intermittent'))} tier 3 workloads are expected `intermittent`"),
    ('README', f"for the {len(temporal)}-workload Temporal manifest"),
    ('MILESTONES', f"No exact replay for {WORDS.get(len(untraced))} suites"),
]
for where, text in expected_text:
    body = milestones if where == 'MILESTONES' else squashed[where]
    if text not in body:
        errors.append({'error': 'corpus statement disagrees with the manifests', 'where': where, 'expected': text})
for test in untraced + list(inventory['tests_not_qualified']):
    if f'`{test}`' not in milestones:
        errors.append({'error': 'milestones omit a non-default ./tests disposition', 'test': test})
skips = [(suite['test'], subtest) for suite in tests for subtest in suite.get('skip') or []]
skip_rows = []
for suite, subtest in skips:
    # The milestones name a skipped subtest by its leaf, or by the shared suffix of the Nexus outcome pair.
    leaf = subtest.rsplit('/', 1)[-1]
    named = leaf in milestones or leaf.endswith('Operation_Outcomes') and '…Operation_Outcomes' in milestones
    skip_rows.append({'suite': suite, 'subtest': subtest, 'named_in_milestones': bool(named)})
    if not named:
        errors.append({'error': 'milestones omit a skipped ./tests subtest', 'suite': suite, 'subtest': subtest})
outcome_skips = sum(subtest.endswith('Operation_Outcomes') for _, subtest in skips)
if outcome_skips and f'({WORDS.get(outcome_skips, {4: "four"}.get(outcome_skips))} skips)' not in milestones:
    errors.append({'error': 'milestones state a different number of Nexus outcome skips', 'manifest': outcome_skips})
for test in inventory['tests_forward_clock']:
    inventory.setdefault('forward_clock_suites_named_in_milestones', {})[test] = f'`{test}`' in milestones
inventory['skipped_subtests'] = skip_rows
linux_findings = collections.defaultdict(list)
for suite in tier3:
    linux = (suite.get('platform_expectations') or {}).get('linux/amd64') or suite['expectation']
    linux_findings[linux.get('finding', '')].append(suite['id'])
inventory['temporal_linux_tier3_findings'] = {finding: sorted(ids) for finding, ids in linux_findings.items()}
probe_finding = [finding for finding, ids in linux_findings.items() if ids == ['frontend-system-info']]
suite_findings = [finding for finding, ids in linux_findings.items() if 'frontend-system-info' not in ids]
attribution_text = ('the twelve `./tests` suites cite the linux replay divergence recorded in the '
                    '[milestones](../../.plans/GOMAD_MILESTONES.md#open-findings), and the probe cites its own finding')
if len(probe_finding) != 1 or len(suite_findings) != 1 or len(linux_findings[suite_findings[0]]) != 12 or attribution_text not in squashed['README']:
    errors.append({'error': 'README attributes the linux tier 3 expectations differently from temporal.json', 'findings': inventory['temporal_linux_tier3_findings']})
if len(inventory['temporal_linux_tier3']) != 1 or inventory['temporal_darwin_qualified'] != len(temporal):
    errors.append({'error': 'README platform summary no longer matches temporal.json', 'inventory': inventory})
workflow = read('.github/workflows/gomad3.yml')
ci = {'host_tier_step': 'run: make -C tools/gomad3 test-host' in workflow, 'continue_on_error': 'continue-on-error' in workflow}
if not ci['host_tier_step'] or ci['continue_on_error'] or 'runtime, and host tiers as gates' not in squashed['README']:
    errors.append({'error': 'README CI gate statement disagrees with the workflow', **ci})


# --- forward clock probe on the pinned toolchain ----------------------------

toolchain_go = GOMAD3 / '.toolchain/bin/go'
probe = {'toolchain': str(toolchain_go.relative_to(ROOT)), 'state': 'inconclusive'}
EXPECTED = {
    'strict': {'consecutive_now_differ': 'false', 'since_negative_in_busy_stretch': 'false', 'until_positive_in_busy_stretch': 'false',
               'millisecond_readings_tie': 'true', 'now_derived_deadline_after_longer_timer': 'false',
               'since_short_after_one_second_sleep': 'false', 'now_sub_long_after_one_second_sleep': 'false',
               'since_wall_only_reading_positive': 'false', 'until_wall_only_reading_negative': 'false'},
    'forward': {'consecutive_now_differ': 'true', 'since_negative_in_busy_stretch': 'true', 'until_positive_in_busy_stretch': 'true',
                'millisecond_readings_tie': 'true', 'now_derived_deadline_after_longer_timer': 'true',
                'since_short_after_one_second_sleep': 'true', 'now_sub_long_after_one_second_sleep': 'true',
                'since_wall_only_reading_positive': 'true', 'until_wall_only_reading_negative': 'true'},
}
if toolchain_go.exists():
    with tempfile.TemporaryDirectory() as directory:
        work = pathlib.Path(directory)
        shutil.copy(OUT / 'clock-probe.go.txt', work / 'main.go')
        (work / 'go.mod').write_text('module clockprobe\n\ngo 1.27\n')
        environment = {'PATH': '/usr/bin:/bin', 'HOME': directory, 'GOCACHE': str(work / 'cache'), 'CGO_ENABLED': '0',
                       'GOWORK': 'off', 'GOFLAGS': '', 'GOTOOLCHAIN': 'local', 'GOENV': 'off'}
        build = subprocess.run([str(toolchain_go), 'build', '-o', 'probe', '.'], cwd=work, env=environment, capture_output=True, text=True, timeout=300)
        probe['build_status'] = build.returncode
        probe['go_version'] = subprocess.run([str(toolchain_go), 'version'], env=environment, capture_output=True, text=True).stdout.strip()
        if build.returncode == 0:
            probe['state'], probe['observed'] = 'observed', {}
            for policy in ['strict', 'forward']:
                run_environment = {'GOMADSEED': '1', 'TZ': 'UTC'}
                if policy == 'forward':
                    run_environment['GOMAD3_CLOCK_TICK'] = 'forward'
                result = subprocess.run([str(work / 'probe')], env=run_environment, capture_output=True, text=True, timeout=60)
                observed = dict(line.split() for line in result.stdout.splitlines())
                probe['observed'][policy] = {'status': result.returncode, 'readings': observed}
                if result.returncode or observed != EXPECTED[policy]:
                    errors.append({'error': 'clock probe disagrees with the documented behavior', 'policy': policy, 'observed': observed, 'stderr': result.stderr[-400:]})
        else:
            probe['build_stderr'] = build.stderr[-400:]
if probe['state'] != 'observed':
    errors.append({'error': 'clock probe is inconclusive: the pinned toolchain could not build it', **probe})


# --- preservation, whitespace, and binding -----------------------------------

roadmap = {'gomad_next_unchanged_from_head': git('diff', '--quiet', 'HEAD', '--', '.plans/GOMAD_NEXT.md', check=False).returncode == 0,
           'gomad_next_sha256': hashlib.sha256(read('.plans/GOMAD_NEXT.md').encode()).hexdigest(),
           'legacy_roadmaps_absent': not any((ROOT / '.plans' / name).exists() for name in ['GOMAD3_NEXT.md', 'GOMAD_GAPS.md', 'GOMAD_FOLLOWUPS.md'])}
if not all(value for key, value in roadmap.items() if key != 'gomad_next_sha256'):
    errors.append({'error': 'GOMAD_NEXT roadmap update was not preserved', **roadmap})
whitespace = {}
for label, revision in [('baseline_to_current', BASE), ('head_to_current', 'HEAD')]:
    argv = ['diff', '--check', revision, '--', *PATHS]
    result = git(*argv, check=False)
    whitespace[label] = {'argv': ['git', *argv], 'status': result.returncode, 'output': result.stdout[-2000:]}
    if result.returncode:
        errors.append({'error': 'whitespace check failed', 'range': label, 'output': result.stdout[-2000:]})

companions = []
for argv in [[sys.executable, str(OUT / 'verify-vocabulary.py')],
             [sys.executable, str(OUT / 'verify-documentation.py'), str(BIN)],
             ['git', 'diff', '--check', 'HEAD', '--', *PATHS, str(OUT.relative_to(ROOT))],
             [str(FLOWCTL), 'validate', '--spec', OUT.name]]:
    result = subprocess.run(argv, cwd=ROOT, capture_output=True, text=True, timeout=600)
    companions.append({'argv': argv, 'cwd': str(ROOT), 'status': result.returncode, 'output_tail': (result.stdout + result.stderr)[-600:]})
    if result.returncode:
        errors.append({'error': 'companion check failed', 'argv': argv, 'status': result.returncode})

# Bind every file the audit read. HEAD does not identify an uncommitted input, so each
# input records its own hash and whether it differs from HEAD.
input_paths = sorted(inputs)
dirty = {line[3:] for line in git('status', '--porcelain', '--', *input_paths).stdout.splitlines()}
bound_inputs = {path: {'sha256': hashlib.sha256((ROOT / path).read_bytes()).hexdigest(), 'differs_from_head': path in dirty}
                for path in input_paths}
build_key = GOMAD3 / '.toolchain/build-key'
toolchain_binding = {'go_sha256': hashlib.sha256(toolchain_go.read_bytes()).hexdigest() if toolchain_go.exists() else None,
                     'build_key': build_key.read_text().strip() if build_key.exists() else None,
                     'probe_source_sha256': hashlib.sha256((OUT / 'clock-probe.go.txt').read_bytes()).hexdigest()}

report = {
    'schema': 'fn-111.guide-audit/v1',
    'invocation': {'argv': [sys.executable, *sys.argv], 'cwd': str(pathlib.Path.cwd()), 'python': sys.version.split()[0]},
    'companion_checks': companions,
    'inputs': bound_inputs, 'inputs_differing_from_head': sorted(dirty), 'toolchain': toolchain_binding,
    'baseline_revision': BASE, 'head_revision': git('rev-parse', 'HEAD').stdout.strip(),
    'working_tree': git('status', '--short', '--', *PATHS).stdout.splitlines(),
    'file_sha256': {path: 'sha256:' + hashlib.sha256(read(path).encode()).hexdigest() for path in PATHS},
    'binary_sha256': {tool: 'sha256:' + hashlib.sha256((BIN / tool).read_bytes()).hexdigest() for tool in ['gomad', 'gomadtool']},
    'fences': fence_report, 'navigation': navigation, 'stale_terms': stale,
    'commands': commands, 'actual_help': {key: {name: value for name, value in entry.items() if name != 'output'} for key, entry in helps.items()},
    'examples': examples, 'make_examples': make_examples, 'flag_attribution': attribution, 'unknown_documented_flags': unknown_flags,
    'claim_matrix': matrix, 'explore_classifications': classification_rows, 'platforms': platforms,
    'corpus_inventory': inventory, 'ci': ci, 'clock_probe': probe, 'roadmap': roadmap, 'whitespace': whitespace,
    'requirement_evidence': {
        'R1': 'vocabulary-audit.json term_assessments (task 1); README Parity Case matrix row',
        'R2': 'vocabulary-audit.json identifier comparison (task 1); documentation-audit.json identifiers_preserved_in_order; stale_terms',
        'R3': 'vocabulary-audit.json distinctions (task 1); claim_matrix backend/fidelity and tape rows',
        'R4': 'claim_matrix R4 rows; clock_probe; platforms; corpus_inventory',
        'R5': 'examples; make_examples; flag_attribution; claim_matrix R5 rows; explore_classifications; actual_help',
        'R6': 'claim_matrix R6 rows; roadmap; platforms.guides.TUTORIAL',
        'R7': 'navigation; fences; stale_terms; claim_matrix R7 rows',
        'R8': 'head_revision; file_sha256; binary_sha256; whitespace; this mapping',
    },
    'fn109_r9_d5_reuse': {
        'reused': ['both supported platforms named consistently with version.json and the boundary manifest (platforms)',
                   'implemented choice replay and exploration and both backends described against source (claim_matrix R4 rows)',
                   'capability support, repeatability, exact replay, and expectation matching kept separate (corpus_inventory, README corpus statements)',
                   'current residual findings recorded: the milestones name every tests.json suite that runs without a choice trace, every suite whose expectation is not qualified, and every skipped subtest (corpus_inventory); skip reasons and owners stay in tests.generator.json and are not compared'],
        'not_covered': ['intentional Go interface changes from fn-109 R4-R7 and R11-R14, which are not implemented',
                        'architecture fitness checks (fn-109 R8 / fn-105 D4)', 'any documentation of interfaces that fn-109 has yet to change'],
        'owner': 'fn-109 R9 and fn-105 D5 stay open; this audit closes neither',
    },
    'still_open': ['fn-105 D12 and D14 replay fixes', 'fn-105 D13 tracing policy and D15 trace capacity', 'fn-109 R9 interface documentation'],
    'limits': ['head_revision identifies the commit below an uncommitted tree; inputs and file_sha256 bind what was actually read, and inputs_differing_from_head lists the uncommitted ones.',
               'Statement patterns prove a sentence is present and its implementing line exists; the pairing of each row was judged by reading both.',
               'linux/amd64 behavior is taken from source, manifests, and the CI workflow; the probe and help text ran on the host recorded in clock_probe.',
               'No qualification workload was executed; manifest expectations are reported as expectations, not as observed qualification.'],
    'errors': errors,
}
(OUT / 'guide-audit.json').write_text(json.dumps(report, indent=2, ensure_ascii=False) + '\n')
print(json.dumps({'head': report['head_revision'], 'examples_checked': len(examples), 'example_errors': sum('error' in item for item in examples),
                  'make_targets_checked': len(make_examples), 'claim_rows': len(matrix),
                  'claim_rows_passing': sum(row['statement_present'] and row['source_matches'] for row in matrix),
                  'links_checked': len(navigation), 'fenced_blocks': {path: sum(counts.values()) for path, counts in fence_report.items()},
                  'clock_probe': probe['state'], 'whitespace': {label: item['status'] for label, item in whitespace.items()},
                  'errors': errors}, indent=2, ensure_ascii=False))
sys.exit(bool(errors))
