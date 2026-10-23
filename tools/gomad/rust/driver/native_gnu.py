#!/usr/bin/env python3
"""Closed prefreeze Linux GNU compiler envelope for the native identity harness."""
import argparse
import os
from pathlib import Path
import re
import shutil
import stat
import struct
import sys
import time

from driver import HERE, command, read, require, sha, write
import live
import native_roles

SYSTEM = Path('/')
SCHEMA = 'gomad.gnu-identity-toolchain/v1'
BINUTILS = ('as', 'ld', 'ar', 'ranlib', 'nm', 'objcopy', 'objdump', 'strip')
DELEGATES = ('cc1', 'cc1plus', 'collect2', 'lto-wrapper', *BINUTILS)
SOURCES = ('native_gnu.py', 'driver.py', 'launch.py', 'native_roles.py', 'live.py', 'primary_stages.py')
PREFIXES = ('/usr/libexec', '/usr/lib64', '/usr/lib', '/usr/local/lib', '/usr/include', '/usr/local/include', '/lib', '/lib64')


def native_machine():
    require(sys.platform.startswith('linux') and os.uname().machine in ('aarch64', 'x86_64'),
            'GNU identity source-build envelope requires native Linux aarch64/x86_64')
    return 183 if os.uname().machine == 'aarch64' else 62


def elf(raw):
    if raw[:4] != b'\x7fELF':
        return None
    require(len(raw) >= 64 and raw[4:6] == b'\x02\x01', 'GNU envelope supports ELF64 little-endian only')
    machine = struct.unpack_from('<H', raw, 18)[0]
    require(machine in (62, 183), 'GNU envelope supports native x86_64/aarch64 only')
    offset = struct.unpack_from('<Q', raw, 32)[0]
    size, count = struct.unpack_from('<HH', raw, 54)
    require((size == 56 or count == size == 0) and count <= 1024 and offset + size * count <= len(raw), 'malformed compiler ELF program headers')
    headers = [struct.unpack_from('<IIQQQQQQ', raw, offset + index * size) for index in range(count)]
    def segment(header):
        begin, length = header[2], header[5]
        require(begin + length <= len(raw), 'malformed compiler ELF segment')
        return raw[begin:begin + length]
    interpreter = [segment(header).rstrip(b'\0').decode() for header in headers if header[0] == 3]
    require(len(interpreter) <= 1, 'multiple compiler ELF interpreters')
    dynamic = [segment(header) for header in headers if header[0] == 2]
    require(len(dynamic) <= 1, 'multiple compiler ELF dynamic tables')
    if not dynamic:
        return dict(machine=machine, interpreter=interpreter, needed=[], search=[])
    table = dynamic[0]
    require(len(table) % 16 == 0, 'malformed compiler ELF dynamic entries')
    entries = []
    for position in range(0, len(table), 16):
        tag, value = struct.unpack_from('<qQ', table, position)
        if tag == 0:
            break
        entries.append((tag, value))
    strings = [value for tag, value in entries if tag == 5]
    lengths = [value for tag, value in entries if tag == 10]
    require(len(strings) == len(lengths) == 1, 'compiler ELF string table missing')
    locations = [header[2] + strings[0] - header[3] for header in headers if header[0] == 1
                 and header[3] <= strings[0] and strings[0] + lengths[0] <= header[3] + header[5]]
    require(len(locations) == 1 and lengths[0] <= 64 << 20, 'compiler ELF string table outside load segments')
    strings = raw[locations[0]:locations[0] + lengths[0]]
    def text(index):
        require(index < len(strings) and b'\0' in strings[index:], 'unterminated compiler ELF string')
        return strings[index:strings.index(b'\0', index)].decode()
    return dict(machine=machine, interpreter=interpreter,
                needed=[text(value) for tag, value in entries if tag == 1],
                search=[part for tag, value in entries if tag in (15, 29) for part in text(value).split(':')])


def aliases(path, table):
    path = Path(path)
    require(path.is_absolute(), 'GNU input must be absolute')
    for _ in range(40):
        current = Path(path.anchor)
        found = False
        for part in path.parts[1:]:
            current /= part
            if current.is_symlink():
                target = os.readlink(current)
                table[str(current)] = target
                tail = path.relative_to(current)
                path = (Path(target) if Path(target).is_absolute() else current.parent / target) / tail
                path = Path(os.path.normpath(path))
                found = True
                break
        if not found:
            return Path(os.path.normpath(path))
    raise ValueError('GNU input alias cycle')


def probes(linker, loader):
    result = {name: [linker, flag] for name, flag in (
        ('version', '--version'), ('machine', '-dumpmachine'), ('gcc-version', '-dumpfullversion'),
        ('search', '-print-search-dirs'), ('specs', '-dumpspecs'), ('sysroot', '-print-sysroot'))}
    result.update({name: [linker, '-print-prog-name=' + name] for name in DELEGATES})
    result.update({'include-c': [linker, '-E', '-x', 'c', '/dev/null', '-v'],
                   'include-cxx': [linker, '-E', '-x', 'c++', '/dev/null', '-v'], 'loader-cache': [loader, '-p']})
    return result


def probe_environment(root):
    return dict(PATH='/usr/bin:/bin', HOME=str(root / 'home'), TMPDIR=str(root / 'tmp'), LC_ALL='C', TZ='UTC')


def validate_linker(linker):
    require(Path(linker).resolve() == Path(linker) and re.fullmatch(
        re.escape(str(SYSTEM / 'usr/bin')) + r'/(aarch64|x86_64)-linux-gnu-gcc-[0-9]+', linker),
        'GNU envelope requires its supported canonical system GCC driver')
    info = elf(native_roles.regular(linker).read_bytes())
    require(info and info['machine'] == native_machine(), 'GNU driver is not a native supported ELF executable')


def derive(root, tools_path, linker):
    require(root.is_dir() and root.resolve() == root and stat.S_IMODE(root.stat().st_mode) == 0o700
            and root.stat().st_uid == os.geteuid(), 'GNU prefreeze root is not its private actual owner directory')
    tools = native_roles.read(tools_path)
    repo = tools['repo']
    require(tools['schema'] == 'gomad.driver-static-plan/v1', 'GNU envelope requires the actual static tool plan')
    validate_linker(linker)
    alias_table, files, missing = {}, set(), set()
    def add(path):
        resolved = aliases(path, alias_table)
        native_roles.regular(resolved)
        files.add(str(resolved))
        return resolved
    loader = str(aliases(SYSTEM / 'usr/sbin/ldconfig', alias_table))
    owner = native_roles.read(root / 'producer.json')
    require(set(owner) == {'pid', 'start_monotonic_nanos', 'end_monotonic_nanos', 'sources'}
            and type(owner['pid']) is int and owner['pid'] > 0
            and type(owner['start_monotonic_nanos']) is int and type(owner['end_monotonic_nanos']) is int
            and 0 < owner['start_monotonic_nanos'] < owner['end_monotonic_nanos']
            and owner['sources'] == {str(HERE / name): sha(HERE / name) for name in SOURCES}, 'GNU prefreeze producer identity changed')
    actual_probes = probes(linker, loader)
    commands, pids = {}, set()
    for name, argv in actual_probes.items():
        row = native_roles.read(root / (name + '.command.json'))
        process = native_roles.read(root / (name + '.process.json'))
        live.completed_command(row, process, argv, repo)
        require(row['environment_mode'] == 'exact' and row['effective_environment'] == row['environment'] == probe_environment(root),
                'GNU probe substituted its environment')
        require(all(type(row[field]) is int for field in ('parent_pid', 'started_monotonic_nanos', 'reaped_monotonic_nanos'))
                and row['parent_pid'] == process['stage_pid'] == owner['pid'] and row['clock'] == process['clock'] == 'CLOCK_MONOTONIC'
                and row['pid'] not in pids and owner['start_monotonic_nanos'] <= row['started_monotonic_nanos']
                < row['reaped_monotonic_nanos'] <= owner['end_monotonic_nanos'], 'GNU probe borrowed another producer/lifecycle')
        pids.add(row['pid'])
        for field in ('parent_pid', 'started_monotonic_nanos', 'reaped_monotonic_nanos'):
            require(row[field] == process[field], 'GNU probe process/command lifecycle differs')
        commands[name] = {suffix: sha(add(root / (name + '.' + suffix))) for suffix in ('command.json', 'process.json', 'stdout', 'stderr')}
    output = lambda name: (root / (name + '.stdout')).read_text().strip()
    machine = output('machine')
    require(machine in ('aarch64-linux-gnu', 'x86_64-linux-gnu') and machine in linker
            and 'gcc' in output('version').splitlines()[0].lower() and re.fullmatch(r'[0-9]+(?:\.[0-9]+)*', output('gcc-version'))
            and output('specs') and output('sysroot') in ('', '/'), 'unsupported GNU compiler family/machine/specs/sysroot')
    expected_machine = 183 if machine.startswith('aarch64') else 62
    require(native_machine() == expected_machine, 'GNU toolchain does not match this actual native source-build host')
    require(elf(Path(linker).read_bytes())['machine'] == expected_machine, 'GNU compiler is not native')
    programs = {}
    for name in DELEGATES:
        declared = output(name)
        path = declared if Path(declared).is_absolute() else shutil.which(declared, path=str(SYSTEM / 'usr/bin') + ':' + str(SYSTEM / 'bin'))
        require(path, 'missing GNU delegate ' + name)
        resolved = add(path)
        require(any(resolved.is_relative_to(SYSTEM / prefix) for prefix in ('usr/bin', 'usr/lib/gcc', 'usr/libexec/gcc', 'usr/' + machine + '/bin')) and elf(resolved.read_bytes())['machine'] == expected_machine,
                'unsupported GNU delegate placement or architecture')
        programs[name] = str(resolved)
    def supported(path):
        resolved = aliases(path, alias_table)
        require(Path(path).is_absolute() and any(resolved.is_relative_to((SYSTEM / prefix.lstrip('/')).resolve()) for prefix in (*PREFIXES, '/usr/' + machine + '/lib', '/usr/' + machine + '/include')),
                'GNU search path escapes the supported system envelope')
        return resolved
    walked = set()
    def walk(path):
        resolved = aliases(path, alias_table)
        if resolved in walked: return
        walked.add(resolved)
        for directory, directories, names in os.walk(resolved, followlinks=False):
            for name in names:
                add(Path(directory) / name)
            for name in directories:
                child = Path(directory) / name
                if child.is_symlink():
                    target = aliases(child, alias_table)
                    require(target.is_relative_to(resolved) or any(target.is_relative_to((SYSTEM / prefix.lstrip('/')).resolve())
                            for prefix in PREFIXES), 'GNU directory alias escapes supported compiler/include roots')
                    if not target.is_relative_to(resolved): walk(target)
    program_roots = re.findall(r'^programs: =(.*)$', (root / 'search.stdout').read_text(), re.MULTILINE)
    require(len(program_roots) == 1, 'GNU executable search probe missing')
    program_search = []
    for name in program_roots[0].split(':'):
        path = aliases(name, alias_table)
        require(path.is_absolute() and any(path.is_relative_to((SYSTEM / prefix).resolve())
                for prefix in ('usr/lib', 'usr/libexec', 'usr/bin', 'bin', 'usr/' + machine + '/bin')), 'GNU executable search root escapes its supported prefix')
        program_search.append(str(path))
        for delegate in DELEGATES:
            candidate = path / delegate
            if candidate.is_file(): add(candidate)
            else: missing.add(str(candidate))
    version_root = Path(programs['cc1']).parent
    require(any(version_root.is_relative_to(SYSTEM / prefix) for prefix in ('usr/lib/gcc', 'usr/libexec/gcc')), 'GNU internal compiler root missing')
    walk(version_root)
    libraries = re.findall(r'^libraries: =(.*)$', (root / 'search.stdout').read_text(), re.MULTILINE)
    require(len(libraries) == 1, 'GNU library search probe missing')
    search = []
    for name in libraries[0].split(':'):
        path = supported(name)
        if not path.exists():
            missing.add(str(path)); continue
        require(path.is_dir(), 'GNU library search root is not a directory')
        search.append(str(path))
        for entry in path.iterdir():
            if entry.is_file():
                add(entry)
        for child in ('ldscripts', 'bfd-plugins'):
            if (path / child).is_dir():
                walk(path / child)
    includes = []
    for name in ('include-c', 'include-cxx'):
        raw = (root / (name + '.stderr')).read_text()
        match = re.search(r'#include <\.\.\.> search starts here:\n(.*?)End of search list\.', raw, re.DOTALL)
        require(match, 'GNU actual header search probe missing')
        for line in match[1].splitlines():
            path = supported(line.strip())
            require(path.is_dir(), 'GNU header search input missing')
            includes.append(str(path)); walk(path)
    require(not (SYSTEM / 'etc/ld.so.preload').exists(), 'GNU envelope refuses a global loader preload')
    missing.add(str(SYSTEM / 'etc/ld.so.preload'))
    for name in ('etc/ld.so.cache', 'etc/ld.so.conf'):
        add(SYSTEM / name)
    if (SYSTEM / 'etc/ld.so.conf.d').is_dir():
        walk(SYSTEM / 'etc/ld.so.conf.d')
    add(linker); add(loader); add(tools_path); add(root / 'producer.json')
    for name in SOURCES:
        add(HERE / name)
    rust_root = Path(tools['tools']['rustc']).parent.parent
    walk(rust_root)
    python = add(tools['tools']['python'])
    git = add(shutil.which('git', path=str(SYSTEM / 'usr/bin') + ':' + str(SYSTEM / 'bin')))
    cache = {}
    for name, candidate in re.findall(r'^\s*(\S+)\s+\([^\n]*\)\s+=>\s+(\S+)\s*$', (root / 'loader-cache.stdout').read_text(), re.MULTILINE):
        cache.setdefault(name, []).append(candidate)
    pending = list(files)
    visited = set()
    while pending:
        path = Path(pending.pop())
        if str(path) in visited:
            continue
        visited.add(str(path))
        info = elf(path.read_bytes())
        if info is None:
            continue
        require(info['machine'] == expected_machine, 'GNU closure contains a foreign dynamic ELF machine')
        for interpreter in info['interpreter']:
            candidate = add(interpreter)
            if str(candidate) not in visited: pending.append(str(candidate))
        for needed in info['needed']:
            require('/' not in needed, 'GNU closure has an unsupported absolute DT_NEEDED entry')
            candidates = list(cache.get(needed, []))
            for value in info['search']:
                require(value and '$' not in value.replace('${ORIGIN}', '').replace('$ORIGIN', ''), 'unsupported GNU dynamic-loader expansion')
                base = value.replace('${ORIGIN}', str(path.parent)).replace('$ORIGIN', str(path.parent))
                require(Path(base).is_absolute(), 'relative GNU ELF loader search')
                candidate = Path(base) / needed
                if candidate.is_file(): candidates.append(str(candidate))
            found = False
            for candidate in candidates:
                resolved = aliases(candidate, alias_table)
                if not resolved.is_file(): continue
                other = elf(resolved.read_bytes())
                if other and other['machine'] == expected_machine:
                    found = True; resolved = add(candidate)
                    if str(resolved) not in visited: pending.append(str(resolved))
            require(found, 'GNU compiler ELF dependency is not captured: ' + needed)
    bin_dir = root / 'bin'
    links = dict(cc=linker, gcc=linker, **{'c++': linker, 'g++': linker, 'git': str(git)})
    links.update({name: programs[name] for name in BINUTILS})
    require(bin_dir.is_dir() and bin_dir.resolve() == bin_dir and stat.S_IMODE(bin_dir.stat().st_mode) == 0o700
            and bin_dir.stat().st_uid == os.geteuid() and {path.name for path in bin_dir.iterdir()} == set(links),
            'GNU executable lookup contains foreign or missing entries')
    for name, target in links.items():
        path = bin_dir / name
        require(path.is_symlink() and os.readlink(path) == target, 'GNU closed executable lookup changed')
        aliases(path, alias_table)
    require(len(files) <= 100000 and sum(Path(path).stat().st_size for path in files) <= 16 << 30, 'GNU compiler envelope exceeds bounded native toolchain inputs')
    return dict(schema=SCHEMA, root=str(root), tools_plan=str(tools_path), linker=linker, machine=machine,
                programs=programs, python=str(python), git=str(git), program_search=sorted(set(program_search)), search=sorted(set(search)), includes=sorted(set(includes)),
                missing=sorted(missing), aliases=alias_table, files={path: sha(path) for path in sorted(files)}, commands=commands)


def prepare(tools_path, linker, root):
    tools = native_roles.read(tools_path)
    validate_linker(linker)
    require(root.is_absolute() and root.resolve() == root and not root.exists(), 'GNU prefreeze root must be fresh canonical absolute')
    root.mkdir(mode=0o700)
    for name in ('home', 'tmp', 'bin'):(root / name).mkdir(mode=0o700)
    native_machine()
    info = Path('/dev/null').stat()
    require(stat.S_ISCHR(info.st_mode) and info.st_rdev == os.makedev(1, 3), 'GNU header probes require actual Linux /dev/null')
    started = time.clock_gettime_ns(time.CLOCK_MONOTONIC)
    loader = str((SYSTEM / 'usr/sbin/ldconfig').resolve())
    env = probe_environment(root)
    for name, argv in probes(linker, loader).items():
        command(dict(repo=tools['repo'], environment=env), root, name, argv, timeout=60, exact_environment=env)
        if name == 'version':
            require('gcc' in (root / 'version.stdout').read_text().splitlines()[0].lower(), 'GNU envelope requires actual GNU GCC')
    write(root / 'producer.json', dict(pid=os.getpid(), start_monotonic_nanos=started,
        end_monotonic_nanos=time.clock_gettime_ns(time.CLOCK_MONOTONIC), sources={str(HERE / name): sha(HERE / name) for name in SOURCES}))
    programs = {}
    for name in BINUTILS:
        value=(root / (name+'.stdout')).read_text().strip()
        programs[name]=str(Path(value if Path(value).is_absolute() else shutil.which(value,path=str(SYSTEM / 'usr/bin') + ':' + str(SYSTEM / 'bin'))).resolve())
    git=str(Path(shutil.which('git',path=str(SYSTEM / 'usr/bin') + ':' + str(SYSTEM / 'bin'))).resolve())
    for name,target in dict(cc=linker,gcc=linker,**{'c++':linker,'g++':linker,'git':git},**programs).items():
        (root/'bin'/name).symlink_to(target)
    result = derive(root, tools_path, linker)
    write(root / 'gnu-toolchain.json', result)
    return result


def inspect(config, tools_path):
    binding = config.get('identity_harness', {})
    require(set(binding) == {'toolchain_plan'}, 'identity harness requires its actual prepared GNU toolchain plan')
    path = Path(binding['toolchain_plan'])
    declared = native_roles.read(path)
    require(declared['schema'] == SCHEMA and declared['tools_plan'] == str(tools_path)
            and path == Path(declared['root']) / 'gnu-toolchain.json', 'GNU toolchain plan has foreign roots/static tools')
    actual = derive(Path(declared['root']), tools_path, declared['linker'])
    require(declared == actual, 'GNU toolchain probes/delegates/search/loader closure changed')
    return path, actual


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode',choices=('prepare',))
    parser.add_argument('--tools-plan',type=Path,required=True)
    parser.add_argument('--linker',required=True)
    parser.add_argument('--root',type=Path,required=True)
    args=parser.parse_args()
    prepare(args.tools_plan,args.linker,args.root)


if __name__=='__main__':main()
