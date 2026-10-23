"""Read original Go source without mistaking embedded guest source for tests.

This inventory records source obligations. It supplies no execution verdict.
"""
import hashlib
import re


def masked(source):
    result = bytearray(source)
    cursor = 0
    while cursor < len(source):
        start = cursor
        if source[cursor:cursor + 2] == b'//':
            end = source.find(b'\n', cursor + 2)
            cursor = len(source) if end < 0 else end
        elif source[cursor:cursor + 2] == b'/*':
            end = source.find(b'*/', cursor + 2)
            if end < 0: raise ValueError('unterminated Go block comment')
            cursor = end + 2
        elif source[cursor] in (34, 39, 96):
            quote = source[cursor]
            cursor += 1
            while cursor < len(source):
                if source[cursor] == quote:
                    cursor += 1
                    break
                if source[cursor] == 92 and quote != 96: cursor += 1
                cursor += 1
            else:
                raise ValueError('unterminated Go literal')
        else:
            cursor += 1
            continue
        for index in range(start, cursor):
            if result[index] not in (10, 13): result[index] = 32
    return bytes(result)


def closing(code, start, left, right):
    depth = 0
    for cursor in range(start, len(code)):
        if code[cursor] == left: depth += 1
        if code[cursor] == right:
            depth -= 1
            if not depth: return cursor
    raise ValueError('unbalanced Go source delimiter')


def arguments(code, start):
    """Locate the first call argument using masked original byte coordinates."""
    stack = []
    pairs = {41: 40, 93: 91, 125: 123}
    for cursor in range(start, len(code)):
        token = code[cursor]
        if token == 44 and not stack: return cursor
        if token in (40, 91, 123): stack.append(token)
        elif token in pairs:
            if not stack: return cursor
            if stack.pop() != pairs[token]: raise ValueError('mismatched Go source delimiter')
    raise ValueError('unterminated Go call argument')


def functions(source):
    code = masked(source)
    result = []
    previous_end = 0
    for match in re.finditer(rb'^func (\w+)\(', code, re.M):
        if match.start() < previous_end: continue
        open_paren = match.end() - 1
        parameters_end = closing(code, open_paren, 40, 41)
        body_start = code.find(b'{', parameters_end)
        if body_start < 0: raise ValueError('missing Go function body')
        # Anonymous struct/interface result types precede the function body.
        while re.search(rb'\b(?:struct|interface)\s*$', code[parameters_end:body_start]):
            parameters_end = closing(code, body_start, 123, 125) + 1
            body_start = code.find(b'{', parameters_end)
            if body_start < 0: raise ValueError('missing Go function body')
        end = closing(code, body_start, 123, 125) + 1
        previous_end = end
        body = code[body_start:end]
        calls = sorted(set(item[1].decode() for item in re.finditer(rb'(?<![\w.])(\w+)\s*\(', body))
                       - {'func', 'if', 'for', 'switch', 'select', 'return'})
        subtests = []
        for item in re.finditer(rb'\bt\.Run\(', body):
            start = body_start + item.start()
            argument_start = body_start + item.end()
            argument_end = arguments(code, argument_start)
            call_end = closing(code, argument_start - 1, 40, 41) + 1
            parents = [parent for parent in subtests if parent['start_byte'] < start < parent['end_byte']]
            subtests.append(dict(start_byte=start, end_byte=call_end,
                                 expression=source[argument_start:argument_end].decode().strip(),
                                 parent=parents[-1]['start_byte'] if parents else None))
        result.append(dict(name=match[1].decode(), start_byte=match.start(), end_byte=end,
                           start_line=source[:match.start()].count(b'\n') + 1,
                           end_line=source[:end].count(b'\n') + 1,
                           sha256=hashlib.sha256(source[match.start():end]).hexdigest(),
                           calls=calls, subtests=subtests))
    return result


SLIM = {
    'common/archiver/provider/provider_cloud_gomad_test.go': 'TestCloudArchiversRefuseConfiguredSchemesWithoutFallback',
    'temporal/membership_test.go': 'TestDefaultMembershipLinksRingpop',
    'tests/testcore/flag_sql_wasm_slim_test.go': 'TestSlimGuestRefusesUnlinkedSQLDriversWithoutFallback',
}
PREFIX = 'tools/gomad/internal/backends/wasm/'
MAKEFILES = ('tools/gomad/Makefile', PREFIX + 'Makefile')


def inventory(repo, commit, retained_repo=None):
    """Inventory exact source obligations; unresolved rows cannot earn a PASS."""
    import json
    from pathlib import Path
    import subprocess
    from collections import defaultdict
    if not re.fullmatch('[0-9a-f]{40}', commit): raise ValueError('original source commit must be exact')
    def git(*args):
        return subprocess.check_output(['git', '-C', str(repo), *args])
    tree = git('ls-tree', '-r', '-z', commit, '--', PREFIX, *SLIM, MAKEFILES[0])
    sources, makes, contents, declarations = {}, {}, {}, {}
    for row in tree.split(b'\0'):
        if not row: continue
        header, path = row.decode().split('\t'); mode, kind, blob = header.split()
        if not (path.endswith('_test.go') or path in MAKEFILES): continue
        if mode != '100644' or kind != 'blob': raise ValueError('nonregular original source')
        raw = git('cat-file', 'blob', blob)
        item = dict(blob=blob, sha256=hashlib.sha256(raw).hexdigest())
        if path in MAKEFILES:
            makes[path] = item
        else:
            sources[path] = item; contents[path] = raw; declarations[path] = functions(raw)
    if set(makes) != set(MAKEFILES) or not set(SLIM) <= set(sources):
        raise ValueError('incomplete original Make/source universe')
    retained = defaultdict(list)
    if retained_repo is not None:
        for path in sorted((Path(retained_repo) / 'tools/gomad').rglob('*_test.go')):
            if 'testdata' in path.parts: continue
            raw = path.read_bytes()
            for item in functions(raw):
                if re.fullmatch(r'Test[A-Z0-9_]\w*', item['name']):
                    relative = str(path.relative_to(retained_repo))
                    retained[item['name']].append(dict(source=relative, source_sha256=hashlib.sha256(raw).hexdigest(),
                        span_sha256=item['sha256'], testcase='go/go.temporal.io/server/' +
                        relative.rsplit('/', 1)[0] + '/' + item['name']))
    packages = defaultdict(dict)
    for path, items in declarations.items():
        if '/testdata/' in path: continue
        for item in items:
            if item['name'] in packages[path.rsplit('/', 1)[0]]:
                raise ValueError('ambiguous original package helper: ' + item['name'])
            packages[path.rsplit('/', 1)[0]][item['name']] = (path, item)
    roots, sites, excluded = {}, {}, {}
    def literal(expression):
        if re.fullmatch(r'"[^"\\]*"', expression): return json.loads(expression)
        return None
    for path, items in declarations.items():
        raw = contents[path]
        if '/testdata/' in path:
            excluded[path] = dict(reason='Go recursive package discovery excludes testdata directories',
                                  test_roots=[item['name'] for item in items if re.fullmatch(r'Test[A-Z0-9_]\w*', item['name'])])
            continue
        package = packages[path.rsplit('/', 1)[0]]
        for item in items:
            name = item['name']
            if not re.fullmatch(r'Test[A-Z0-9_]\w*', name) or name == 'TestMain': continue
            if path in SLIM and name != SLIM[path]: continue
            key = path + '#' + name
            closure, pending = {}, [(path, item)]
            while pending:
                helper_path, helper = pending.pop(); helper_id = helper_path + '#' + helper['name']
                if helper_id in closure: continue
                closure[helper_id] = (helper_path, helper)
                for call in helper['calls']:
                    if call in package: pending.append(package[call])
            direct_sites, helper_sites = [], []
            for helper_id, (helper_path, helper) in sorted(closure.items()):
                for site in helper['subtests']:
                    site_id = key + '@' + ('' if helper_id == key else helper_path + ':') + str(site['start_byte'])
                    value = literal(site['expression'])
                    sites[site_id] = dict(source=helper_path, owner=helper_id, line=contents[helper_path][:site['start_byte']].count(b'\n') + 1,
                        expression=site['expression'], parent_start_byte=site['parent'],
                        expansions=[] if value is None else [value], status='UNEXECUTED')
                    (direct_sites if helper_id == key else helper_sites).append(site_id)
            span = {field: item[field] for field in ('start_byte', 'end_byte', 'start_line', 'end_line', 'sha256')}
            body = raw[item['start_byte']:item['end_byte']]
            code = masked(body)
            assertion_lines, skips = [], []
            for offset, line in enumerate(code.splitlines()):
                original = body.splitlines()[offset].decode().strip()
                absolute = item['start_line'] + offset
                if re.search(rb'\.(?:Fatalf?|Errorf?|FailNow|Fail)\s*\(', line):
                    assertion_lines.append(dict(line=absolute, source=original))
                if re.search(rb'\.Skipf?\s*\(', line): skips.append(dict(line=absolute, source=original))
            roots[key] = dict(source_span=span, status='UNEXECUTED', evidence=[],
                retained_exact_source_candidates=[candidate for candidate in retained[name] if candidate['span_sha256'] == item['sha256']],
                retained_changed_source_candidates=[candidate for candidate in retained[name] if candidate['span_sha256'] != item['sha256']],
                assertion_lines=assertion_lines, skip_predicates=skips,
                helper_source_spans={helper_id: dict(source=helper_path, **{field: helper[field] for field in ('start_byte', 'end_byte', 'sha256')})
                    for helper_id, (helper_path, helper) in sorted(closure.items()) if helper_id != key},
                subtest_sites=direct_sites, helper_subtest_sites=helper_sites,
                build_constraints=re.findall(r'^//go:build (.*)', raw.decode(), re.M))
    return dict(schema='gomad.original-test-wasm-source-inventory/v1', reference_commit=commit, qualification=False,
        makefiles=makes, sources=sources, discovery_exclusions=excluded, callables=roots, subtest_sites=sites,
        counts=dict(callables=len(roots), source_files=len(sources), excluded_fixture_files=len(excluded),
                    subtest_sites=len(sites), unresolved_dynamic_sites=sum(not item['expansions'] for item in sites.values()),
                    retained_exact_source_candidates=sum(bool(item['retained_exact_source_candidates']) for item in roots.values()),
                    retained_changed_source_candidates=sum(bool(item['retained_changed_source_candidates']) for item in roots.values())))
