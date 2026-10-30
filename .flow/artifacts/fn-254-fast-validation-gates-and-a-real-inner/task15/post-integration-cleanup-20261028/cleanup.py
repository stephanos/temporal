"""Prepared exact-private-cache cleanup; only explicit reviewed execution can mutate the fixed target."""
import argparse
import datetime
import hashlib
import importlib.util
import json
import os
import re
import shutil
import signal
import stat
import subprocess
import sys
import uuid
from pathlib import Path

ROOT = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
HERE = Path(__file__).resolve().parent
PROOF = Path('/tmp/umpire-fn25415-final-9babff6c815b-47070/proof-bKkF3n')
TARGET = Path('/tmp/umpire-fn25415-final-9babff6c815b-47070/proof-bKkF3n/target')
PARENT = ROOT / '.flow/tmp/fn25416-provider-validation/target'
CAS = ROOT / '.recovery/fn25415-native-proof'
CARGO = Path('/home/agent/.rustup/toolchains/1.99.1-aarch64-unknown-linux-gnu/bin/cargo')
PRIMARY_SHA = '68208a5aa0e5df79783d97b740a07da3f0808dce4935aa076450630a5183b610'
SUPPLEMENTAL_SHA = 'e22efa29f106d7a2f8bf96374b080ee8d5c67b73b1dc4adc864daf7b9554a06a'
CLASSIFICATION_SHA = '78fa352ec05730c873597c1b928d80734e779884861dbc1a2915181997d6e9a5'
TOOLS_SHA = '8b11813eda5a3a5482ca065f8603e57fdaf43b946b3db40acc67345128550de2'
KNOWN_FOREIGN = {10: (0, 'socat'), 542: (0, 'dockerd'), 557: (0, 'containerd')}
TAG_SIGNATURE = b'Signature: 8a477f597d28d172789f06886806bc55'


def digest(path):
    value = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            value.update(chunk)
    return value.hexdigest()


def timestamp():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def free_space():
    rows = []
    for path in [ROOT, Path('/tmp')]:
        info = os.statvfs(path)
        rows.append(dict(path=str(path), realpath=str(path.resolve()), device=path.stat().st_dev,
                         free_bytes=info.f_bavail * info.f_frsize))
    return rows


def allocated(target):
    if not os.path.lexists(target):
        return 0
    if target.is_symlink() or not target.is_dir():
        return target.lstat().st_blocks * 512
    inodes = {}
    for parent, directories, files in os.walk(target, followlinks=False):
        for name in ['.', *directories, *files]:
            path = Path(parent) if name == '.' else Path(parent) / name
            info = path.lstat()
            inodes[(info.st_dev, info.st_ino)] = info.st_blocks * 512
    return sum(inodes.values())


def node(path, include_hash=False):
    info = path.lstat()
    value = dict(device=info.st_dev, inode=info.st_ino, mode=oct(info.st_mode))
    if stat.S_ISREG(info.st_mode):
        value.update(size=info.st_size, mtime_ns=info.st_mtime_ns)
        if include_hash:
            value['sha256'] = digest(path)
    elif stat.S_ISLNK(info.st_mode):
        value['link'] = os.readlink(path)
    return value


def walk_error(error):
    raise error


def protected_snapshot():
    # Active task12 may evolve parent contents; preserve its exact directory identity, never clean it.
    identities = {str(path): node(path) for path in [ROOT, ROOT/'model', PROOF, PROOF/'framework', PARENT, ROOT/'.recovery', ROOT.parent/'.recovery']}
    proof_files = {}
    for parent, directories, files in os.walk(PROOF, followlinks=False, onerror=walk_error):
        directory = Path(parent)
        if directory == PROOF:
            directories[:] = [name for name in directories if name != 'target']
        for name in directories + files:
            path = directory / name
            if path.is_dir() and not path.is_symlink():
                continue
            proof_files[str(path.relative_to(PROOF))] = node(path, include_hash=True)
    recovery = {}
    for recovery_root in [ROOT/'.recovery', ROOT.parent/'.recovery']:
        for parent, directories, files in os.walk(recovery_root, followlinks=False, onerror=walk_error):
            for name in ['.', *directories, *files]:
                path = Path(parent) if name == '.' else Path(parent) / name
                recovery[str(path)] = node(path)
    return dict(directory_identities=identities, proof_outside_target=proof_files, recovery_nodes=recovery,
                active_parent_scope='Exact directory identity only; task12-owned contents are protected and are not required idle or byte-stable.')


def guard_target(expected_inode=None, allow_absent=False):
    assert TARGET == Path('/tmp/umpire-fn25415-final-9babff6c815b-47070/proof-bKkF3n/target')
    assert TARGET.parent == PROOF and TARGET.name == 'target' and TARGET.is_absolute()
    assert TARGET != PARENT and not TARGET.is_relative_to(ROOT) and not ROOT.is_relative_to(TARGET)
    assert not CAS.is_relative_to(TARGET) and not TARGET.is_relative_to(CAS)
    for path in [TARGET, *TARGET.parents]:
        assert not path.is_symlink(), 'target or ancestor became a symlink'
    assert str(TARGET.resolve()) == str(TARGET), 'target realpath escaped fixed literal'
    if not TARGET.exists():
        assert allow_absent, 'private target unexpectedly absent'
        return None
    info = TARGET.lstat()
    assert stat.S_ISDIR(info.st_mode), 'target is not a directory'
    identity = info.st_dev, info.st_ino
    if expected_inode is not None:
        assert identity == expected_inode, 'private target directory identity changed'
    return identity


def verify_remaining(target, expected, require_complete=False):
    """Read-only helper; no new files, changed original bytes/inodes/modes, or special nodes allowed."""
    by_path = {row['path']: row for row in expected}
    found = []
    for parent, directories, files in os.walk(target, followlinks=False, onerror=walk_error):
        for name in directories:
            assert stat.S_ISDIR((Path(parent)/name).lstat().st_mode), 'symlink/special directory in remainder'
        for name in files:
            path = Path(parent)/name
            before = path.lstat()
            assert stat.S_ISREG(before.st_mode), 'symlink/special file in remainder'
            relative = str(path.relative_to(target))
            assert relative in by_path, 'new file not admitted by classification: ' + relative
            row = by_path[relative]
            assert (before.st_dev, before.st_ino, before.st_size, oct(stat.S_IMODE(before.st_mode))) == (row['device'], row['inode'], row['size'], row['mode']), 'original member metadata changed: ' + relative
            assert digest(path) == row['sha256'], 'original member hash changed: ' + relative
            after = path.stat()
            assert (after.st_mtime_ns, after.st_ctime_ns, after.st_size, after.st_ino) == (before.st_mtime_ns, before.st_ctime_ns, before.st_size, before.st_ino), 'member changed during verification'
            assert row['disposition'] in ['primary-retained-exact-path', 'primary-retained-identical-content', 'supplemental-custody-required-preserve-until-verified', 'proposed-disposable-cargo-compiled-object']
            found.append(relative)
    assert not require_complete or set(found) == set(by_path), 'full private membership changed'
    return dict(files=len(found), new_files_allowed=False, symlinks_or_specials_allowed=False,
                original_hash_inode_size_mode_verified=True)


def load_module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def ownership(inspector, acknowledge_limits):
    value = inspector.ownership(TARGET)
    assert not value['matches'], 'private target has active owners/handles'
    if value['unreadable']:
        assert acknowledge_limits, 'explicit recorded-daemon visibility acknowledgement required'
    for row in value['unreadable']:
        assert KNOWN_FOREIGN.get(row['pid']) == (row.get('uid'), row.get('comm')), 'unrecognized unreadable process'
    value['limitation'] = 'No readable private handles matched. Listed foreign-daemon metadata remains unknown; this is not complete process visibility.'
    return value


def verify_custody(supplement):
    manifest = CAS/'manifests'/(SUPPLEMENTAL_SHA+'.json')
    result = supplement.verify(manifest, SUPPLEMENTAL_SHA)
    assert result['verified'] and result['supplemental_unique_members'] == 10708
    assert result['primary_verification']['manifest_sha256'] == PRIMARY_SHA
    assert result['primary_verification']['archive_members'] == 4801
    return result


def write_receipt(path, receipt, initial=False):
    if initial:
        with path.open('x') as stream:
            json.dump(receipt, stream, indent=2)
            stream.write('\n')
    else:
        temporary = HERE / (path.name + '.update-' + uuid.uuid4().hex)
        with temporary.open('x') as stream:
            json.dump(receipt, stream, indent=2)
            stream.write('\n')
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temporary, path)


def execute(options):
    run_id = uuid.uuid4().hex
    receipt_path = options.receipt or HERE/('cleanup-'+run_id+'.json')
    assert receipt_path.resolve().parent == HERE and receipt_path.name.startswith('cleanup-') and not receipt_path.is_symlink()
    receipt_path = receipt_path.resolve()
    receipt = dict(kind='root-authorized-exact-private-cache-cleanup', target=str(TARGET), started_utc=timestamp(),
                   script_sha256=digest(Path(__file__)), mutation_started=False, complete=False, commands=[],
                   protected_shared_target=str(PARENT), cleanup_scope='Fixed literal private target only',
                   free_before=free_space(), allocated_before=allocated(TARGET))
    write_receipt(receipt_path, receipt, initial=True)
    supplement = None
    protected = None
    target_inode = None
    failure = None
    try:
        assert options.expected_script_sha256 == receipt['script_sha256'], 'cleanup script differs from reviewed SHA'
        assert options.expected_primary_sha256 == PRIMARY_SHA
        assert options.expected_supplemental_sha256 == SUPPLEMENTAL_SHA
        assert options.expected_classification_sha256 == CLASSIFICATION_SHA
        assert options.expected_tools_sha256 == TOOLS_SHA
        assert digest(HERE/'tooling-manifest.json') == TOOLS_SHA
        tooling = json.loads((HERE/'tooling-manifest.json').read_text())
        for name, checksum in tooling['files'].items():
            assert digest(HERE/name) == checksum, 'sealed tooling input changed'
        assert digest(HERE/'classification.json') == CLASSIFICATION_SHA
        census = json.loads((HERE/'classification.json').read_text())
        target_inode = guard_target()
        supplement = load_module('cleanup_supplement', HERE/'supplemental.py')
        inspector = load_module('cleanup_inspector', HERE/'classify.py')
        collector, _ = supplement.helpers()
        supplement.validate_map(census, collector)
        receipt['custody_before'] = verify_custody(supplement)
        receipt['ownership_before'] = ownership(inspector, options.accept_recorded_daemon_visibility_limits)
        receipt['classification_before'] = verify_remaining(TARGET, census['all_files'], require_complete=True)
        protected = protected_snapshot()
        receipt['protected_before'] = protected
        receipt['cargo'] = dict(path=str(CARGO), sha256=digest(CARGO), realpath=str(CARGO.resolve()))
        assert CARGO.is_file() and not CARGO.is_symlink() and CARGO.resolve() == CARGO
        assert '1.99.1' in (PROOF/'framework/rust-toolchain.toml').read_text()
        tags = census['proposed_tagged_cargo_clean']
        expected_tags = {
            TARGET/'guest-cache/1d27b310f3351f024f36b327f8b0df4eef5c2099d9783729fe4dfaf0e87bf048/wasm32-wasip1',
            TARGET/'tests/runtime-capability-controls/target',
            TARGET/'tests/trybuild/aarch64-unknown-linux-gnu', TARGET/'wasm32-wasip1'}
        assert len(tags) == 4 and {Path(row['target']) for row in tags} == expected_tags
        environment = dict(os.environ)
        for key in ['BASH_ENV','CARGO_TARGET_DIR','CARGO_BUILD_TARGET','RUSTC_WRAPPER','RUSTC_WORKSPACE_WRAPPER','RUSTFLAGS','CARGO_ENCODED_RUSTFLAGS']:
            environment.pop(key, None)
        environment.update(MISE_AUTO_INSTALL='false', MISE_EXEC_AUTO_INSTALL='false')
        for index, tag in enumerate(tags):
            guard_target(target_inode)
            nested = Path(tag['target'])
            assert nested in expected_tags and nested.is_relative_to(TARGET) and nested.resolve() == nested and not nested.is_symlink()
            assert (nested/'CACHEDIR.TAG').is_file() and not (nested/'CACHEDIR.TAG').is_symlink()
            assert (nested/'CACHEDIR.TAG').read_bytes().startswith(TAG_SIGNATURE)
            assert digest(nested/'CACHEDIR.TAG') == tag['tag_sha256']
            command = [str(CARGO), 'clean', '--target-dir', str(nested)]
            assert command == tag['command']
            row = dict(command=command, cwd=str(PROOF/'framework'), started_utc=timestamp(),
                       ownership_before=ownership(inspector, options.accept_recorded_daemon_visibility_limits),
                       remaining_before=verify_remaining(TARGET, census['all_files']),
                       free_before=free_space(), allocated_before=allocated(TARGET), terminal=False)
            receipt['commands'].append(row)
            receipt['mutation_started'] = True
            write_receipt(receipt_path, receipt)
            stdout = HERE/('cleanup-'+run_id+'-cargo-'+str(index)+'.stdout.log')
            stderr = HERE/('cleanup-'+run_id+'-cargo-'+str(index)+'.stderr.log')
            row.update(stdout=str(stdout), stderr=str(stderr), pid=None)
            child = None
            try:
                with stdout.open('xb') as a, stderr.open('xb') as b:
                    child = subprocess.Popen(command, cwd=PROOF/'framework', env=environment, stdout=a, stderr=b, start_new_session=True)
                    row['pid'] = child.pid
                    write_receipt(receipt_path, receipt)
                    try:
                        child.wait(timeout=600)
                    except BaseException:
                        if child.poll() is None:
                            os.killpg(child.pid, signal.SIGTERM)
                            try:
                                child.wait(timeout=10)
                            except subprocess.TimeoutExpired:
                                os.killpg(child.pid, signal.SIGKILL)
                                child.wait()
                        raise
            finally:
                row.update(exit_code=None if child is None else child.returncode,
                           terminal=child is not None and child.poll() is not None, ended_utc=timestamp(),
                           stdout_sha256=digest(stdout) if stdout.exists() else None,
                           stderr_sha256=digest(stderr) if stderr.exists() else None,
                           free_after=free_space(), allocated_after=allocated(TARGET))
                write_receipt(receipt_path, receipt)
            assert row['exit_code'] == 0, 'official nested Cargo clean failed'
            row['remaining_after'] = verify_remaining(TARGET, census['all_files'])
            assert protected_snapshot() == protected, 'protected source/proof/CAS/parent identity changed'
            write_receipt(receipt_path, receipt)
        guard_target(target_inode)
        receipt['remainder_admission'] = dict(ownership=ownership(inspector, options.accept_recorded_daemon_visibility_limits),
            classification=verify_remaining(TARGET, census['all_files']), allocated_bytes=allocated(TARGET), free=free_space())
        assert protected_snapshot() == protected
        assert shutil.rmtree.avoids_symlink_attacks, 'fd-based symlink-resistant removal required'
        receipt['remainder_action'] = dict(method='shutil.rmtree', exact_literal=str(TARGET), wildcard=False, force_tag=False, started_utc=timestamp())
        write_receipt(receipt_path, receipt)
        shutil.rmtree(TARGET)
        receipt['remainder_action']['ended_utc'] = timestamp()
        assert not os.path.lexists(TARGET), 'private target remains'
    except BaseException as error:
        failure = error
        receipt['failure'] = dict(type=type(error).__name__, message=str(error), at=timestamp())
    finally:
        receipt['free_after'] = free_space()
        receipt['allocated_after'] = allocated(TARGET)
        receipt['private_target_absent'] = not os.path.lexists(TARGET)
        if receipt['mutation_started'] and supplement is not None:
            try:
                receipt['custody_after'] = verify_custody(supplement)
                receipt['ownership_after'] = ownership(inspector, options.accept_recorded_daemon_visibility_limits)
            except BaseException as error:
                receipt['post_mutation_custody_or_ownership_failure'] = dict(type=type(error).__name__, message=str(error))
                failure = failure or error
        if protected is not None:
            try:
                receipt['protected_after'] = protected_snapshot()
                assert receipt['protected_after'] == protected, 'protected paths changed'
                receipt['protected_exact_checks_passed'] = True
            except BaseException as error:
                receipt['protected_failure'] = dict(type=type(error).__name__, message=str(error))
                failure = failure or error
        receipt['complete'] = failure is None and receipt['private_target_absent'] and receipt.get('protected_exact_checks_passed', False) and receipt.get('custody_after', {}).get('verified', False)
        receipt['ended_utc'] = timestamp()
        receipt['free_space_limit'] = 'Source and /tmp are separate devices; concurrent task12 activity may affect observed free bytes. Parent contents are not asserted byte-stable.'
        write_receipt(receipt_path, receipt)
        print(json.dumps(dict(receipt=str(receipt_path), complete=receipt['complete'], mutation_started=receipt['mutation_started'], private_target_absent=receipt['private_target_absent']), indent=2), flush=True)
    return 0 if receipt['complete'] else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--execute', action='store_true')
    parser.add_argument('--expected-primary-sha256')
    parser.add_argument('--expected-supplemental-sha256')
    parser.add_argument('--expected-classification-sha256')
    parser.add_argument('--expected-tools-sha256')
    parser.add_argument('--expected-script-sha256')
    parser.add_argument('--accept-recorded-daemon-visibility-limits', action='store_true')
    parser.add_argument('--receipt', type=Path)
    options = parser.parse_args()
    if not options.execute:
        print(json.dumps(dict(prepared_only=True, mutation_started=False, target=str(TARGET))))
        return 0
    for key in ['expected_primary_sha256','expected_supplemental_sha256','expected_classification_sha256','expected_tools_sha256','expected_script_sha256']:
        assert re.fullmatch('[0-9a-f]{64}', getattr(options, key) or ''), key + ' required'
    return execute(options)


if __name__ == '__main__':
    sys.exit(main())
